import { cloneDocument, createBlock, randomId, type IdFactory, type OutlineDocument } from "./document";

function siblings(document: OutlineDocument, parentId: string | null): string[] {
  return parentId === null ? document.roots : document.blocks[parentId]?.children ?? [];
}

function assertBlock(document: OutlineDocument, blockId: string) {
  const block = document.blocks[blockId];
  if (!block) throw new Error(`Unknown block: ${blockId}`);
  return block;
}

export function insertRoot(
  document: OutlineDocument,
  text = "",
  index = document.roots.length,
  idFactory: IdFactory = randomId,
): { document: OutlineDocument; blockId: string } {
  const next = cloneDocument(document);
  const blockId = idFactory();
  if (next.blocks[blockId]) throw new Error(`Duplicate block: ${blockId}`);
  next.blocks[blockId] = createBlock(blockId, text);
  next.roots.splice(clampIndex(index, next.roots.length), 0, blockId);
  return { document: next, blockId };
}

export function insertAfter(
  document: OutlineDocument,
  referenceId: string,
  text = "",
  idFactory: IdFactory = randomId,
): { document: OutlineDocument; blockId: string } {
  const reference = assertBlock(document, referenceId);
  const next = cloneDocument(document);
  const blockId = idFactory();
  if (next.blocks[blockId]) throw new Error(`Duplicate block: ${blockId}`);
  next.blocks[blockId] = createBlock(blockId, text, reference.parentId);
  const list = siblings(next, reference.parentId);
  list.splice(list.indexOf(referenceId) + 1, 0, blockId);
  return { document: next, blockId };
}

export function appendSibling(
  document: OutlineDocument,
  referenceId: string,
  text = "",
  idFactory: IdFactory = randomId,
): { document: OutlineDocument; blockId: string } {
  const reference = assertBlock(document, referenceId);
  const next = cloneDocument(document);
  const blockId = idFactory();
  if (next.blocks[blockId]) throw new Error(`Duplicate block: ${blockId}`);
  next.blocks[blockId] = createBlock(blockId, text, reference.parentId);
  const list = siblings(next, reference.parentId);
  list.push(blockId);
  return { document: next, blockId };
}

export function insertChild(
  document: OutlineDocument,
  parentId: string,
  text = "",
  index = assertBlock(document, parentId).children.length,
  idFactory: IdFactory = randomId,
): { document: OutlineDocument; blockId: string } {
  assertBlock(document, parentId);
  const next = cloneDocument(document);
  const blockId = idFactory();
  if (next.blocks[blockId]) throw new Error(`Duplicate block: ${blockId}`);
  next.blocks[blockId] = createBlock(blockId, text, parentId);
  next.blocks[parentId].children.splice(clampIndex(index, next.blocks[parentId].children.length), 0, blockId);
  return { document: next, blockId };
}

export function removeBlock(document: OutlineDocument, blockId: string): OutlineDocument {
  const block = assertBlock(document, blockId);
  const next = cloneDocument(document);
  const list = siblings(next, block.parentId);
  list.splice(list.indexOf(blockId), 1);
  const remove = (id: string) => {
    for (const childId of next.blocks[id].children) remove(childId);
    delete next.blocks[id];
  };
  remove(blockId);
  return next;
}

export const remove = removeBlock;

export function indent(document: OutlineDocument, blockId: string): OutlineDocument {
  const block = assertBlock(document, blockId);
  const list = siblings(document, block.parentId);
  const index = list.indexOf(blockId);
  if (index <= 0) return document;
  return moveBlock(document, blockId, list[index - 1], Number.MAX_SAFE_INTEGER);
}

export function outdent(document: OutlineDocument, blockId: string): OutlineDocument {
  const block = assertBlock(document, blockId);
  if (block.parentId === null) return document;
  const parent = assertBlock(document, block.parentId);
  const parentSiblings = siblings(document, parent.parentId);
  return moveBlock(document, blockId, parent.parentId, parentSiblings.indexOf(parent.id) + 1);
}

export function moveBlock(
  document: OutlineDocument,
  blockId: string,
  newParentId: string | null,
  newIndex: number,
): OutlineDocument {
  const block = assertBlock(document, blockId);
  if (newParentId !== null) {
    assertBlock(document, newParentId);
    if (newParentId === blockId || descendants(document, blockId).includes(newParentId)) {
      throw new Error("Cannot move a block into its own subtree");
    }
  }
  const next = cloneDocument(document);
  const oldList = siblings(next, block.parentId);
  const oldIndex = oldList.indexOf(blockId);
  oldList.splice(oldIndex, 1);
  const newList = siblings(next, newParentId);
  let targetIndex = clampIndex(newIndex, newList.length);
  if (block.parentId === newParentId && oldIndex < newIndex) targetIndex = Math.max(0, targetIndex);
  newList.splice(targetIndex, 0, blockId);
  next.blocks[blockId].parentId = newParentId;
  return next;
}

export function reorder(document: OutlineDocument, blockId: string, newIndex: number): OutlineDocument {
  return moveBlock(document, blockId, assertBlock(document, blockId).parentId, newIndex);
}

export function ancestors(document: OutlineDocument, blockId: string): string[] {
  const result: string[] = [];
  let parentId = assertBlock(document, blockId).parentId;
  const seen = new Set<string>();
  while (parentId !== null) {
    if (seen.has(parentId)) throw new Error("Cycle detected in document");
    seen.add(parentId);
    result.unshift(parentId);
    parentId = assertBlock(document, parentId).parentId;
  }
  return result;
}

export function descendants(document: OutlineDocument, blockId: string): string[] {
  const result: string[] = [];
  const visit = (id: string) => {
    for (const child of assertBlock(document, id).children) {
      result.push(child);
      visit(child);
    }
  };
  visit(blockId);
  return result;
}

export function visibleBlockIds(document: OutlineDocument, focusedBlockId: string | null = null): string[] {
  const result: string[] = [];
  const visit = (id: string, includeSelf: boolean) => {
    const block = assertBlock(document, id);
    if (includeSelf) result.push(id);
    if (!block.collapsed) for (const child of block.children) visit(child, true);
  };
  if (focusedBlockId) visit(focusedBlockId, false);
  else for (const root of document.roots) visit(root, true);
  return result;
}

function clampIndex(index: number, length: number) {
  return Math.max(0, Math.min(index, length));
}
