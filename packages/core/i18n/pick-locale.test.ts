import { describe, expect, it } from "vitest";
import { matchLocale, pickLocale } from "./pick-locale";
import type { LocaleAdapter } from "./types";

function makeAdapter(
  overrides: Partial<LocaleAdapter> = {},
): LocaleAdapter {
  return {
    getUserChoice: () => null,
    getSystemPreferences: () => [],
    persist: () => {},
    ...overrides,
  };
}

describe("matchLocale", () => {
  it("returns DEFAULT_LOCALE when given an empty list", () => {
    expect(matchLocale([])).toBe("en");
  });

  it("matches a clean supported tag", () => {
    expect(matchLocale(["en"])).toBe("en");
  });

  it("collapses region-tagged BCP-47 to the supported base", () => {
    expect(matchLocale(["en-US"])).toBe("en");
  });

  it("falls back to DEFAULT_LOCALE when no candidate matches", () => {
    expect(matchLocale(["fr", "ja", "ko"])).toBe("en");
  });

  it("returns DEFAULT_LOCALE for unsupported locales", () => {
    expect(matchLocale(["zh-Hans"])).toBe("en");
    expect(matchLocale(["zh-Hant"])).toBe("en");
  });

  it("returns DEFAULT_LOCALE for malformed BCP-47 tags rather than throwing", () => {
    expect(matchLocale(["----"])).toBe("en");
    expect(matchLocale(["x-private-only"])).toBe("en");
  });
});

describe("pickLocale", () => {
  it("prefers explicit user choice over system signal", () => {
    const adapter = makeAdapter({
      getUserChoice: () => "en",
      getSystemPreferences: () => ["en-US"],
    });
    expect(pickLocale(adapter)).toBe("en");
  });

  it("falls back to system preferences when no user choice", () => {
    const adapter = makeAdapter({
      getSystemPreferences: () => ["en-US"],
    });
    expect(pickLocale(adapter)).toBe("en");
  });

  it("returns DEFAULT_LOCALE when neither choice nor preference yields a match", () => {
    const adapter = makeAdapter({
      getUserChoice: () => null,
      getSystemPreferences: () => ["fr", "ja"],
    });
    expect(pickLocale(adapter)).toBe("en");
  });

  it("ignores empty-string user choice and falls through to system", () => {
    const adapter = makeAdapter({
      getUserChoice: () => "",
      getSystemPreferences: () => ["en-US"],
    });
    expect(pickLocale(adapter)).toBe("en");
  });
});
