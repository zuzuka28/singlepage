import { describe, expect, it } from "vitest";
import { createDocument } from "./document";
import { ancestors, appendSibling, indent, insertAfter, insertChild, insertRoot, outdent, removeBlock, reorder } from "./tree";

const ids = (...values: string[]) => {
  let index = 0;
  return () => values[index++];
};

describe("tree operations", () => {
  it("inserts root, sibling, and child blocks", () => {
    let document = insertRoot(createDocument(), "one", 0, ids("a")).document;
    document = insertAfter(document, "a", "two", ids("b")).document;
    document = insertChild(document, "a", "child", 0, ids("c")).document;
    expect(document.roots).toEqual(["a", "b"]);
    expect(document.blocks.a.children).toEqual(["c"]);
    expect(document.blocks.c.parentId).toBe("a");
  });

  it("appends new siblings to the end of their current level", () => {
    let document = insertRoot(createDocument(), "first", 0, ids("first")).document;
    document = insertAfter(document, "first", "last", ids("last")).document;
    document = appendSibling(document, "first", "new root", ids("new-root")).document;
    document = insertChild(document, "first", "first child", 0, ids("first-child")).document;
    document = insertChild(document, "first", "last child", 1, ids("last-child")).document;
    document = appendSibling(document, "first-child", "new child", ids("new-child")).document;

    expect(document.roots).toEqual(["first", "last", "new-root"]);
    expect(document.blocks.first.children).toEqual(["first-child", "last-child", "new-child"]);
    expect(document.blocks["new-child"].parentId).toBe("first");
  });

  it("inserts directly after a block when editing a list", () => {
    let document = insertRoot(createDocument(), "first", 0, ids("first")).document;
    document = insertAfter(document, "first", "last", ids("last")).document;
    document = insertAfter(document, "first", "middle", ids("middle")).document;
    expect(document.roots).toEqual(["first", "middle", "last"]);
  });

  it("indents under the previous sibling and outdents after its parent", () => {
    let document = insertRoot(createDocument(), "a", 0, ids("a")).document;
    document = insertAfter(document, "a", "b", ids("b")).document;
    document = indent(document, "b");
    expect(document.roots).toEqual(["a"]);
    expect(document.blocks.a.children).toEqual(["b"]);
    document = outdent(document, "b");
    expect(document.roots).toEqual(["a", "b"]);
  });

  it("removes a whole subtree", () => {
    let document = insertRoot(createDocument(), "a", 0, ids("a")).document;
    document = insertChild(document, "a", "b", 0, ids("b")).document;
    document = insertChild(document, "b", "c", 0, ids("c")).document;
    document = removeBlock(document, "b");
    expect(document.blocks).toEqual({ a: expect.anything() });
    expect(document.blocks.a.children).toEqual([]);
  });

  it("reorders siblings and traverses ancestors root-first", () => {
    let document = insertRoot(createDocument(), "a", 0, ids("a")).document;
    document = insertAfter(document, "a", "b", ids("b")).document;
    document = insertAfter(document, "b", "c", ids("c")).document;
    document = reorder(document, "c", 0);
    document = insertChild(document, "a", "d", 0, ids("d")).document;
    document = insertChild(document, "d", "e", 0, ids("e")).document;
    expect(document.roots).toEqual(["c", "a", "b"]);
    expect(ancestors(document, "e")).toEqual(["a", "d"]);
  });

  it("refuses cycles", () => {
    let document = insertRoot(createDocument(), "a", 0, ids("a")).document;
    document = insertChild(document, "a", "b", 0, ids("b")).document;
    expect(() => indent(document, "a")).not.toThrow();
  });
});
