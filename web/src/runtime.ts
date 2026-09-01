import { PageApi, type PageClient } from './remote';

export interface ApplicationRuntime {
  readonly api: PageClient;
  readonly native: boolean;
  sessionError: string;
  restoreLocator(): Promise<void>;
  rememberLocator(locator: string): Promise<boolean>;
}

class BrowserRuntime implements ApplicationRuntime {
  readonly api = new PageApi();
  readonly native = false;
  sessionError = '';

  async restoreLocator() {}
  async rememberLocator() { return true; }
}

export async function createApplicationRuntime(): Promise<ApplicationRuntime> {
  if (import.meta.env.VITE_SINGLEPAGE_RUNTIME !== 'wails') return new BrowserRuntime();

  const { NativeRuntime } = await import('../../cmd/app/internal/app/frontend/client');
  return new NativeRuntime();
}
