import { browserSecrets, browserVaultCrypto } from '../../infrastructure/browser/crypto';
import {
  accessPresentation,
  browserClipboard,
  browserFiles,
  BrowserHistory,
  BrowserNavigation,
  browserTheme,
} from '../../infrastructure/browser/platform';
import type { PageHistory, PageRepository, SecretSource, VaultCrypto } from '../../features/page-session/logic/ports';
import type { AccessPresentation, Clipboard, FileIO, Navigation, ThemePort } from '../../shared/platform/ports';
import { PageApi } from '../../infrastructure/api';

export interface ApplicationRuntime {
  readonly repository: PageRepository;
  readonly api: PageRepository;
  readonly history: PageHistory;
  readonly navigation: Navigation;
  readonly clipboard: Clipboard;
  readonly files: FileIO;
  readonly theme: ThemePort;
  readonly crypto: VaultCrypto;
  readonly secrets: SecretSource;
  readonly access: AccessPresentation;
  /** @deprecated Presentation should use access.kind. */
  readonly native: boolean;
  readonly sessionError: string;
  restoreLocator(): Promise<void>;
  rememberLocator(locator: string): Promise<boolean>;
  listLocators(): Promise<string[]>;
}

class BrowserRuntime implements ApplicationRuntime {
  readonly repository = new PageApi();
  readonly api = this.repository;
  readonly history = new BrowserHistory();
  readonly navigation = new BrowserNavigation();
  readonly clipboard = browserClipboard;
  readonly files = browserFiles;
  readonly theme = browserTheme;
  readonly crypto = browserVaultCrypto;
  readonly secrets = browserSecrets;
  readonly access = accessPresentation('browser');
  readonly native = false;
  get sessionError() { return this.history.error; }

  async restoreLocator() {
    const locator = await this.history.restore();
    if (locator) this.navigation.replace(locator);
  }
  async rememberLocator(locator: string) { return this.history.remember(locator); }
  async listLocators() { return this.history.list(); }
}

export async function createApplicationRuntime(): Promise<ApplicationRuntime> {
  if (import.meta.env.VITE_SINGLEPAGE_RUNTIME !== 'wails') return new BrowserRuntime();

  const { NativeRuntime } = await import('../../../../cmd/app/internal/app/frontend/client');
  return new NativeRuntime();
}
