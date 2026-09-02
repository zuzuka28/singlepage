import { fromBase64, toBase64 } from '../../../../../web/src/infrastructure/crypto';
import {
  RemoteApiError,
  RevisionConflictError,
  type CreatePageRequest,
  type PageClient,
  type RemotePage,
  type RotatePageRequest,
  type UpdatePageRequest
} from '../../../../../web/src/infrastructure/api';
import type { ApplicationRuntime } from '../../../../../web/src/app/logic/runtime';
import { browserSecrets, browserVaultCrypto } from '../../../../../web/src/infrastructure/browser/crypto';
import {
  accessPresentation,
  browserClipboard,
  browserFiles,
  BrowserNavigation,
  browserTheme
} from '../../../../../web/src/infrastructure/browser/platform';
import type { PageHistory } from '../../../../../web/src/features/page-session/logic/ports';
import {
  CreatePage,
  GetPage,
  ListLocators,
  RememberLocator,
  RestoreLocator,
  RotatePage,
  UpdatePage
} from './bindings/singlepage/cmd/app/internal/page/service';

class NativePageClient implements PageClient {
  async createPage(request: CreatePageRequest, locator = '') {
    return call<{ revision: number }>(() => CreatePage({
      ...request,
      salt: toBase64(request.salt),
      ciphertext: toBase64(request.ciphertext)
    }, locator) as unknown as Promise<{ revision: number }>);
  }

  async getPage(id: string): Promise<RemotePage> {
    const page = await call<{ revision: number; salt: string; ciphertext: string }>(
      () => GetPage(id) as unknown as Promise<{ revision: number; salt: string; ciphertext: string }>
    );
    return {
      revision: page.revision,
      salt: fromBase64(page.salt),
      ciphertext: fromBase64(page.ciphertext)
    };
  }

  async updatePage(id: string, writeToken: string, request: UpdatePageRequest) {
    return call<{ revision: number }>(() => UpdatePage(id, writeToken, {
      expectedRevision: request.expectedRevision,
      ciphertext: toBase64(request.ciphertext),
      ...(request.salt ? { salt: toBase64(request.salt) } : {}),
      ...(request.newWriteToken ? { newWriteToken: request.newWriteToken } : {})
    }) as unknown as Promise<{ revision: number }>);
  }

  async rotatePage(id: string, writeToken: string, request: RotatePageRequest, locator = '') {
    return call<{ revision: number }>(() => RotatePage(id, writeToken, {
      ...request,
      salt: toBase64(request.salt),
      ciphertext: toBase64(request.ciphertext)
    }, locator) as unknown as Promise<{ revision: number }>);
  }
}

export class NativeRuntime implements ApplicationRuntime {
  readonly repository: PageClient = new NativePageClient();
  readonly api = this.repository;
  readonly history = new NativeHistory();
  readonly navigation = new BrowserNavigation();
  readonly clipboard = browserClipboard;
  readonly files = browserFiles;
  readonly theme = browserTheme;
  readonly crypto = browserVaultCrypto;
  readonly secrets = browserSecrets;
  readonly access = accessPresentation('local-vault');
  readonly native = true;
  get sessionError() { return this.history.error; }

  async restoreLocator() {
    const locator = await this.history.restore();
    if (locator) this.navigation.replace(locator);
  }

  async rememberLocator(locator: string) { return this.history.remember(locator); }

  async listLocators() { return this.history.list(); }
}

class NativeHistory implements PageHistory {
  readonly available = true;
  error = '';

  async restore() {
    try {
      const locator = await RestoreLocator();
      this.error = '';
      return locator || null;
    } catch (error) {
      console.error('Native session restore failed', error);
      this.error = 'Unable to restore the last page.';
      return null;
    }
  }

  async remember(locator: string) {
    try {
      await RememberLocator(locator);
      this.error = '';
      return true;
    } catch (error) {
      console.error('Native session persistence failed', error);
      this.error = 'Unable to save restart recovery. Keep this window open and retry.';
      return false;
    }
  }

  async list() {
    try {
      const locators = await ListLocators();
      this.error = '';
      return locators;
    } catch (error) {
      console.error('Native page history failed', error);
      this.error = 'Unable to load previously opened pages.';
      return [];
    }
  }
}

async function call<T>(operation: () => Promise<T>): Promise<T> {
  try {
    return await operation();
  } catch (error) {
    const details = errorDetails(error);
    if (details.status === 409) throw new RevisionConflictError();
    throw new RemoteApiError(details.status, details.message);
  }
}

function errorDetails(error: unknown): { status: number; message: string } {
  const candidate = error instanceof Error && error.cause ? error.cause : error;
  if (candidate && typeof candidate === 'object' && 'status' in candidate) {
    const status = Number(candidate.status);
    const message = 'message' in candidate ? String(candidate.message) : 'Native request failed';
    if (Number.isFinite(status)) return { status, message };
  }
  if (error instanceof Error) {
    try {
      const parsed = JSON.parse(error.message) as { status?: number; message?: string };
      if (typeof parsed.status === 'number') return { status: parsed.status, message: parsed.message ?? error.message };
    } catch {}
    return { status: 500, message: error.message };
  }
  return { status: 500, message: 'Native request failed' };
}
