import { fromBase64, toBase64 } from "../crypto";

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

export class RemoteApiError extends Error {
  constructor(public readonly status: number, message: string) { super(message); }
}

export class RevisionConflictError extends RemoteApiError {
  constructor() { super(409, "This page was changed elsewhere."); }
}

export class PageApi {
  constructor(
    private readonly baseUrl = "",
    private readonly fetchImpl: typeof fetch = globalThis.fetch.bind(globalThis),
  ) {}

  async createPage(request: CreatePageRequest): Promise<{ revision: number }> {
    return this.request("/api/pages", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...request, salt: toBase64(request.salt), ciphertext: toBase64(request.ciphertext) }),
    });
  }

  async getPage(id: string): Promise<RemotePage> {
    const response = await this.request<{ revision: number; salt: string; ciphertext: string }>(`/api/pages/${encodeURIComponent(id)}`);
    return { revision: response.revision, salt: fromBase64(response.salt), ciphertext: fromBase64(response.ciphertext) };
  }

  async updatePage(id: string, writeToken: string, request: UpdatePageRequest): Promise<{ revision: number }> {
    const body: Record<string, unknown> = {
      expectedRevision: request.expectedRevision,
      ciphertext: toBase64(request.ciphertext),
    };
    if (request.salt) body.salt = toBase64(request.salt);
    if (request.newWriteToken) body.newWriteToken = request.newWriteToken;
    return this.request(`/api/pages/${encodeURIComponent(id)}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${writeToken}` },
      body: JSON.stringify(body),
    });
  }

  async rotatePage(id: string, writeToken: string, request: RotatePageRequest): Promise<{ revision: number }> {
    return this.request(`/api/pages/${encodeURIComponent(id)}/rotate`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${writeToken}` },
      body: JSON.stringify({
        ...request,
        salt: toBase64(request.salt),
        ciphertext: toBase64(request.ciphertext),
      }),
    });
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await this.fetchImpl(`${this.baseUrl}${path}`, init);
    if (response.status === 409) throw new RevisionConflictError();
    if (!response.ok) {
      const message = (await response.text()).trim() || `Remote request failed (${response.status})`;
      throw new RemoteApiError(response.status, message);
    }
    return response.json() as Promise<T>;
  }
}
