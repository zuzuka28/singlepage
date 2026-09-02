import { assign, sendTo, setup } from 'xstate';
import type { OutlineDocument } from '../../../entities/outline';
import type { OpenSession, SaveState } from './model';
import { autosaveMachine, type AutosaveResult } from './autosave.machine';

export interface OpenSessionInput {
  session: OpenSession;
  save(session: OpenSession, document: OutlineDocument): Promise<OpenSession>;
  onSession?(session: OpenSession): void;
  onAutosave?(status: SaveState, dirty: boolean, epoch: number): void;
}

export interface OpenSessionContext {
  session: OpenSession;
  epoch: number;
  save: OpenSessionInput['save'];
  onSession?: OpenSessionInput['onSession'];
  onAutosave?: OpenSessionInput['onAutosave'];
}

export type OpenSessionEvent =
  | { type: 'EDIT'; document: OutlineDocument }
  | { type: 'PERSISTED'; revision: number; epoch: number }
  | { type: 'PASSWORD_CHANGED'; password: string; writeToken: string; revision: number; epoch: number }
  | { type: 'LINK_ROTATED'; session: OpenSession; epoch: number }
  | { type: 'FLUSH' }
  | { type: 'AUTOSAVE.SAVED'; result: AutosaveResult; epoch: number };

export const openSessionMachine = setup({
  types: {
    context: {} as OpenSessionContext,
    input: {} as OpenSessionInput,
    events: {} as OpenSessionEvent,
  },
  actors: { autosave: autosaveMachine },
  guards: {
    currentEpoch: ({ context, event }) => 'epoch' in event && event.epoch === context.epoch,
  },
  actions: {
    edit: assign(({ context, event }) => event.type === 'EDIT'
      ? { session: { ...context.session, document: event.document } }
      : {}),
    persisted: assign(({ context, event }) => event.type === 'PERSISTED'
      ? { session: { ...context.session, revision: event.revision } }
      : {}),
    passwordChanged: assign(({ context, event }) => event.type === 'PASSWORD_CHANGED'
      ? { session: { ...context.session, password: event.password, writeToken: event.writeToken, revision: event.revision } }
      : {}),
    linkRotated: assign(({ event }) => event.type === 'LINK_ROTATED'
      ? { session: event.session, epoch: event.epoch }
      : {}),
    autosaveSaved: assign(({ context, event }) => {
      if (event.type !== 'AUTOSAVE.SAVED' || !event.result.session) return {};
      const session = { ...event.result.session, document: context.session.document };
      context.onSession?.(session);
      return { session };
    }),
    queueAutosave: sendTo('autosave', ({ context, event }) => ({
      type: 'CHANGE' as const,
      snapshot: event.type === 'EDIT' ? event.document : context.session.document,
      baseSession: event.type === 'EDIT' ? { ...context.session, document: event.document } : context.session,
    })),
    flushAutosave: sendTo('autosave', { type: 'FLUSH' }),
  },
}).createMachine({
  id: 'open-session',
  initial: 'active',
  context: ({ input }) => ({
    session: input.session,
    epoch: input.session.epoch,
    save: input.save,
    onSession: input.onSession,
    onAutosave: input.onAutosave,
  }),
  states: {
    active: {
      invoke: {
        id: 'autosave',
        src: 'autosave',
        input: ({ context, self }) => ({
          epoch: context.epoch,
          save: async (document: OutlineDocument, _generation: number, _epoch: number, baseSession?: OpenSession) => {
            const session = await context.save(baseSession ?? context.session, document);
            return { revision: session.revision, session };
          },
          onSaved: (result: AutosaveResult, _generation: number, epoch: number) => {
            self.send({ type: 'AUTOSAVE.SAVED', result, epoch });
          },
          onStatus: context.onAutosave,
        }),
      },
      on: {
        EDIT: { actions: ['edit', 'queueAutosave'] },
        FLUSH: { actions: 'flushAutosave' },
        PERSISTED: { guard: 'currentEpoch', actions: 'persisted' },
        PASSWORD_CHANGED: { guard: 'currentEpoch', actions: 'passwordChanged' },
        LINK_ROTATED: { guard: 'currentEpoch', actions: 'linkRotated' },
        'AUTOSAVE.SAVED': { guard: 'currentEpoch', actions: 'autosaveSaved' },
      },
    },
  },
});
