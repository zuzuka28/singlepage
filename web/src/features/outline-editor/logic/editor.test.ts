import { describe, expect, it } from 'vitest';
import { createDocument, insertChild, insertRoot, updateBlock } from '../../../entities/outline';
import {
  expandSelection,
  indentSelected,
  interpretEditorKey,
  moveBlockRelative,
  moveSelectedRelative,
  outdentSelected,
  removeSelectedBlocks,
  selectOutlineView,
} from './editor';

describe('outline commands and selectors', () => {
  it('interprets Enter, Ctrl+Enter, Tab, Backspace and reorder without DOM access', () => {
    const first = insertRoot(createDocument(), '', 0, () => 'first');
    const base = updateBlock(first.document, 'first', { text: 'first' });
    const context = { document: base, id: 'first', focusedId: null, query: '', visibleIds: ['first'] };
    const enter = interpretEditorKey(context, { key: 'Enter' })!;
    expect(enter.document.roots).toHaveLength(2);
    const child = interpretEditorKey(context, { key: 'Enter', ctrl: true })!;
    expect(child.document.blocks.first.children).toHaveLength(1);

    const withChild = insertChild(base, 'first', '', 0, () => 'child').document;
    const tab = interpretEditorKey(
      { document: withChild, id: 'child', focusedId: null, query: '', visibleIds: ['first', 'child'] },
      { key: 'Tab', shift: true },
    )!;
    expect(tab.document.blocks.child.parentId).toBeNull();

    const emptyChild = updateBlock(withChild, 'child', { text: '' });
    const removed = interpretEditorKey(
      { document: emptyChild, id: 'child', focusedId: null, query: '', visibleIds: ['first', 'child'] },
      { key: 'Backspace' },
    )!;
    expect(removed.document.blocks.child).toBeUndefined();
  });

  it('derives filtering, breadcrumbs, visible ids and autocomplete in one selector', () => {
    const root = insertRoot(createDocument(), '', 0, () => 'root');
    const child = insertChild(root.document, 'root', '', 0, () => 'child');
    const document = updateBlock(updateBlock(child.document, 'root', { text: 'Root #work' }), 'child', { text: 'status::open' });
    const view = selectOutlineView(document, 'root', 'status:open');
    expect(view.displayRootIds).toEqual(['child']);
    expect(view.trail).toEqual(['root']);
    expect(view.visibleIds).toEqual(['child']);
    expect(view.autocomplete.tags).toContain('work');
  });

  it('selects whole branches and removes only the selected branch roots', () => {
    const root = insertRoot(createDocument(), 'Root', 0, () => 'root');
    const child = insertChild(root.document, 'root', 'Child', 0, () => 'child');
    const grandchild = insertChild(child.document, 'child', 'Grandchild', 0, () => 'grandchild');
    const sibling = insertRoot(grandchild.document, 'Sibling', 1, () => 'sibling');
    const selected = new Set(['root', 'child', 'grandchild']);

    expect(removeSelectedBlocks(sibling.document, selected).roots).toEqual(['sibling']);
  });

  it('expands repeated select-all from the current block through its ancestors', () => {
    const root = insertRoot(createDocument(), 'Root', 0, () => 'root');
    const child = insertChild(root.document, 'root', 'Child', 0, () => 'child');
    const grandchild = insertChild(child.document, 'child', 'Grandchild', 0, () => 'grandchild');
    const sibling = insertChild(grandchild.document, 'root', 'Sibling', 1, () => 'sibling');
    const otherRoot = insertRoot(sibling.document, 'Other', 1, () => 'other-root');

    const current = expandSelection(otherRoot.document, 'grandchild', new Set());
    expect([...current]).toEqual(['grandchild']);
    const parent = expandSelection(otherRoot.document, 'grandchild', current);
    expect(parent).toEqual(new Set(['child', 'grandchild']));
    const rootBranch = expandSelection(otherRoot.document, 'grandchild', parent);
    expect(rootBranch).toEqual(new Set(['root', 'child', 'grandchild', 'sibling']));
    expect(expandSelection(otherRoot.document, 'grandchild', rootBranch)).toEqual(new Set(Object.keys(otherRoot.document.blocks)));
  });

  it('moves a subtree before, after, or inside another block without allowing cycles', () => {
    const first = insertRoot(createDocument(), 'First', 0, () => 'first');
    const second = insertRoot(first.document, 'Second', 1, () => 'second');
    const third = insertRoot(second.document, 'Third', 2, () => 'third');
    const child = insertChild(third.document, 'first', 'Child', 0, () => 'child');

    const before = moveBlockRelative(child.document, 'first', 'third', 'before');
    expect(before?.roots).toEqual(['second', 'first', 'third']);
    expect(before?.blocks.first.children).toEqual(['child']);

    const after = moveBlockRelative(before!, 'first', 'third', 'after');
    expect(after?.roots).toEqual(['second', 'third', 'first']);

    const inside = moveBlockRelative(after!, 'third', 'second', 'inside');
    expect(inside?.roots).toEqual(['second', 'first']);
    expect(inside?.blocks.second.children).toEqual(['third']);
    expect(moveBlockRelative(inside!, 'second', 'third', 'inside')).toBeNull();
  });

  it('moves selected branches as one ordered group', () => {
    const first = insertRoot(createDocument(), 'First', 0, () => 'first');
    const second = insertRoot(first.document, 'Second', 1, () => 'second');
    const third = insertRoot(second.document, 'Third', 2, () => 'third');
    const fourth = insertRoot(third.document, 'Fourth', 3, () => 'fourth');
    const selected = new Set(['second', 'fourth']);

    const before = moveSelectedRelative(fourth.document, selected, 'first', 'before');
    expect(before?.roots).toEqual(['second', 'fourth', 'first', 'third']);
    const inside = moveSelectedRelative(before!, selected, 'third', 'inside');
    expect(inside?.roots).toEqual(['first', 'third']);
    expect(inside?.blocks.third.children).toEqual(['second', 'fourth']);
  });

  it('indents and outdents selected sibling branches together', () => {
    const first = insertRoot(createDocument(), 'First', 0, () => 'first');
    const second = insertRoot(first.document, 'Second', 1, () => 'second');
    const third = insertRoot(second.document, 'Third', 2, () => 'third');
    const selected = new Set(['second', 'third']);

    const indented = indentSelected(third.document, selected);
    expect(indented.roots).toEqual(['first']);
    expect(indented.blocks.first.children).toEqual(['second', 'third']);

    const outdented = outdentSelected(indented, selected);
    expect(outdented.roots).toEqual(['first', 'second', 'third']);
    expect(outdented.blocks.first.children).toEqual([]);
  });
});
