import { describe, expect, it } from "vitest";
import { changePassword, decryptJson, encryptJson, rotateAccess } from "./crypto";

const fast = { iterations: 1_000 };

describe("Web Crypto helpers", () => {
  it("round-trips encrypted JSON", async () => {
    const encrypted = await encryptJson({ private: "данные" }, "password", "secret", undefined, fast);
    await expect(decryptJson(encrypted, "password", "secret", fast)).resolves.toEqual({ private: "данные" });
  });

  it("rejects a wrong password or URL secret without exposing plaintext", async () => {
    const encrypted = await encryptJson({ private: true }, "password", "secret", undefined, fast);
    await expect(decryptJson(encrypted, "wrong", "secret", fast)).rejects.toThrow();
    await expect(decryptJson(encrypted, "password", "wrong", fast)).rejects.toThrow();
  });

  it("password rotation invalidates the old password", async () => {
    const rotated = await changePassword({ version: 2 }, "new-password", "secret", fast);
    await expect(decryptJson(rotated, "new-password", "secret", fast)).resolves.toEqual({ version: 2 });
    await expect(decryptJson(rotated, "old-password", "secret", fast)).rejects.toThrow();
  });

  it("access rotation creates a new secret and invalidates the old one", async () => {
    const old = await encryptJson({ version: 2 }, "password", "old-secret", undefined, fast);
    const rotated = await rotateAccess({ version: 2 }, "password", fast);
    expect(rotated.urlSecret).not.toBe("old-secret");
    expect(rotated.writeToken.length).toBeGreaterThan(20);
    await expect(decryptJson(rotated, "password", rotated.urlSecret, fast)).resolves.toEqual({ version: 2 });
    await expect(decryptJson(rotated, "password", "old-secret", fast)).rejects.toThrow();
    await expect(decryptJson(old, "password", "old-secret", fast)).resolves.toEqual({ version: 2 });
  });
});
