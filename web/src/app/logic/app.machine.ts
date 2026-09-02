import { assign, fromPromise, sendTo, setup, type SnapshotFrom } from 'xstate';
import { createDocument, type OutlineDocument } from '../../entities/outline';
import type { AppViewState, OpenSession, SaveState, SessionIdentity } from '../../features/page-session/logic/model';
import { identityFor } from '../../features/page-session/logic/use-cases';
import type { PageRepository, RemotePage } from '../../features/page-session/logic/ports';
import { openSessionMachine } from '../../features/page-session/logic/open-session.machine';

export interface AppMachineInput {
  repository: PageRepository;
  locator?: string;
  save(session: OpenSession, document: OutlineDocument): Promise<OpenSession>;
}

export interface AppContext {
  repository: PageRepository;
  identity: SessionIdentity | null;
  remotePage: RemotePage | null;
  session: OpenSession | null;
  epoch: number;
  saveState: SaveState;
  autosaveDirty: boolean;
  save: AppMachineInput['save'];
}

export type AppEvent =
  | { type: 'NAVIGATE'; locator: string }
  | { type: 'OPENED'; session: OpenSession }
  | { type: 'CREATED'; session: OpenSession }
  | { type: 'EDIT'; document: OutlineDocument }
  | { type: 'FLUSH' }
  | { type: 'SESSION.SAVED'; session: OpenSession }
  | { type: 'AUTOSAVE.STATUS'; status: SaveState; dirty: boolean; epoch: number }
  | { type: 'SAVE.FAILED'; status: Extract<SaveState, 'conflict' | 'revoked'> }
  | { type: 'RESET' }
  | { type: 'xstate.done.actor.load'; output: LoadOutput }
  | { type: 'xstate.error.actor.load'; error: unknown };

interface LoadInput {
  repository: PageRepository;
  identity: SessionIdentity;
  epoch: number;
}

interface LoadOutput {
  remotePage: RemotePage;
  epoch: number;
}

