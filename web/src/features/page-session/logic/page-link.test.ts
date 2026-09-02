import { describe, expect, it } from 'vitest';
import { pageIDFromLocator, parsePageLink } from './page-link';

const locator = '/p/0123456789abcdef#fedcba9876543210';

describe('parsePageLink', () => {
  it('accepts relative page locators', () => {
    expect(parsePageLink(locator, 'https://singlepage.example/')).toEqual({
      href: `https://singlepage.example${locator}`,
      locator,
      origin: 'https://singlepage.example'
    });
  });

  it('accepts full page links and normalizes a trailing slash', () => {
    expect(parsePageLink(`https://notes.example/p/0123456789abcdef/#fedcba9876543210`, 'https://singlepage.example/')).toEqual({
      href: `https://notes.example${locator}`,
      locator,
      origin: 'https://notes.example'
    });
  });

  it.each([
    '',
    '/p/short#fedcba9876543210',
    '/p/0123456789abcdef#short',
    '/other/0123456789abcdef#fedcba9876543210'
  ])('rejects invalid input %j', (value) => {
    expect(parsePageLink(value, 'https://singlepage.example/')).toBeNull();
  });
});

it('extracts the page ID for history labels', () => {
  expect(pageIDFromLocator(locator)).toBe('0123456789abcdef');
});
