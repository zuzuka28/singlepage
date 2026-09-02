import { describe, expect, it, vi } from 'vitest';
import { createDocument } from '../../../entities/outline';
import { changeSessionPassword, createSession, rotateSessionLink, saveSession, unlockSession } from './use-cases';
import type { PageRepository, SecretSource, VaultCrypto } from './ports';

function fixtures() {
  const repository: PageRepository = {
    createPage: vi.fn(async () => ({ revision: 1 })),
    getPage: vi.fn(),
    updatePage: vi.fn(async () => ({ revision: 2 })),
    rotatePage: vi.fn(async () => ({ revision: 3 })),
  };
  const crypto: VaultCrypto = {
    encrypt: vi.fn(async () => ({ salt: new Uint8Array([3]), ciphertext: new Uint8Array([4]) })),
    decrypt: vi.fn(async () => ({ document: createDocument(), writeToken: 'decrypted-token' })),
  };
  const values = {
    pageId: ['pageid0000000001', 'pageid0000000002'],
    urlSecret: ['url-secret-1', 'url-secret-2'],
    writeToken: ['write-token-1', 'write-token-2', 'write-token-3'],
  };
  const secrets: SecretSource = {
    pageId: () => values.pageId.shift()!,
    urlSecret: () => values.urlSecret.shift()!,
    writeToken: () => values.writeToken.shift()!,
    salt: () => new Uint8Array([1]),
  };
  return { repository, crypto, secrets };
}

describe('page session use cases', () => {
  it('creates an encrypted page and passes the locator atomically without leaking credentials', async () => {
    const services = fixtures();
    const session = await createSession(services, { password: 'private-password' });

    expect(services.repository.createPage).toHaveBeenCalledWith(
      {
        id: 'pageid0000000001',
        salt: new Uint8Array([3]),
        ciphertext: new Uint8Array([4]),
        writeToken: 'write-token-1',
      },
      '/p/pageid0000000001#url-secret-1',
    );
    expect(JSON.stringify(vi.mocked(services.repository.createPage).mock.calls[0])).not.toContain('private-password');
    expect(session.password).toBe('private-password');
  });

  it('unlocks through crypto and remembers only the locator', async () => {
    const services = fixtures();
    const history = { available: true, error: '', restore: vi.fn(), remember: vi.fn(async () => true), list: vi.fn() };
    const session = await unlockSession(
      { crypto: services.crypto, history },
      { pageId: 'pageid0000000001', urlSecret: 'url-secret-1', locator: '/p/pageid0000000001#url-secret-1' },
      { revision: 7, salt: new Uint8Array([1]), ciphertext: new Uint8Array([2]) },
      'private-password',
    );
    expect(history.remember).toHaveBeenCalledWith('/p/pageid0000000001#url-secret-1');
    expect(session.writeToken).toBe('decrypted-token');
  });

  it('saves the encrypted document against the current revision and advances the session', async () => {
    const services = fixtures();
    const session = await createSession(services, { password: 'private-password' });
    const document = createDocument();

    const saved = await saveSession(services, session, document);

    expect(services.repository.updatePage).toHaveBeenCalledWith(
      session.pageId,
      session.writeToken,
      {
        expectedRevision: session.revision,
        ciphertext: new Uint8Array([4]),
      },
    );
    expect(saved).toEqual(expect.objectContaining({ document, revision: 2 }));
  });

  it('changes the password and replaces the write capability atomically', async () => {
    const services = fixtures();
    const original = await createSession(services, { password: 'old-password' });
    const changed = await changeSessionPassword(services, original, original.document, 'new-password');
    expect(services.repository.updatePage).toHaveBeenCalledWith(
      original.pageId,
      original.writeToken,
      expect.objectContaining({ expectedRevision: 1, newWriteToken: 'write-token-2' }),
    );
    expect(changed).toEqual(expect.objectContaining({ password: 'new-password', writeToken: 'write-token-2' }));
  });

  it('rotates the page identity and access link atomically', async () => {
    const services = fixtures();
    const original = await createSession(services, { password: 'old-password' });
    const changed = await changeSessionPassword(services, original, original.document, 'new-password');
    const rotated = await rotateSessionLink(services, changed, changed.document);
    expect(services.repository.rotatePage).toHaveBeenCalledWith(
      original.pageId,
      'write-token-2',
      expect.objectContaining({ newId: 'pageid0000000002', newWriteToken: 'write-token-3' }),
      '/p/pageid0000000002#url-secret-2',
    );
    expect(rotated.revision).toBe(3);
  });
});
