import { assign, fromPromise, setup } from 'xstate';
import type { OutlineDocument } from '../../../entities/outline';
import { toApplicationError } from './errors';
import type { OpenSession, SaveState } from './model';

export interface AutosaveResult {
  revision: number;
  session?: OpenSession;
}

export interface AutosaveInput {
  epoch: number;
  save(snapshot: OutlineDocument, generation: number, epoch: number, baseSession?: OpenSession): Promise<AutosaveResult>;
  onSaved?(result: AutosaveResult, generation: number, epoch: number): void;
  onStatus?(status: SaveState, dirty: boolean, epoch: number): void;
  debounceMs?: number;
  retryMs?: number;
}

export interface AutosaveContext extends AutosaveInput {
  dirtyGeneration: number;
  savedGeneration: number;
  pending: OutlineDocument | null;
  activeGeneration: number;
  activeEpoch: number;
  revision: number | null;
  pendingBaseSession: OpenSession | null;
}

export type AutosaveEvent =
  | { type: 'CHANGE'; snapshot: OutlineDocument; baseSession?: OpenSession }
  | { type: 'FLUSH' }
  | { type: 'RESET'; epoch: number }
  | { type: 'xstate.done.actor.persist'; output: SaveActorOutput }
  | { type: 'xstate.error.actor.persist'; error: unknown };

interface SaveActorInput {
  snapshot: OutlineDocument;
  generation: number;
  epoch: number;
  save: AutosaveInput['save'];
  baseSession?: OpenSession;
}

interface SaveActorOutput extends AutosaveResult {
  generation: number;
  epoch: number;
}

export const autosaveMachine = setup({
  types: {
    context: {} as AutosaveContext,
    input: {} as AutosaveInput,
    events: {} as AutosaveEvent,
  },
  actors: {
    persistSnapshot: fromPromise<SaveActorOutput, SaveActorInput>(async ({ input }) => ({
      ...(await input.save(input.snapshot, input.generation, input.epoch, input.baseSession)),
      generation: input.generation,
      epoch: input.epoch,
    })),
  },
  delays: {
    debounce: ({ context }) => context.debounceMs ?? 650,
    retry: ({ context }) => context.retryMs ?? 2_500,
  },
  guards: {
    hasPending: ({ context }) => context.pending !== null && context.dirtyGeneration > context.savedGeneration,
    newerPending: ({ context, event }) =>
      event.type === 'xstate.done.actor.persist' && context.dirtyGeneration > event.output.generation,
    conflict: ({ event }) => event.type === 'xstate.error.actor.persist' && toApplicationError(event.error).code === 'conflict',
    revoked: ({ event }) => {
      if (event.type !== 'xstate.error.actor.persist') return false;
      return ['unauthorized', 'forbidden', 'not-found'].includes(toApplicationError(event.error).code);
    },
  },
  actions: {
    markDirty: assign(({ context, event }) => event.type === 'CHANGE'
      ? {
          pending: event.snapshot,
          pendingBaseSession: event.baseSession ?? null,
          dirtyGeneration: context.dirtyGeneration + 1,
        }
      : {}),
    beginSave: assign(({ context }) => ({
      activeGeneration: context.dirtyGeneration,
      activeEpoch: context.epoch,
    })),
    acceptSave: assign(({ context, event }) => {
      if (event.type !== 'xstate.done.actor.persist' || event.output.epoch !== context.epoch) return {};
      context.onSaved?.(event.output, event.output.generation, event.output.epoch);
      return {
        savedGeneration: event.output.generation,
        revision: event.output.revision,
        ...(event.output.session && context.dirtyGeneration > event.output.generation
          ? {
              pendingBaseSession: {
                ...event.output.session,
                document: context.pending ?? event.output.session.document,
              },
            }
          : {}),
      };
    }),
    reset: assign(({ event }) => event.type === 'RESET' ? {
      epoch: event.epoch,
      dirtyGeneration: 0,
      savedGeneration: 0,
      pending: null,
      activeGeneration: 0,
      activeEpoch: event.epoch,
      revision: null,
      pendingBaseSession: null,
    } : {}),
    reportSaved: ({ context }) => context.onStatus?.('saved', false, context.epoch),
    reportSaving: ({ context }) => context.onStatus?.('saving', true, context.epoch),
    reportError: ({ context }) => context.onStatus?.('error', true, context.epoch),
    reportConflict: ({ context }) => context.onStatus?.('conflict', true, context.epoch),
    reportRevoked: ({ context }) => context.onStatus?.('revoked', true, context.epoch),
  },
}).createMachine({
  id: 'autosave',
  initial: 'clean',
  context: ({ input }) => ({
    ...input,
    dirtyGeneration: 0,
    savedGeneration: 0,
    pending: null,
    activeGeneration: 0,
    activeEpoch: input.epoch,
    revision: null,
    pendingBaseSession: null,
  }),
  on: {
    RESET: { target: '.clean', actions: 'reset' },
  },
  states: {
    clean: {
      entry: 'reportSaved',
      on: { CHANGE: { target: 'debouncing', actions: 'markDirty' } },
    },
    debouncing: {
      entry: 'reportSaving',
      on: {
        CHANGE: { actions: 'markDirty', reenter: true },
        FLUSH: { target: 'saving', guard: 'hasPending' },
      },
      after: { debounce: { target: 'saving', guard: 'hasPending' } },
    },
    saving: {
      entry: ['beginSave', 'reportSaving'],
      on: { CHANGE: { actions: 'markDirty' } },
      invoke: {
        id: 'persist',
        src: 'persistSnapshot',
        input: ({ context }) => ({
          snapshot: context.pending!,
          generation: context.activeGeneration,
          epoch: context.activeEpoch,
          save: context.save,
          baseSession: context.pendingBaseSession ?? undefined,
        }),
        onDone: [
          { target: 'debouncing', guard: 'newerPending', actions: 'acceptSave' },
          { target: 'clean', actions: 'acceptSave' },
        ],
        onError: [
          { target: 'conflict', guard: 'conflict' },
          { target: 'revoked', guard: 'revoked' },
          { target: 'retrying' },
        ],
      },
    },
    retrying: {
      entry: 'reportError',
      on: {
        CHANGE: { actions: 'markDirty' },
        FLUSH: { target: 'saving' },
      },
      after: { retry: 'saving' },
    },
    conflict: { entry: 'reportConflict' },
    revoked: { entry: 'reportRevoked' },
  },
});

export function autosaveStatus(snapshot: { value: unknown }): 'saved' | 'saving' | 'error' | 'conflict' | 'revoked' {
  if (snapshot.value === 'clean') return 'saved';
  if (snapshot.value === 'conflict') return 'conflict';
  if (snapshot.value === 'revoked') return 'revoked';
  if (snapshot.value === 'retrying') return 'error';
  return 'saving';
}

export function isAutosaveDirty(snapshot: { value: unknown; context: AutosaveContext }): boolean {
  return snapshot.context.dirtyGeneration !== snapshot.context.savedGeneration || snapshot.value === 'saving';
}
