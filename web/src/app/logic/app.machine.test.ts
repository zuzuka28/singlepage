import { createActor, waitFor } from 'xstate';
import { describe, expect, it, vi } from 'vitest';
import { createDocument, insertRoot, updateBlock } from '../../entities/outline';
import { appMachine, selectAppView, selectAutosaveDirty } from './app.machine';
import type { OpenSession } from '../../features/page-session/logic/model';
import type { PageRepository, RemotePage } from '../../features/page-session/logic/ports';

describe('application lifecycle machine', () => {
  it('selects start and locked states', async () => {
    const page: RemotePage = { revision: 1, salt: new Uint8Array([1]), ciphertext: new Uint8Array([2]) };
    const repository = repositoryWith(async () => page);
    const actor = createActor(appMachine, { input: { repository, save: async (session) => session } }).start();
    expect(selectAppView(actor.getSnapshot())).toEqual({ kind: 'start' });
    actor.send({ type: 'NAVIGATE', locator: '/p/abcdefghijklmnop#secret' });
    const locked = await waitFor(actor, (snapshot) => snapshot.matches('locked'));
    expect(selectAppView(locked)).toEqual({
      kind: 'locked',
      identity: { pageId: 'abcdefghijklmnop', urlSecret: 'secret', locator: '/p/abcdefghijklmnop#secret' },
      remotePage: page,
    });
    actor.stop();
  });

  it('selects the missing state when loading a page fails', async () => {
    const repository = repositoryWith(async () => { throw new Error('not found'); });
    const actor = createActor(appMachine, { input: { repository, save: async (session) => session } }).start();

    actor.send({ type: 'NAVIGATE', locator: '/p/abcdefghijklmnop#secret' });

    const missing = await waitFor(actor, (snapshot) => snapshot.matches('missing'));
    expect(selectAppView(missing)).toEqual({
      kind: 'missing',
      identity: { pageId: 'abcdefghijklmnop', urlSecret: 'secret', locator: '/p/abcdefghijklmnop#secret' },
    });
    actor.stop();
  });

  it('aborts stale loads so A cannot replace B', async () => {
    const pending = new Map<string, { resolve: (page: RemotePage) => void; signal?: AbortSignal }>();
    const repository = repositoryWith((id, signal) => new Promise((resolve) => pending.set(id, { resolve, signal })));
    const actor = createActor(appMachine, { input: { repository, save: async (session) => session } }).start();
    actor.send({ type: 'NAVIGATE', locator: '/p/aaaaaaaaaaaaaaaa#a' });
    await vi.waitFor(() => expect(pending.has('aaaaaaaaaaaaaaaa')).toBe(true));
    actor.send({ type: 'NAVIGATE', locator: '/p/bbbbbbbbbbbbbbbb#b' });
    await vi.waitFor(() => expect(pending.has('bbbbbbbbbbbbbbbb')).toBe(true));
    expect(pending.get('aaaaaaaaaaaaaaaa')?.signal?.aborted).toBe(true);
    pending.get('aaaaaaaaaaaaaaaa')!.resolve({ revision: 1, salt: new Uint8Array(), ciphertext: new Uint8Array() });
    pending.get('bbbbbbbbbbbbbbbb')!.resolve({ revision: 2, salt: new Uint8Array(), ciphertext: new Uint8Array() });
    const locked = await waitFor(actor, (snapshot) => snapshot.matches('locked'));
    expect(selectAppView(locked).kind).toBe('locked');
    expect(locked.context.identity?.pageId).toBe('bbbbbbbbbbbbbbbb');
    actor.stop();
  });

  it('runs the root -> open-session -> autosave hierarchy with one session owner', async () => {
    const repository = repositoryWith(vi.fn());
    const document = insertRoot(createDocument(), '', 0, () => 'root').document;
    const session: OpenSession = {
      pageId: 'abcdefghijklmnop',
      urlSecret: 'secret',
      locator: '/p/abcdefghijklmnop#secret',
      document,
      password: 'password1',
      writeToken: 'token',
      revision: 1,
      remotePage: { revision: 1, salt: new Uint8Array([1]), ciphertext: new Uint8Array([2]) },
      epoch: 1,
    };
    const save = vi.fn(async (current: OpenSession, nextDocument) => ({
      ...current,
      document: nextDocument,
      revision: 2,
      remotePage: { ...current.remotePage, revision: 2 },
    }));
    const actor = createActor(appMachine, { input: { repository, save } }).start();
    actor.send({ type: 'CREATED', session });
    actor.send({ type: 'EDIT', document: updateBlock(document, 'root', { text: 'changed' }) });
    expect(selectAppView(actor.getSnapshot())).toEqual(expect.objectContaining({
      kind: 'open',
      session: expect.objectContaining({ document: expect.objectContaining({
        blocks: expect.objectContaining({ root: expect.objectContaining({ text: 'changed' }) }),
      }) }),
      saveState: 'saving',
    }));
    expect(selectAutosaveDirty(actor.getSnapshot())).toBe(true);
    actor.send({ type: 'FLUSH' });
    const saved = await waitFor(actor, (snapshot) => snapshot.context.session?.revision === 2, { timeout: 2_000 });
    expect(save).toHaveBeenCalledTimes(1);
    expect(saved.context.session?.document.blocks.root.text).toBe('changed');
    const clean = await waitFor(actor, (snapshot) => !selectAutosaveDirty(snapshot), { timeout: 2_000 });
    expect(selectAppView(clean)).toEqual(expect.objectContaining({ kind: 'open', saveState: 'saved' }));
    actor.stop();
  });

  it('rebases edits made during an active save onto the saved revision', async () => {
    const repository = repositoryWith(vi.fn());
    const document = insertRoot(createDocument(), '', 0, () => 'root').document;
    const session: OpenSession = {
      pageId: 'abcdefghijklmnop',
      urlSecret: 'secret',
      locator: '/p/abcdefghijklmnop#secret',
      document,
      password: 'password1',
      writeToken: 'token',
      revision: 1,
      remotePage: { revision: 1, salt: new Uint8Array([1]), ciphertext: new Uint8Array([2]) },
      epoch: 1,
    };
    const completions: Array<(session: OpenSession) => void> = [];
    const save = vi.fn((current: OpenSession, nextDocument) => new Promise<OpenSession>((resolve) => {
      completions.push((savedSession) => resolve({ ...savedSession, document: nextDocument }));
    }));
    const actor = createActor(appMachine, { input: { repository, save } }).start();
    const firstEdit = updateBlock(document, 'root', { text: 'one' });
    const secondEdit = updateBlock(document, 'root', { text: 'two' });

    actor.send({ type: 'CREATED', session });
    actor.send({ type: 'EDIT', document: firstEdit });
    actor.send({ type: 'FLUSH' });
    await vi.waitFor(() => expect(save).toHaveBeenCalledTimes(1));

    actor.send({ type: 'EDIT', document: secondEdit });
    completions[0]({
      ...session,
      document: firstEdit,
      revision: 2,
      remotePage: { ...session.remotePage, revision: 2 },
    });

    await vi.waitFor(() => expect(save).toHaveBeenCalledTimes(2), { timeout: 2_000 });
    expect(save.mock.calls[1][0].revision).toBe(2);
    expect(save.mock.calls[1][1].blocks.root.text).toBe('two');
    expect(actor.getSnapshot().context.session?.document.blocks.root.text).toBe('two');

    completions[1]({
      ...session,
      document: secondEdit,
      revision: 3,
      remotePage: { ...session.remotePage, revision: 3 },
    });
    const clean = await waitFor(actor, (snapshot) => !selectAutosaveDirty(snapshot), { timeout: 2_000 });
    expect(clean.context.session?.revision).toBe(3);
    expect(clean.context.session?.document.blocks.root.text).toBe('two');
    actor.stop();
  });
});

function repositoryWith(getPage: PageRepository['getPage']): PageRepository {
  return {
    createPage: vi.fn(),
    getPage,
    updatePage: vi.fn(),
    rotatePage: vi.fn(),
  };
}
