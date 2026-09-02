export interface Navigation {
  readonly origin: string;
  readonly locator: string;
  replace(locator: string): void;
  assign(href: string): void;
  reload(): void;
}

export interface Clipboard {
  writeText(value: string): Promise<void>;
}

export interface FileIO {
  exportMarkdown(contents: string, fileName: string): void;
}

export interface ThemePort {
  read(): 'light' | 'dark' | null;
  apply(theme: 'light' | 'dark', persist?: boolean): void;
}

export interface AccessPresentation {
  readonly kind: 'browser' | 'local-vault';
  present(locator: string): string;
}
