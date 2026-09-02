import { isIsoDate, normalize } from "./metadata";

export type Comparison = "=" | ">" | ">=" | "<" | "<=";
export interface Condition<T> { value: T; negated: boolean }
export interface PropertyCondition extends Condition<string> { key: string; comparison: Comparison }
export interface DateCondition extends Condition<string> { comparison?: Comparison; end?: string }

export interface SearchQuery {
  text: Condition<string>[];
  tags: Condition<string>[];
  properties: PropertyCondition[];
  dates: DateCondition[];
}

export function parseQuery(input: string): SearchQuery {
  const query: SearchQuery = { text: [], tags: [], properties: [], dates: [] };
  for (const rawToken of tokenize(input)) {
    const negated = rawToken.startsWith("-") && rawToken.length > 1;
    const token = negated ? rawToken.slice(1) : rawToken;
    if (token.startsWith("#") && token.length > 1) {
      query.tags.push({ value: normalize(token.slice(1)), negated });
      continue;
    }
    if (token.startsWith("@")) {
      const date = parseDateCondition(token.slice(1), negated);
      if (date) { query.dates.push(date); continue; }
    }
    const property = /^([\p{L}\p{N}_-]+):(>=|<=|>|<)?(.+)$/u.exec(token);
    if (property) {
      query.properties.push({ key: normalize(property[1]), comparison: (property[2] || "=") as Comparison, value: normalize(property[3]), negated });
      continue;
    }
    if (token) query.text.push({ value: normalize(token), negated });
  }
  return query;
}

function tokenize(input: string): string[] {
  const tokens: string[] = [];
  for (const match of input.matchAll(/"([^"]+)"|'([^']+)'|(\S+)/g)) tokens.push(match[1] ?? match[2] ?? match[3]);
  return tokens;
}

function parseDateCondition(value: string, negated: boolean): DateCondition | null {
  const range = /^(\d{4}-\d{2}-\d{2})\.\.(\d{4}-\d{2}-\d{2})$/.exec(value);
  if (range && isIsoDate(range[1]) && isIsoDate(range[2])) return { value: range[1], end: range[2], negated };
  const single = /^(>=|<=|>|<)?(\d{4}-\d{2}-\d{2})$/.exec(value);
  if (!single || !isIsoDate(single[2])) return null;
  return { comparison: (single[1] || "=") as Comparison, value: single[2], negated };
}
