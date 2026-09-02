import { describe, expect, it, vi } from "vitest";
import { PageApi, RevisionConflictError } from "./page-api";

describe("opaque remote API", () => {
  it("encodes opaque bytes and bearer capability without plaintext fields", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('{"revision":2}', { status: 200 }));
    const api = new PageApi("https://example.test", fetchMock);
    await api.updatePage("page/id", "write-secret", { expectedRevision: 1, ciphertext: new Uint8Array([1, 2]), salt: new Uint8Array([3]) });
    const [request] = fetchMock.mock.calls[0] as [Request];
    expect(request.url).toBe("https://example.test/api/pages/page%2Fid");
    expect(request.headers.get('Authorization')).toBe("Bearer write-secret");
    expect(await request.json()).toEqual({ expectedRevision: 1, ciphertext: "AQI=", salt: "Aw==" });
  });

  it("decodes fetched opaque bytes", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('{"revision":4,"salt":"Aw==","ciphertext":"AQI="}'));
    await expect(new PageApi("", fetchMock).getPage("id")).resolves.toEqual({
      revision: 4,
      salt: new Uint8Array([3]),
      ciphertext: new Uint8Array([1, 2]),
    });
    expect((fetchMock.mock.calls[0][0] as Request).url).toBe('http://singlepage.invalid/api/pages/id');
  });

  it("encodes atomic page-id rotation", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('{"revision":1}', { status: 200 }));
    const api = new PageApi("https://example.test", fetchMock);
    await api.rotatePage("old-id", "old-token", {
      newId: "new-id",
      salt: new Uint8Array([3]),
      ciphertext: new Uint8Array([1, 2]),
      newWriteToken: "new-token",
    });

    const [request] = fetchMock.mock.calls[0] as [Request];
    expect(request.url).toBe("https://example.test/api/pages/old-id/rotate");
    expect(request.headers.get('Authorization')).toBe("Bearer old-token");
    expect(await request.json()).toEqual({
      newId: "new-id",
      salt: "Aw==",
      ciphertext: "AQI=",
      newWriteToken: "new-token",
    });
  });

  it("maps stale writes to an explicit conflict", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response("", { status: 409 }));
    await expect(new PageApi("", fetchMock).updatePage("id", "token", { expectedRevision: 1, ciphertext: new Uint8Array([1]) }))
      .rejects.toBeInstanceOf(RevisionConflictError);
  });
});
