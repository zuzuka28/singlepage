import { normalize, propertyValueText, type IndexedBlock, type MetadataValue } from "./metadata";
import { parseQuery, type Comparison, type DateCondition, type SearchQuery } from "./query";

export interface SearchResult { id: string; score: number; ancestors: string[] }

export function search(index: IndexedBlock[], input: string | SearchQuery): SearchResult[] {
  const query = typeof input === "string" ? parseQuery(input) : input;
  return index
    .filter((block) => matchesQuery(block, query))
    .map((block) => ({ id: block.id, score: score(block, query), ancestors: block.ancestors }))
    .sort((a, b) => b.score - a.score);
}

export function matchesQuery(block: IndexedBlock, query: SearchQuery): boolean {
  return (
    query.text.every((condition) => condition.negated !== block.normalizedText.includes(condition.value)) &&
    query.tags.every((condition) => condition.negated !== block.effective.tags.has(condition.value)) &&
    query.properties.every((condition) => {
      const property = block.effective.properties.get(condition.key);
      const matched = property !== undefined && compareProperty(property, condition.comparison, condition.value);
      return condition.negated !== matched;
    }) &&
    query.dates.every((condition) => condition.negated !== [...block.effective.dates].some((date) => compareDate(date, condition)))
  );
}

function compareProperty(property: MetadataValue, comparison: Comparison, queryValue: string): boolean {
  if (comparison !== "=" || /^[+-]?(?:\d+\.?\d*|\.\d+)$/.test(queryValue)) {
    const expected = Number(queryValue);
    if (property.type !== "number" || !Number.isFinite(expected)) return false;
    return compare(property.value, comparison, expected);
  }
  return normalize(propertyValueText(property)).includes(queryValue);
}

function compareDate(date: string, condition: DateCondition): boolean {
  if (condition.end) return date >= condition.value && date <= condition.end;
  return compare(date, condition.comparison ?? "=", condition.value);
}

function compare<T extends number | string>(actual: T, comparison: Comparison, expected: T): boolean {
  if (comparison === ">") return actual > expected;
  if (comparison === ">=") return actual >= expected;
  if (comparison === "<") return actual < expected;
  if (comparison === "<=") return actual <= expected;
  return actual === expected;
}

function score(block: IndexedBlock, query: SearchQuery): number {
  let result = 0;
  const tokens = block.normalizedText.split(/[^\p{L}\p{N}_-]+/u);
  for (const condition of query.text.filter((item) => !item.negated)) {
    if (block.normalizedText === condition.value) result += 100;
    else if (tokens.includes(condition.value)) result += 20;
    else if (block.normalizedText.includes(condition.value)) result += 5;
  }
  return result;
}

export interface AutocompleteData {
  tags: string[];
  properties: Record<string, string[]>;
}

export function buildAutocomplete(index: IndexedBlock[]): AutocompleteData {
  const tags = new Set<string>();
  const properties = new Map<string, Set<string>>();
  for (const block of index) {
    for (const tag of block.local.tags) tags.add(tag);
    for (const [key, value] of block.local.properties) {
      if (!properties.has(key)) properties.set(key, new Set());
      properties.get(key)!.add(propertyValueText(value));
    }
  }
  return {
    tags: [...tags].sort(),
    properties: Object.fromEntries([...properties].sort().map(([key, values]) => [key, [...values].sort()])),
  };
}
