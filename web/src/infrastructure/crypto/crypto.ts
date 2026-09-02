const encoder = new TextEncoder();
const decoder = new TextDecoder();
const AES_IV_BYTES = 12;
const ENVELOPE_VERSION = 1;

export interface KeyDerivationOptions { iterations?: number }

export interface EncryptedPayload {
  salt: Uint8Array;
  ciphertext: Uint8Array;
}

export interface RotatedEncryption extends EncryptedPayload {
  urlSecret: string;
  writeToken: string;
}

function webCrypto(): Crypto {
  if (!globalThis.crypto?.subtle) throw new Error("Web Crypto API is unavailable");
  return globalThis.crypto;
}

export function randomBytes(length: number): Uint8Array {
  return webCrypto().getRandomValues(new Uint8Array(length));
}

export function generateSecret(length = 32): string {
  return toBase64Url(randomBytes(length));
}

export const generateWriteToken = generateSecret;

export async function deriveEncryptionKey(
  password: string,
  urlSecret: string,
  salt: Uint8Array,
  options: KeyDerivationOptions = {},
): Promise<CryptoKey> {
  const crypto = webCrypto();
  const passwordMaterial = await crypto.subtle.importKey("raw", encoder.encode(password), "PBKDF2", false, ["deriveBits"]);
  const passwordBits = await crypto.subtle.deriveBits(
    { name: "PBKDF2", hash: "SHA-256", salt: copyBuffer(salt), iterations: options.iterations ?? 310_000 },
    passwordMaterial,
    256,
  );
  const hkdfMaterial = await crypto.subtle.importKey("raw", passwordBits, "HKDF", false, ["deriveKey"]);
  return crypto.subtle.deriveKey(
    { name: "HKDF", hash: "SHA-256", salt: encoder.encode(urlSecret), info: encoder.encode("zero-knowledge-outline/v1") },
    hkdfMaterial,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

export async function encryptBytes(plaintext: Uint8Array, key: CryptoKey): Promise<Uint8Array> {
  const iv = randomBytes(AES_IV_BYTES);
  const encrypted = new Uint8Array(
    await webCrypto().subtle.encrypt({ name: "AES-GCM", iv: copyBuffer(iv) }, key, copyBuffer(plaintext)),
  );
  const envelope = new Uint8Array(1 + iv.length + encrypted.length);
  envelope[0] = ENVELOPE_VERSION;
  envelope.set(iv, 1);
  envelope.set(encrypted, 1 + iv.length);
  return envelope;
}

export async function decryptBytes(envelope: Uint8Array, key: CryptoKey): Promise<Uint8Array> {
  if (envelope.length <= 1 + AES_IV_BYTES || envelope[0] !== ENVELOPE_VERSION) throw new Error("Invalid encrypted payload");
  const iv = envelope.slice(1, 1 + AES_IV_BYTES);
  const ciphertext = envelope.slice(1 + AES_IV_BYTES);
  return new Uint8Array(
    await webCrypto().subtle.decrypt({ name: "AES-GCM", iv: copyBuffer(iv) }, key, copyBuffer(ciphertext)),
  );
}

export async function encryptJson<T>(
  value: T,
  password: string,
  urlSecret: string,
  salt = randomBytes(16),
  options: KeyDerivationOptions = {},
): Promise<EncryptedPayload> {
  const key = await deriveEncryptionKey(password, urlSecret, salt, options);
  return { salt, ciphertext: await encryptBytes(encoder.encode(JSON.stringify(value)), key) };
}

export async function decryptJson<T>(
  payload: EncryptedPayload,
  password: string,
  urlSecret: string,
  options: KeyDerivationOptions = {},
): Promise<T> {
  const key = await deriveEncryptionKey(password, urlSecret, payload.salt, options);
  return JSON.parse(decoder.decode(await decryptBytes(payload.ciphertext, key))) as T;
}

export async function changePassword<T>(
  value: T,
  newPassword: string,
  urlSecret: string,
  options: KeyDerivationOptions = {},
): Promise<EncryptedPayload> {
  return encryptJson(value, newPassword, urlSecret, randomBytes(16), options);
}

export async function rotateAccess<T>(
  value: T,
  password: string,
  options: KeyDerivationOptions = {},
): Promise<RotatedEncryption> {
  const urlSecret = generateSecret();
  const writeToken = generateWriteToken();
  return { ...(await encryptJson(value, password, urlSecret, randomBytes(16), options)), urlSecret, writeToken };
}

export function toBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
  }
  return btoa(binary);
}

export function fromBase64(value: string): Uint8Array {
  const binary = atob(value);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

export function toBase64Url(bytes: Uint8Array): string {
  return toBase64(bytes).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

export function fromBase64Url(value: string): Uint8Array {
  return fromBase64(value.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(value.length / 4) * 4, "="));
}

function copyBuffer(bytes: Uint8Array): ArrayBuffer {
  return Uint8Array.from(bytes).buffer;
}
