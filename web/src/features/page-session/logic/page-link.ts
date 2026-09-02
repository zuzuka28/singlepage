const pagePathPattern = /^\/p\/([A-Za-z0-9_-]{16,128})\/?$/;
const secretPattern = /^[A-Za-z0-9_-]{16,128}$/;

export interface PageLink {
  href: string;
  locator: string;
  origin: string;
}

export function parsePageLink(value: string, base: string): PageLink | null {
  const input = value.trim();
  if (!input) return null;

  let url: URL;
  try {
    url = new URL(input, base);
  } catch {
    return null;
  }

  const route = pagePathPattern.exec(url.pathname);
  const secret = url.hash.slice(1);
  if (!route || !secretPattern.test(secret)) return null;

  const locator = `/p/${route[1]}#${secret}`;
  return { href: new URL(locator, url).href, locator, origin: url.origin };
}

export function pageIDFromLocator(locator: string): string {
  return pagePathPattern.exec(locator.split('#', 1)[0])?.[1] ?? locator;
}
