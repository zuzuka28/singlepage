import { createActor, waitFor } from 'xstate';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createDocument, insertRoot, updateBlock } from '../../../entities/outline';
import { ApplicationError } from './errors';
import { autosaveMachine, isAutosaveDirty } from './autosave.machine';

describe('autosave actor', () => {
  afterEach(() => vi.useRealTimers());

  it('debounces, serializes and coalesces intermediate generations', async () => {
    vi.useFakeTimers();
    const base = insertRoot(createDocument(), '', 0, () => 'root').document;
    const calls: string[] = [];
    const completions: Array<(value: { revision: number }) => void> = [];
    const actor = createActor(autosaveMachine, {
      input: {
        epoch: 1,
        debounceMs: 10,
        save: (document) => new Promise((resolve) => {
          calls.push(document.blocks.root.text);
          completions.push(resolve);
        }),
      },
    }).start();

    actor.send({ type: 'CHANGE', snapshot: updateBlock(base, 'root', { text: 'one' }) });
    expect(isAutosaveDirty(actor.getSnapshot())).toBe(true);
    await vi.advanceTimersByTimeAsync(10);
    expect(calls).toEqual(['one']);
    actor.send({ type: 'CHANGE', snapshot: updateBlock(base, 'root', { text: 'two' }) });
    actor.send({ type: 'CHANGE', snapshot: updateBlock(base, 'root', { text: 'three' }) });
    completions[0]({ revision: 2 });
    await vi.advanceTimersByTimeAsync(10);
    expect(calls).toEqual(['one', 'three']);
    completions[1]({ revision: 3 });
    await vi.runAllTimersAsync();
    await waitFor(actor, (snapshot) => snapshot.matches('clean'));
    expect(actor.getSnapshot().context.savedGeneration).toBe(3);
    actor.stop();
  });

  it.each([
    ['conflict', 'conflict'],
    ['unauthorized', 'revoked'],
    ['not-found', 'revoked'],
  ] as const)('maps %s into a terminal %s state', async (code, state) => {
    vi.useFakeTimers();
    const document = insertRoot(createDocument(), '', 0, () => 'root').document;
    const actor = createActor(autosaveMachine, {
      input: {
        epoch: 1,
        debounceMs: 0,
        save: async () => { throw new ApplicationError(code, code); },
      },
    }).start();
    actor.send({ type: 'CHANGE', snapshot: document });
    await vi.runAllTimersAsync();
    await waitFor(actor, (snapshot) => snapshot.matches(state));
    expect(isAutosaveDirty(actor.getSnapshot())).toBe(true);
    actor.stop();
  });
});
