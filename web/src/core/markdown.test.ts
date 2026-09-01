import { describe, expect, it } from "vitest";
import { createDocument } from "./document";
import { insertChild, insertRoot } from "./tree";
import { parseMarkdown, serializeMarkdown } from "./markdown";

const ids = (...values: string[]) => {
  let index = 0;
  return () => values[index++];
};

describe("Markdown import and export", () => {
  it("serializes a nested outline as a Markdown list", () => {
    let document = insertRoot(createDocument(), "Project #work", 0, ids("root")).document;
    document = insertChild(document, "root", "First line\nsecond line", 0, ids("child")).document;
    document = insertChild(document, "child", "Nested", 0, ids("nested")).document;

    expect(serializeMarkdown(document)).toBe(
      "- Project #work\n  - First line\n    second line\n    - Nested\n",
    );
  });

  it("parses unordered and ordered Markdown lists into an outline", () => {
    const document = parseMarkdown(
      "- First\n    1. Child\n    2. Sibling\n- Second\n",
      ids("first", "child", "sibling", "second"),
    );

    expect(document.roots).toEqual(["first", "second"]);
    expect(document.blocks.first.text).toBe("First");
    expect(document.blocks.first.children).toEqual(["child", "sibling"]);
    expect(document.blocks.child.parentId).toBe("first");
  });

  it("round-trips multiline content that looks like nested Markdown", () => {
    let source = insertRoot(createDocument(), "Line one\n\n- still text\n\\- escaped text", 0, ids("source")).document;
    source = insertChild(source, "source", "Actual child", 0, ids("source-child")).document;

    const restored = parseMarkdown(serializeMarkdown(source), ids("restored", "restored-child"));

    expect(restored.blocks.restored.text).toBe("Line one\n\n- still text\n\\- escaped text");
    expect(restored.blocks.restored.children).toEqual(["restored-child"]);
    expect(restored.blocks["restored-child"].text).toBe("Actual child");
  });

  it("imports plain text as top-level blocks", () => {
    const document = parseMarkdown("First paragraph\n\nSecond paragraph", ids("first", "second"));

    expect(document.roots).toEqual(["first", "second"]);
    expect(document.blocks.first.text).toBe("First paragraph");
    expect(document.blocks.second.text).toBe("Second paragraph");
  });
});
