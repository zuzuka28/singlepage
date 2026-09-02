import { createBlock, createDocument, randomId, type IdFactory, type OutlineDocument } from "./document";

const LIST_ITEM = /^([ \t]*)(?:[-+*]|\d+[.)])(?:[ \t]+(.*))?$/;
const LIST_MARKER = /^(\s*)(\\*)([-+*](?:[ \t]|$)|\d+[.)](?:[ \t]|$))/;

export function serializeMarkdown(document: OutlineDocument): string {
  const lines: string[] = [];

  const visit = (blockId: string, depth: number) => {
    const block = document.blocks[blockId];
    if (!block) return;

    const indent = "  ".repeat(depth);
    const textLines = block.text.split("\n");
    lines.push(`${indent}- ${textLines[0] ?? ""}`);
    for (const line of textLines.slice(1)) {
      lines.push(`${indent}  ${escapeContinuation(line)}`);
    }
    for (const childId of block.children) visit(childId, depth + 1);
  };

  for (const rootId of document.roots) visit(rootId, 0);
  return lines.length ? `${lines.join("\n")}\n` : "";
}

export function parseMarkdown(value: string, idFactory: IdFactory = randomId): OutlineDocument {
  const document = createDocument();
  const parents: Array<{ indent: number; id: string }> = [];
  let currentId: string | null = null;
  let currentListIndent: number | null = null;

  const appendRoot = (text: string) => {
    const id = nextId(document, idFactory);
    document.blocks[id] = createBlock(id, text);
    document.roots.push(id);
    currentId = id;
    currentListIndent = null;
  };

  const lines = value.replace(/\r\n?/g, "\n").split("\n");
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    const item = LIST_ITEM.exec(line);
    if (item) {
      const indent = indentationWidth(item[1]);
      while (parents.length && indent <= parents[parents.length - 1].indent) parents.pop();
      const parentId = parents.at(-1)?.id ?? null;
      const id = nextId(document, idFactory);
      document.blocks[id] = createBlock(id, item[2] ?? "", parentId);
      if (parentId) document.blocks[parentId].children.push(id);
      else document.roots.push(id);
      parents.push({ indent, id });
      currentId = id;
      currentListIndent = indent;
      continue;
    }

    const width = indentationWidth(line.match(/^[ \t]*/)?.[0] ?? "");
    if (currentId && currentListIndent !== null && width > currentListIndent) {
      const continuationIndent = currentListIndent + 2;
      const continuation = unescapeContinuation(stripIndent(line, continuationIndent));
      document.blocks[currentId].text += `\n${continuation}`;
      continue;
    }

    parents.length = 0;
    if (line.trim() === "") {
      currentId = null;
      currentListIndent = null;
      continue;
    }

    if (currentId && currentListIndent === null) document.blocks[currentId].text += `\n${line}`;
    else appendRoot(line);
  }

  return document;
}

function escapeContinuation(line: string): string {
  return line.replace(LIST_MARKER, (_match, whitespace: string, slashes: string, marker: string) =>
    `${whitespace}\\${slashes}${marker}`,
  );
}

function unescapeContinuation(line: string): string {
  return line.replace(/^(\s*)\\(\\*(?:[-+*](?:[ \t]|$)|\d+[.)](?:[ \t]|$)))/, "$1$2");
}

function indentationWidth(value: string): number {
  return value.replace(/\t/g, "    ").length;
}

function stripIndent(value: string, width: number): string {
  let remaining = width;
  let index = 0;
  while (index < value.length && remaining > 0) {
    if (value[index] === " ") remaining -= 1;
    else if (value[index] === "\t") remaining -= 4;
    else break;
    index += 1;
  }
  return value.slice(index);
}

function nextId(document: OutlineDocument, idFactory: IdFactory): string {
  const id = idFactory();
  if (document.blocks[id]) throw new Error(`Duplicate block: ${id}`);
  return id;
}