export const appMachine = setup({
  types: {
    context: {} as AppContext,
    input: {} as AppMachineInput,
    events: {} as AppEvent,
  },
  actors: {
    loadPage: fromPromise<LoadOutput, LoadInput>(async ({ input, signal }) => ({
      remotePage: await input.repository.getPage(input.identity.pageId, signal),
      epoch: input.epoch,
    })),
    openSession: openSessionMachine,
  },
  guards: {
    hasIdentity: ({ context }) => context.identity !== null,
    currentLoad: ({ context, event }) =>
      event.type === 'xstate.done.actor.load' && event.output.epoch === context.epoch,
  },
  actions: {
    navigate: assign(({ context, event }) => event.type === 'NAVIGATE'
      ? {
          identity: parseLocator(event.locator),
          remotePage: null,
          session: null,
          epoch: context.epoch + 1,
          saveState: 'saved' as const,
          autosaveDirty: false,
        }
      : {}),
    loaded: assign(({ event }) => event.type === 'xstate.done.actor.load'
      ? { remotePage: event.output.remotePage }
      : {}),
    opened: assign(({ event }) => event.type === 'OPENED' || event.type === 'CREATED'
      ? {
          session: event.session,
          identity: event.session,
          remotePage: event.session.remotePage,
          epoch: event.session.epoch,
          saveState: 'saved' as const,
          autosaveDirty: false,
        }
      : {}),
    reset: assign(({ context }) => ({
      identity: null,
      remotePage: null,
      session: null,
      epoch: context.epoch + 1,
      saveState: 'saved' as const,
      autosaveDirty: false,
    })),
    sessionSaved: assign(({ context, event }) => event.type === 'SESSION.SAVED' && event.session.epoch === context.epoch
      ? { session: event.session, remotePage: event.session.remotePage }
      : {}),
    editSession: assign(({ context, event }) => event.type === 'EDIT' && context.session
      ? { session: { ...context.session, document: event.document }, autosaveDirty: true, saveState: 'saving' as const }
      : {}),
    autosaveStatus: assign(({ context, event }) => event.type === 'AUTOSAVE.STATUS' && event.epoch === context.epoch
      ? { saveState: event.status, autosaveDirty: event.dirty }
      : {}),
    saveFailed: assign(({ event }) => event.type === 'SAVE.FAILED'
      ? { saveState: event.status, autosaveDirty: true }
      : {}),
    forwardEdit: sendTo('openSession', ({ event }) => ({
      type: 'EDIT' as const,
      document: event.type === 'EDIT' ? event.document : createDocument(),
    })),
    flushAutosave: sendTo('openSession', { type: 'FLUSH' }),
  },
}).createMachine({
  id: 'app',
  initial: 'route',
  context: ({ input }) => ({
    repository: input.repository,
    identity: input.locator ? parseLocator(input.locator) : null,
    remotePage: null,
    session: null,
    epoch: 0,
    saveState: 'saved',
    autosaveDirty: false,
    save: input.save,
  }),
  on: {
    NAVIGATE: { target: '.loading', actions: 'navigate', reenter: true },
    RESET: { target: '.start', actions: 'reset' },
  },
  states: {
    route: {
      always: [
        { target: 'loading', guard: 'hasIdentity' },
        { target: 'start' },
      ],
    },
    start: { on: { CREATED: { target: 'open', actions: 'opened' } } },
    loading: {
      always: { target: 'start', guard: ({ context }) => context.identity === null },
      invoke: {
        id: 'load',
        src: 'loadPage',
        input: ({ context }) => ({ repository: context.repository, identity: context.identity!, epoch: context.epoch }),
        onDone: { target: 'locked', guard: 'currentLoad', actions: 'loaded' },
        onError: { target: 'missing' },
      },
    },
    locked: { on: { OPENED: { target: 'open', actions: 'opened' } } },
    open: {
      invoke: {
        id: 'openSession',
        src: 'openSession',
        input: ({ context, self }) => ({
          session: context.session!,
          save: context.save,
          onSession: (session: OpenSession) => self.send({ type: 'SESSION.SAVED', session }),
          onAutosave: (status: SaveState, dirty: boolean, epoch: number) => {
            self.send({ type: 'AUTOSAVE.STATUS', status, dirty, epoch });
          },
        }),
      },
      on: {
        EDIT: { actions: ['editSession', 'forwardEdit'] },
        FLUSH: { actions: 'flushAutosave' },
        'SESSION.SAVED': { actions: 'sessionSaved' },
        'AUTOSAVE.STATUS': { actions: 'autosaveStatus' },
        'SAVE.FAILED': { actions: 'saveFailed' },
        CREATED: { target: 'open', actions: 'opened', reenter: true },
      },
    },
    missing: {},
  },
});

export type AppSnapshot = SnapshotFrom<typeof appMachine>;

export function selectAppView(snapshot: AppSnapshot): AppViewState {
  const { context } = snapshot;
  if (snapshot.matches('loading') && context.identity) return { kind: 'loading', identity: context.identity };
  if (snapshot.matches('locked') && context.identity && context.remotePage) {
    return { kind: 'locked', identity: context.identity, remotePage: context.remotePage };
  }
  if (snapshot.matches('missing') && context.identity) return { kind: 'missing', identity: context.identity };
  if (snapshot.matches('open') && context.session) {
    return { kind: 'open', session: context.session, saveState: context.saveState };
  }
  return { kind: 'start' };
}

export function selectAutosaveDirty(snapshot: AppSnapshot): boolean {
  return snapshot.matches('open') && snapshot.context.autosaveDirty;
}

export function parseLocator(locator: string): SessionIdentity | null {
  const url = new URL(locator, 'https://singlepage.invalid');
  const match = /^\/p\/([A-Za-z0-9_-]{16,128})\/?$/.exec(url.pathname);
  if (!match) return null;
  return identityFor(match[1], url.hash.slice(1));
}

export function emptyOpenSession(identity: SessionIdentity, remotePage: RemotePage, epoch = 1): OpenSession {
  return {
    ...identity,
    document: createDocument(),
    password: '',
    writeToken: '',
    revision: remotePage.revision,
    remotePage,
    epoch,
  };
}
