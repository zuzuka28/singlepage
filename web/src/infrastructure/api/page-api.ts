import createClient, { type Client } from 'openapi-fetch';
import { fromBase64, toBase64 } from '../crypto';
import type { paths } from '../../generated/openapi/schema';
import { ApplicationError, errorFromStatus } from '../../features/page-session/logic/errors';
import type {
  CreatePageRequest,
  PageRepository,
  RemotePage,
  RotatePageRequest,
  UpdatePageRequest,
} from '../../features/page-session/logic/ports';

export type {
  CreatePageRequest,
  PageRepository as PageClient,
  RemotePage,
  RotatePageRequest,
  UpdatePageRequest,
} from '../../features/page-session/logic/ports';

export class RemoteApiError extends ApplicationError {
  constructor(public readonly status: number, message: string) {
    super(errorFromStatus(status), message);
  }
}

export class RevisionConflictError extends RemoteApiError {
  constructor() {
    super(409, 'This page was changed elsewhere.');
  }
}

export class PageApi implements PageRepository {
  private readonly client: Client<paths>;

  constructor(baseUrl = '', fetchImpl: typeof fetch = globalThis.fetch.bind(globalThis)) {
    const effectiveBaseUrl = baseUrl || globalThis.location?.origin || 'http://singlepage.invalid';
    this.client = createClient<paths>({ baseUrl: effectiveBaseUrl, fetch: fetchImpl });
  }

  async createPage(request: CreatePageRequest, _locator?: string): Promise<{ revision: number }> {
    const result = await this.client.POST('/api/pages', {
      body: {
        ...request,
        salt: toBase64(request.salt),
        ciphertext: toBase64(request.ciphertext),
      },
    });
    return unwrap(result);
  }

  async getPage(id: string, signal?: AbortSignal): Promise<RemotePage> {
    const result = await this.client.GET('/api/pages/{id}', {
      params: { path: { id } },
      signal,
    });
    const page = unwrap(result);
    return {
      revision: page.revision,
      salt: fromBase64(page.salt),
      ciphertext: fromBase64(page.ciphertext),
    };
  }

  async updatePage(id: string, writeToken: string, request: UpdatePageRequest): Promise<{ revision: number }> {
    const result = await this.client.PUT('/api/pages/{id}', {
      params: { path: { id } },
      headers: { Authorization: `Bearer ${writeToken}` },
      body: {
        expectedRevision: request.expectedRevision,
        ciphertext: toBase64(request.ciphertext),
        ...(request.salt ? { salt: toBase64(request.salt) } : {}),
        ...(request.newWriteToken ? { newWriteToken: request.newWriteToken } : {}),
      },
    });
    return unwrap(result);
  }

  async rotatePage(id: string, writeToken: string, request: RotatePageRequest, _locator?: string): Promise<{ revision: number }> {
    const result = await this.client.POST('/api/pages/{id}/rotate', {
      params: { path: { id } },
      headers: { Authorization: `Bearer ${writeToken}` },
      body: {
        ...request,
        salt: toBase64(request.salt),
        ciphertext: toBase64(request.ciphertext),
      },
    });
    return unwrap(result);
  }
}

function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.data !== undefined) return result.data;
  const status = result.response.status;
  if (status === 409) throw new RevisionConflictError();
  const message = errorMessage(result.error) || `Remote request failed (${status})`;
  throw new RemoteApiError(status, message);
}

function errorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'error' in error) return String(error.error);
  return typeof error === 'string' ? error.trim() : '';
}
