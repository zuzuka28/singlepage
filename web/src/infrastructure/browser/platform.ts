import type {
  AccessPresentation,
  Clipboard,
  FileIO,
  Navigation,
  ThemePort,
} from '../../shared/platform/ports';
import type { PageHistory } from '../../features/page-session/logic/ports';

export class BrowserHistory implements PageHistory {
  readonly available = false;
  readonly error = '';
  async restore() { return null; }
  async remember(_locator: string) { return true; }
  async list() { return []; }
}

export class BrowserNavigation implements Navigation {
  get origin() { return location.origin; }
  get locator() { return `${location.pathname}${location.hash}`; }
  replace(locator: string) { history.replaceState({}, '', locator); }
  assign(href: string) { location.assign(href); }
  reload() { location.reload(); }
}

export const browserClipboard: Clipboard = {
  async writeText(value) {
    if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
    await navigator.clipboard.writeText(value);
  },
};

export const browserFiles: FileIO = {
  exportMarkdown(contents, fileName) {
    const href = URL.createObjectURL(new Blob([contents], { type: 'text/markdown;charset=utf-8' }));
    const link = document.createElement('a');
    link.href = href;
    link.download = fileName;
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 0);
  },
};

export const browserTheme: ThemePort = {
  read() {
    const saved = localStorage.getItem('singlepage-theme');
    if (saved === 'light' || saved === 'dark') return saved;
    return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  },
  apply(theme, persist = true) {
    document.documentElement.dataset.theme = theme;
    document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')?.setAttribute(
      'content',
      theme === 'dark' ? '#1d2027' : '#f7f8fb',
    );
    if (persist) localStorage.setItem('singlepage-theme', theme);
  },
};

export function accessPresentation(kind: AccessPresentation['kind']): AccessPresentation {
  return {
    kind,
    present(locator) {
      return kind === 'browser' ? `${location.origin}${locator}` : locator;
    },
  };
}
