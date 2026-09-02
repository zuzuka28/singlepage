export interface Block {
  id: string;
  parentId: string | null;
  children: string[];
  text: string;
  collapsed: boolean;
}

export interface OutlineDocument {
  formatVersion: 1;
  roots: string[];
  blocks: Record<string, Block>;
}

export type IdFactory = () => string;

export const randomId: IdFactory = () => crypto.randomUUID();

export function createDocument(): OutlineDocument {
  return { formatVersion: 1, roots: [], blocks: {} };
}

export function createBlock(id: string, text = "", parentId: string | null = null): Block {
  return { id, parentId, children: [], text, collapsed: false };
}

export function cloneDocument(document: OutlineDocument): OutlineDocument {
  const blocks = Object.fromEntries(
    Object.entries(document.blocks).map(([id, block]) => [id, { ...block, children: [...block.children] }]),
  );
  return { ...document, roots: [...document.roots], blocks };
}

export function updateBlock(
  document: OutlineDocument,
  blockId: string,
  patch: Partial<Pick<Block, "text" | "collapsed">>,
): OutlineDocument {
  const block = document.blocks[blockId];
  if (!block) throw new Error(`Unknown block: ${blockId}`);
  return { ...document, blocks: { ...document.blocks, [blockId]: { ...block, ...patch } } };
}

export function serializeDocument(document: OutlineDocument): string {
  return JSON.stringify(document);
}

export function deserializeDocument(value: string): OutlineDocument {
  const parsed = JSON.parse(value) as Partial<OutlineDocument>;
  if (parsed.formatVersion !== 1 || !Array.isArray(parsed.roots) || !parsed.blocks) {
    throw new Error("Unsupported or invalid document");
  }
  return parsed as OutlineDocument;
}
