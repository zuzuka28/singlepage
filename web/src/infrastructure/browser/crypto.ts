import { decryptJson, encryptJson, generateSecret, generateWriteToken, randomBytes } from '../crypto';
import type { SecretSource, VaultCrypto } from '../../features/page-session/logic/ports';

export const browserVaultCrypto: VaultCrypto = {
  encrypt: encryptJson,
  decrypt: decryptJson,
};

export const browserSecrets: SecretSource = {
  pageId: () => generateSecret(16),
  urlSecret: () => generateSecret(32),
  writeToken: () => generateWriteToken(32),
  salt: () => randomBytes(16),
};
