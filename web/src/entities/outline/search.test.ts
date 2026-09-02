import { describe, expect, it } from "vitest";
import { createDocument } from "./document";
import { buildIndex } from "./metadata";
import { parseQuery } from "./query";
import { buildAutocomplete, search } from "./search";
import { insertChild, insertRoot } from "./tree";

const fixed = (value: string) => () => value;

function fixture() {
  let document = insertRoot(createDocument(), "Work #work project::singlepage rating::9\n2026-09-01", 0, fixed("work")).document;
  document = insertChild(document, "work", "Encryption area::security", 0, fixed("encryption")).document;
  document = insertChild(document, "encryption", "WebCrypto leader election", 0, fixed("webcrypto")).document;
  document = insertRoot(document, "Archive #archive status::done rating::4\n2026-10-01", 1, fixed("archive")).document;
  return buildIndex(document);
}

describe("query and search", () => {
  it("parses combined and negated conditions", () => {
    expect(parseQuery('"leader election" #work -status:done rating:>=8 @2026-09-01..2026-09-30')).toEqual({
      text: [{ value: "leader election", negated: false }],
      tags: [{ value: "work", negated: false }],
      properties: [
        { key: "status", comparison: "=", value: "done", negated: true },
        { key: "rating", comparison: ">=", value: "8", negated: false },
      ],
      dates: [{ value: "2026-09-01", end: "2026-09-30", negated: false }],
    });
  });

  it("searches text and inherited metadata with AND semantics", () => {
    expect(search(fixture(), '"leader election" #work project:singlepage area:security rating:>8 @>=2026-09-01').map((x) => x.id))
      .toEqual(["webcrypto"]);
  });

  it("supports negation and date ranges", () => {
    expect(search(fixture(), "#work -status:done @2026-09-01..2026-09-30").map((x) => x.id))
      .toEqual(["work", "encryption", "webcrypto"]);
  });

  it("builds autocomplete only from user metadata", () => {
    expect(buildAutocomplete(fixture())).toEqual({
      tags: ["archive", "work"],
      properties: {
        area: ["security"],
        project: ["singlepage"],
        rating: ["4", "9"],
        status: ["done"],
      },
    });
  });
});
