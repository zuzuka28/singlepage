import type { OutlineDocument } from '../../../entities/outline';

export interface RemotePage {
  revision: number;
  salt: Uint8Array;
  ciphertext: Uint8Array;
}

export interface CreatePageRequest {
  id: string;
  salt: Uint8Array;
  ciphertext: Uint8Array;
  writeToken: string;
}

export interface UpdatePageRequest {
  expectedRevision: number;
  ciphertext: Uint8Array;
  salt?: Uint8Array;
  newWriteToken?: string;
}

export interface RotatePageRequest {
  newId: string;
  salt: Uint8Array;
  ciphertext: Uint8Array;
  newWriteToken: string;
}

export interface PageRepository {
  createPage(request: CreatePageRequest, locator?: string): Promise<{ revision: number }>;
  getPage(id: string, signal?: AbortSignal): Promise<RemotePage>;
  updatePage(id: string, writeToken: string, request: UpdatePageRequest): Promise<{ revision: number }>;
  rotatePage(id: string, writeToken: string, request: RotatePageRequest, locator?: string): Promise<{ revision: number }>;
}

export interface PageHistory {
  readonly available: boolean;
  readonly error: string;
  restore(): Promise<string | null>;
  remember(locator: string): Promise<boolean>;
  list(): Promise<string[]>;
}

export interface EncryptedVault {
  salt: Uint8Array;
  ciphertext: Uint8Array;
}

export interface VaultContents {
  document: OutlineDocument;
  writeToken: string;
}

export interface VaultCrypto {
  encrypt(contents: VaultContents, password: string, urlSecret: string, salt?: Uint8Array): Promise<EncryptedVault>;
  decrypt(payload: EncryptedVault, password: string, urlSecret: string): Promise<VaultContents>;
}

export interface SecretSource {
  pageId(): string;
  urlSecret(): string;
  writeToken(): string;
  salt(): Uint8Array;
}
