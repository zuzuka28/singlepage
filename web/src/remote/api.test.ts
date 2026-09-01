import { describe, expect, it, vi } from "vitest";
import { PageApi, RevisionConflictError } from "./api";

describe("opaque remote API", () => {
  it("encodes opaque bytes and bearer capability without plaintext fields", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('{"revision":2}', { status: 200 }));
    const api = new PageApi("https://example.test", fetchMock);
    await api.updatePage("page/id", "write-secret", { expectedRevision: 1, ciphertext: new Uint8Array([1, 2]), salt: new Uint8Array([3]) });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("https://example.test/api/pages/page%2Fid");
    expect(init?.headers).toMatchObject({ Authorization: "Bearer write-secret" });
    expect(JSON.parse(String(init?.body))).toEqual({ expectedRevision: 1, ciphertext: "AQI=", salt: "Aw==" });
  });

  it("decodes fetched opaque bytes", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response('{"revision":4,"salt":"Aw==","ciphertext":"AQI="}'));
    await expect(new PageApi("", fetchMock).getPage("id")).resolves.toEqual({
      revision: 4,
      salt: new Uint8Array([3]),
      ciphertext: new Uint8Array([1, 2]),
    });
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

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("https://example.test/api/pages/old-id/rotate");
    expect(init?.headers).toMatchObject({ Authorization: "Bearer old-token" });
    expect(JSON.parse(String(init?.body))).toEqual({
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
