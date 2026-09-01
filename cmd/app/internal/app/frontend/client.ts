import { fromBase64, toBase64 } from '../../../../../web/src/crypto';
import {
  RemoteApiError,
  RevisionConflictError,
  type CreatePageRequest,
  type PageClient,
  type RemotePage,
  type RotatePageRequest,
  type UpdatePageRequest
} from '../../../../../web/src/remote';
import type { ApplicationRuntime } from '../../../../../web/src/runtime';
import {
  CreatePage,
  GetPage,
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
  readonly api: PageClient = new NativePageClient();
  readonly native = true;
  sessionError = '';

  async restoreLocator() {
    try {
      const locator = await RestoreLocator();
      history.replaceState({}, '', locator);
    } catch (error) {
      console.error('Native session restore failed', error);
      this.sessionError = 'Unable to restore the last page.';
    }
  }

  async rememberLocator(locator: string) {
    try {
      await RememberLocator(locator);
      this.sessionError = '';
      return true;
    } catch (error) {
      console.error('Native session persistence failed', error);
      this.sessionError = 'Unable to save restart recovery. Keep this window open and retry.';
      return false;
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
