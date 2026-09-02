import type { OutlineDocument } from "./document";

export type MetadataValue =
  | { type: "string"; value: string }
  | { type: "number"; value: number; raw: string }
  | { type: "boolean"; value: boolean; raw: string }
  | { type: "date"; value: string };

export interface Metadata {
  tags: Set<string>;
  properties: Map<string, MetadataValue>;
  dates: Set<string>;
}

export interface IndexedBlock {
  id: string;
  text: string;
  normalizedText: string;
  ancestors: string[];
  local: Metadata;
  effective: Metadata;
}

const tagPattern = /(^|[^\p{L}\p{N}_])#([\p{L}\p{N}_-]+)/gu;
const propertyPattern = /(^|\s)([\p{L}\p{N}_-]+)::([^\s]+)/gu;
const datePattern = /(?<!\d)(\d{4}-\d{2}-\d{2})(?!\d)/g;

export function emptyMetadata(): Metadata {
  return { tags: new Set(), properties: new Map(), dates: new Set() };
}

export function parseMetadata(text: string): Metadata {
  const metadata = emptyMetadata();
  for (const match of text.matchAll(tagPattern)) metadata.tags.add(normalize(match[2]));
  for (const match of text.matchAll(propertyPattern)) {
    const key = normalize(match[2]);
    const raw = match[3].trim();
    metadata.properties.set(key, classifyValue(raw));
  }
  for (const match of text.matchAll(datePattern)) {
    if (isIsoDate(match[1])) metadata.dates.add(match[1]);
  }
  return metadata;
}

export function classifyValue(raw: string): MetadataValue {
  if (isIsoDate(raw)) return { type: "date", value: raw };
  if (/^[+-]?(?:\d+\.?\d*|\.\d+)$/.test(raw)) return { type: "number", value: Number(raw), raw };
  if (/^(?:true|false)$/i.test(raw)) return { type: "boolean", value: raw.toLowerCase() === "true", raw };
  return { type: "string", value: raw };
}

export function buildIndex(document: OutlineDocument): IndexedBlock[] {
  const result: IndexedBlock[] = [];
  const seen = new Set<string>();
  const visit = (id: string, inherited: Metadata, parentAncestors: string[]) => {
    if (seen.has(id)) throw new Error(`Duplicate or cyclic block reference: ${id}`);
    const block = document.blocks[id];
    if (!block) throw new Error(`Unknown block: ${id}`);
    seen.add(id);
    const local = parseMetadata(block.text);
    const effective = mergeMetadata(inherited, local);
    result.push({
      id,
      text: block.text,
      normalizedText: normalize(block.text),
      ancestors: parentAncestors,
      local,
      effective,
    });
    for (const childId of block.children) visit(childId, effective, [...parentAncestors, id]);
  };
  for (const rootId of document.roots) visit(rootId, emptyMetadata(), []);
  return result;
}

export function mergeMetadata(parent: Metadata, local: Metadata): Metadata {
  return {
    tags: new Set([...parent.tags, ...local.tags]),
    properties: new Map([...parent.properties, ...local.properties]),
    dates: new Set([...parent.dates, ...local.dates]),
  };
}

export function propertyValueText(value: MetadataValue): string {
  return value.type === "string" || value.type === "date" ? value.value : value.raw;
}

export function isIsoDate(value: string): boolean {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return false;
  const date = new Date(`${value}T00:00:00Z`);
  return (
    date.getUTCFullYear() === Number(match[1]) &&
    date.getUTCMonth() + 1 === Number(match[2]) &&
    date.getUTCDate() === Number(match[3])
  );
}

export const normalize = (value: string) => value.normalize("NFKC").toLocaleLowerCase();
