import {
  ancestors,
  appendSibling,
  buildAutocomplete,
  buildIndex,
  descendants,
  indent,
  insertAfter,
  insertChild,
  moveBlock,
  outdent,
  removeBlock,
  reorder,
  search,
  updateBlock,
  type OutlineDocument,
} from '../../../entities/outline';
import type { EditorCommandResult, FocusIntent } from './model';

export interface EditorKey {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  alt?: boolean;
  composing?: boolean;
  selectionStart?: number;
  selectionEnd?: number;
}

export interface EditorContext {
  document: OutlineDocument;
  id: string;
  focusedId: string | null;
  query: string;
  visibleIds: string[];
}

export function interpretEditorKey(context: EditorContext, key: EditorKey): EditorCommandResult | null {
  if (key.composing) return null;
  const { document, id } = context;
  const block = document.blocks[id];
  if (!block) return null;

  if (key.key === 'Enter' && (key.meta || key.ctrl)) {
    const inserted = insertChild(document, id);
    return changed(inserted.document, inserted.blockId);
  }
  if (key.key === 'Enter' && !key.shift) {
    const topLevelFilterResult = Boolean(context.query.trim()) && block.parentId === context.focusedId;
    const inserted = topLevelFilterResult ? appendSibling(document, id) : insertAfter(document, id);
    return changed(inserted.document, inserted.blockId);
  }
  if (key.key === 'Tab') {
    return changed(key.shift ? outdent(document, id) : indent(document, id), id);
  }
  if (key.key === 'Backspace' && block.text === '' && block.children.length === 0 && Object.keys(document.blocks).length > 1) {
    const previous = context.visibleIds[Math.max(0, context.visibleIds.indexOf(id) - 1)];
    return changed(removeBlock(document, id), previous);
  }
  if ((key.meta || key.ctrl) && (key.key === 'ArrowUp' || key.key === 'ArrowDown')) {
    const siblings = block.parentId ? document.blocks[block.parentId].children : document.roots;
    const offset = key.key === 'ArrowUp' ? -1 : 1;
    return changed(reorder(document, id, siblings.indexOf(id) + offset), id);
  }
  if (!key.shift && !key.alt && !key.meta && !key.ctrl && (key.key === 'ArrowUp' || key.key === 'ArrowDown')) {
    const collapsed = key.selectionStart === key.selectionEnd;
    const atBoundary = key.key === 'ArrowUp' ? key.selectionStart === 0 : key.selectionStart === block.text.length;
    if (collapsed && atBoundary) {
      const offset = key.key === 'ArrowUp' ? -1 : 1;
      const target = context.visibleIds[context.visibleIds.indexOf(id) + offset];
      if (target) return { document, focusIntent: { id: target, caret: key.key === 'ArrowUp' ? 'end' : 'start' } };
    }
  }
  return null;
}

function changed(document: OutlineDocument, id?: string): EditorCommandResult {
  return { document, focusIntent: id ? { id } : null };
}

export interface OutlineView {
  index: ReturnType<typeof buildIndex>;
  results: ReturnType<typeof search>;
  autocomplete: ReturnType<typeof buildAutocomplete>;
  suggestions: string[];
  rootIds: string[];
  displayRootIds: string[];
  matchedIds: Set<string>;
  filteredIds: Set<string> | null;
  trail: string[];
  visibleIds: string[];
}

export function selectOutlineView(
  document: OutlineDocument,
  focusedId: string | null,
  query: string,
  filterVisibleExtras: ReadonlySet<string> = new Set(),
): OutlineView {
  const index = buildIndex(document);
  const results = query.trim() ? search(index, query) : [];
  const autocomplete = buildAutocomplete(index);
  const rootIds = focusedId ? (document.blocks[focusedId]?.children ?? []) : document.roots;
  const matchedIds = new Set(results.map((result) => result.id));
  const filteredIds = query.trim()
    ? new Set([
        ...results.flatMap((result) => [...result.ancestors, result.id, ...descendants(document, result.id)]),
        ...filterVisibleExtras,
      ])
    : null;
  const displayRootIds = rootIds.filter((id) => !filteredIds || filteredIds.has(id));
  const trail = focusedId ? [...ancestors(document, focusedId), focusedId] : [];
  const visibleIds = flattenVisibleIds(document, rootIds, filteredIds, Boolean(query.trim()));
  return {
    index,
    results,
    autocomplete,
    suggestions: suggestionsFor(query, autocomplete),
    rootIds,
    displayRootIds,
    matchedIds,
    filteredIds,
    trail,
    visibleIds,
  };
}

export function flattenVisibleIds(
  document: OutlineDocument,
  rootIds: readonly string[],
  filteredIds: ReadonlySet<string> | null,
  searching: boolean,
): string[] {
  const ids: string[] = [];
  const visit = (id: string) => {
    if (filteredIds && !filteredIds.has(id)) return;
    const block = document.blocks[id];
    if (!block) return;
    ids.push(id);
    if (searching || !block.collapsed) block.children.forEach(visit);
  };
  rootIds.forEach(visit);
  return ids;
}

