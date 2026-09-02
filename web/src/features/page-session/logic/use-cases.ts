import { createDocument, insertRoot, type OutlineDocument } from '../../../entities/outline';
import type { OpenSession, SessionIdentity } from './model';
import type { PageHistory, PageRepository, RemotePage, SecretSource, VaultCrypto } from './ports';

export interface SessionServices {
  repository: PageRepository;
  crypto: VaultCrypto;
  secrets: SecretSource;
  history?: PageHistory;
}

export interface CreateSessionInput {
  password: string;
  document?: OutlineDocument;
  epoch?: number;
}

export async function createSession(services: SessionServices, input: CreateSessionInput): Promise<OpenSession> {
  const pageId = services.secrets.pageId();
  const urlSecret = services.secrets.urlSecret();
  const writeToken = services.secrets.writeToken();
  const identity = identityFor(pageId, urlSecret);
  const document = input.document ?? insertRoot(createDocument()).document;
  const encrypted = await services.crypto.encrypt({ document, writeToken }, input.password, urlSecret);
  const created = await services.repository.createPage(
    { id: pageId, salt: encrypted.salt, ciphertext: encrypted.ciphertext, writeToken },
    identity.locator,
  );
  return {
    ...identity,
    document,
    password: input.password,
    writeToken,
    revision: created.revision,
    remotePage: { ...encrypted, revision: created.revision },
    epoch: input.epoch ?? 1,
  };
}

export async function unlockSession(
  services: Pick<SessionServices, 'crypto' | 'history'>,
  identity: SessionIdentity,
  remotePage: RemotePage,
  password: string,
  epoch = 1,
): Promise<OpenSession> {
  const vault = await services.crypto.decrypt(remotePage, password, identity.urlSecret);
  if (!vault?.document || !vault.writeToken) throw new Error('Invalid encrypted document');
  if (services.history?.available && !(await services.history.remember(identity.locator))) {
    throw new Error(services.history.error || 'Unable to remember this page');
  }
  return {
    ...identity,
    document: vault.document,
    password,
    writeToken: vault.writeToken,
    revision: remotePage.revision,
    remotePage,
    epoch,
  };
}

export async function saveSession(
  services: Pick<SessionServices, 'repository' | 'crypto'>,
  session: OpenSession,
  document: OutlineDocument,
): Promise<OpenSession> {
  const encrypted = await services.crypto.encrypt(
    { document, writeToken: session.writeToken },
    session.password,
    session.urlSecret,
    session.remotePage.salt,
  );
  const updated = await services.repository.updatePage(session.pageId, session.writeToken, {
    expectedRevision: session.revision,
    ciphertext: encrypted.ciphertext,
  });
  return {
    ...session,
    document,
    revision: updated.revision,
    remotePage: { ...encrypted, revision: updated.revision },
  };
}

export async function changeSessionPassword(
  services: SessionServices,
  session: OpenSession,
  document: OutlineDocument,
  newPassword: string,
): Promise<OpenSession> {
  const writeToken = services.secrets.writeToken();
  const encrypted = await services.crypto.encrypt({ document, writeToken }, newPassword, session.urlSecret);
  const updated = await services.repository.updatePage(session.pageId, session.writeToken, {
    expectedRevision: session.revision,
    ciphertext: encrypted.ciphertext,
    salt: encrypted.salt,
    newWriteToken: writeToken,
  });
  return {
    ...session,
    document,
    password: newPassword,
    writeToken,
    revision: updated.revision,
    remotePage: { ...encrypted, revision: updated.revision },
  };
}

export async function rotateSessionLink(
  services: SessionServices,
  session: OpenSession,
  document: OutlineDocument,
): Promise<OpenSession> {
  const pageId = services.secrets.pageId();
  const urlSecret = services.secrets.urlSecret();
  const writeToken = services.secrets.writeToken();
  const identity = identityFor(pageId, urlSecret);
  const encrypted = await services.crypto.encrypt({ document, writeToken }, session.password, urlSecret);
  const updated = await services.repository.rotatePage(
    session.pageId,
    session.writeToken,
    { newId: pageId, salt: encrypted.salt, ciphertext: encrypted.ciphertext, newWriteToken: writeToken },
    identity.locator,
  );
  return {
    ...session,
    ...identity,
    document,
    writeToken,
    revision: updated.revision,
    remotePage: { ...encrypted, revision: updated.revision },
  };
}

export function identityFor(pageId: string, urlSecret: string): SessionIdentity {
  return { pageId, urlSecret, locator: `/p/${pageId}#${urlSecret}` };
}
