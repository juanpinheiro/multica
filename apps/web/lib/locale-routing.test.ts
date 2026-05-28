import { describe, expect, it } from "vitest";
import {
  isSupportedLocale,
  resolveLocaleFromSignals,
} from "./locale-routing";

describe("locale routing", () => {
  it("accepts only app-supported locale identifiers", () => {
    expect(isSupportedLocale("en")).toBe(true);
    expect(isSupportedLocale("zh-Hans")).toBe(false);
    expect(isSupportedLocale("zh")).toBe(false);
    expect(isSupportedLocale(null)).toBe(false);
  });

  it("defaults to English for any locale", () => {
    expect(
      resolveLocaleFromSignals({
        cookieLocale: "en",
        acceptLanguage: "en-US,en;q=0.9",
      }),
    ).toBe("en");
  });

  it("falls back to English when no cookie is set", () => {
    expect(
      resolveLocaleFromSignals({
        acceptLanguage: "en;q=0.8",
      }),
    ).toBe("en");
  });
});