export function suggestionsFor(value: string, autocomplete: ReturnType<typeof buildAutocomplete>): string[] {
  if (!value) return [];
  const token = value.split(/\s+/).at(-1) ?? '';
  const normalized = token.toLocaleLowerCase();
  if (normalized.startsWith('#')) {
    return autocomplete.tags
      .filter((tag) => tag.startsWith(normalized.slice(1)))
      .slice(0, 6)
      .map((tag) => `#${tag}`)
      .filter((item) => item !== normalized);
  }
  if (normalized.startsWith('@')) {
    return ['@2026-09-01', '@>=2026-09-01', '@2026-09-01..2026-09-30'].filter((item) => item !== normalized);
  }
  const colon = normalized.indexOf(':');
  if (colon >= 0) {
    const key = normalized.slice(0, colon);
    const prefix = normalized.slice(colon + 1);
    return (autocomplete.properties[key] ?? [])
      .filter((item) => item.toLocaleLowerCase().startsWith(prefix))
      .slice(0, 6)
      .map((item) => `${key}:${item}`)
      .filter((item) => item.toLocaleLowerCase() !== normalized);
  }
  return Object.keys(autocomplete.properties)
    .filter((key) => key.startsWith(normalized))
    .slice(0, 6)
    .map((key) => `${key}:`);
}

export function expandPath(document: OutlineDocument, id: string): EditorCommandResult {
  let next = document;
  for (const blockId of [...ancestors(document, id), id]) {
    if (next.blocks[blockId]?.collapsed) next = updateBlock(next, blockId, { collapsed: false });
  }
  return { document: next, focusIntent: { id, caret: 'end' } };
}

export function keepVisible(document: OutlineDocument, id: string, current: ReadonlySet<string>): Set<string> {
  return new Set([...current, ...ancestors(document, id), id]);
}

export function focusIntent(id: string | undefined, caret: 'start' | 'end' | number = 'end'): FocusIntent {
  return id ? { id, caret } : null;
}

export type DropPosition = 'before' | 'inside' | 'after';

export function branchIds(document: OutlineDocument, id: string): string[] {
  return document.blocks[id] ? [id, ...descendants(document, id)] : [];
}

export function expandSelection(
  document: OutlineDocument,
  id: string,
  selectedIds: ReadonlySet<string>,
): Set<string> {
  if (!document.blocks[id]) return new Set();
  const candidates = [
    new Set([id]),
    ...ancestors(document, id).reverse().map((ancestorId) => new Set(branchIds(document, ancestorId))),
    new Set(Object.keys(document.blocks)),
  ].filter((candidate, index, values) => !values.slice(0, index).some((previous) => sameIds(previous, candidate)));
  const currentIndex = candidates.findIndex((candidate) => sameIds(candidate, selectedIds));
  return new Set(candidates[Math.min(currentIndex + 1, candidates.length - 1)] ?? [id]);
}

export function removeSelectedBlocks(
  document: OutlineDocument,
  selectedIds: ReadonlySet<string>,
): OutlineDocument {
  const selectedRoots = orderedSelectionRoots(document, selectedIds);
  return selectedRoots.reduce(
    (next, id) => next.blocks[id] ? removeBlock(next, id) : next,
    document,
  );
}

export function moveSelectedRelative(
  document: OutlineDocument,
  selectedIds: ReadonlySet<string>,
  targetId: string,
  position: DropPosition,
): OutlineDocument | null {
  const selectedRoots = orderedSelectionRoots(document, selectedIds);
  if (!selectedRoots.length || selectedRoots.some((id) => id === targetId || descendants(document, id).includes(targetId))) return null;
  const moveOrder = position === 'after' ? [...selectedRoots].reverse() : selectedRoots;
  let next = document;
  for (const id of moveOrder) {
    const moved = moveBlockRelative(next, id, targetId, position);
    if (!moved) return null;
    next = moved;
  }
  return next;
}

export function indentSelected(
  document: OutlineDocument,
  selectedIds: ReadonlySet<string>,
): OutlineDocument {
  return orderedSelectionRoots(document, selectedIds).reduce(
    (next, id) => next.blocks[id] ? indent(next, id) : next,
    document,
  );
}

export function outdentSelected(
  document: OutlineDocument,
  selectedIds: ReadonlySet<string>,
): OutlineDocument {
  return orderedSelectionRoots(document, selectedIds).reverse().reduce(
    (next, id) => next.blocks[id] ? outdent(next, id) : next,
    document,
  );
}

export function moveBlockRelative(
  document: OutlineDocument,
  sourceId: string,
  targetId: string,
  position: DropPosition,
): OutlineDocument | null {
  const source = document.blocks[sourceId];
  const target = document.blocks[targetId];
  if (!source || !target || sourceId === targetId || descendants(document, sourceId).includes(targetId)) return null;

  if (position === 'inside') {
    const moved = moveBlock(document, sourceId, targetId, target.children.length);
    return moved.blocks[targetId].collapsed ? updateBlock(moved, targetId, { collapsed: false }) : moved;
  }

  const targetSiblings = target.parentId === null ? document.roots : document.blocks[target.parentId].children;
  const sourceSiblings = source.parentId === null ? document.roots : document.blocks[source.parentId].children;
  const targetIndex = targetSiblings.indexOf(targetId);
  const sourceBeforeTarget = sourceSiblings === targetSiblings && sourceSiblings.indexOf(sourceId) < targetIndex;
  const newIndex = position === 'before'
    ? targetIndex - (sourceBeforeTarget ? 1 : 0)
    : targetIndex + (sourceBeforeTarget ? 0 : 1);
  return moveBlock(document, sourceId, target.parentId, newIndex);
}

function orderedSelectionRoots(document: OutlineDocument, selectedIds: ReadonlySet<string>): string[] {
  const ordered: string[] = [];
  const visit = (id: string, selectedAncestor: boolean) => {
    const selected = selectedIds.has(id);
    if (selected && !selectedAncestor) ordered.push(id);
    document.blocks[id]?.children.forEach((childId) => visit(childId, selectedAncestor || selected));
  };
  document.roots.forEach((id) => visit(id, false));
  return ordered;
}

function sameIds(left: ReadonlySet<string>, right: ReadonlySet<string>): boolean {
  return left.size === right.size && [...left].every((id) => right.has(id));
}
