import { describe, expect, it } from "vitest";
import { createDocument } from "./document";
import { buildIndex, parseMetadata } from "./metadata";
import { insertChild, insertRoot } from "./tree";

const fixed = (value: string) => () => value;

describe("metadata", () => {
  it("parses Unicode tags, typed properties, and valid ISO dates", () => {
    const metadata = parseMetadata("Книга #Чтение rating::9 published::2026-09-01 active::true");
    expect([...metadata.tags]).toEqual(["чтение"]);
    expect(metadata.properties.get("rating")).toMatchObject({ type: "number", value: 9 });
    expect(metadata.properties.get("published")).toEqual({ type: "date", value: "2026-09-01" });
    expect(metadata.properties.get("active")).toMatchObject({ type: "boolean", value: true });
    expect([...metadata.dates]).toEqual(["2026-09-01"]);
    expect([...parseMetadata("bad 2026-02-31").dates]).toEqual([]);
  });

  it("parses multiple compact properties on one line", () => {
    const metadata = parseMetadata("project::singlepage area::security");

    expect(metadata.properties.get("project")).toEqual({ type: "string", value: "singlepage" });
    expect(metadata.properties.get("area")).toEqual({ type: "string", value: "security" });
  });

  it("rejects whitespace around the property separator", () => {
    expect(parseMetadata("project :: singlepage").properties.size).toBe(0);
    expect(parseMetadata("project:: singlepage").properties.size).toBe(0);
  });

  it("builds effective metadata with union and nearest property wins", () => {
    let document = insertRoot(createDocument(), "Work #work language::go\n2026-09-01", 0, fixed("a")).document;
    document = insertChild(document, "a", "Prototype #research language::rust\n2026-09-02", 0, fixed("b")).document;
    document = insertChild(document, "b", "WAL", 0, fixed("c")).document;
    const wal = buildIndex(document).find((block) => block.id === "c")!;
    expect([...wal.effective.tags]).toEqual(["work", "research"]);
    expect(wal.effective.properties.get("language")).toEqual({ type: "string", value: "rust" });
    expect([...wal.effective.dates]).toEqual(["2026-09-01", "2026-09-02"]);
    expect(wal.ancestors).toEqual(["a", "b"]);
  });
});
