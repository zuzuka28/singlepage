import { useMachine, useSelector } from '@xstate/svelte';
import { waitFor } from 'xstate';
import { appMachine, selectAppView, selectAutosaveDirty } from './app.machine';
import { saveSession } from '../../features/page-session/logic/use-cases';
import type { ApplicationRuntime } from './runtime';

export function useApplicationActor(runtime: ApplicationRuntime) {
  const actor = useMachine(appMachine, {
    input: {
      repository: runtime.repository,
      locator: runtime.navigation.locator,
      save: (session, document) => saveSession(
        { repository: runtime.repository, crypto: runtime.crypto },
        session,
        document,
      ),
    },
  });

  async function flush() {
    if (!selectAutosaveDirty(actor.actorRef.getSnapshot())) return;
    actor.send({ type: 'FLUSH' });
    await waitFor(actor.actorRef, (snapshot) => {
      if (!selectAutosaveDirty(snapshot)) return true;
      return ['error', 'conflict', 'revoked'].includes(snapshot.context.saveState);
    });
  }

  return {
    send: actor.send,
    view: useSelector(actor.actorRef, selectAppView),
    getView: () => selectAppView(actor.actorRef.getSnapshot()),
    currentSession: () => actor.actorRef.getSnapshot().context.session,
    nextEpoch: () => actor.actorRef.getSnapshot().context.epoch + 1,
    isDirty: () => selectAutosaveDirty(actor.actorRef.getSnapshot()),
    flush,
  };
}

export type ApplicationActorBinding = ReturnType<typeof useApplicationActor>;
