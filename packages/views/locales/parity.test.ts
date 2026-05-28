import { readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { RESOURCES } from "./index";

// Schema-level guard: every EN namespace registered in RESOURCES corresponds
// to a JSON file on disk. Catches drift where a file is deleted but not
// unregistered, or vice versa.

const LOCALES_DIR = dirname(fileURLToPath(import.meta.url));

function jsonNamespacesIn(locale: string): string[] {
  return readdirSync(resolve(LOCALES_DIR, locale))
    .filter((name) => name.endsWith(".json"))
    .map((name) => name.replace(/\.json$/, ""))
    .sort();
}

const en = RESOURCES.en;

describe("locale bundle consistency", () => {
  it("registers every JSON file in RESOURCES (EN)", () => {
    expect(Object.keys(en).sort()).toEqual(jsonNamespacesIn("en"));
  });
});
