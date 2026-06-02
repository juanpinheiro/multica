import { describe, expect, it } from "vitest";
import { latestState } from "./queries";

describe("latestState", () => {
  it("returns empty state for empty input", () => {
    const s = latestState([]);
    expect(s.done).toEqual([]);
    expect(s.left_undone).toEqual([]);
    expect(s.discoveries).toEqual([]);
  });

  it("returns the single handoff's fields", () => {
    const s = latestState([
      { done: ["step A"], left_undone: ["step B"], discoveries: ["bug"] },
    ]);
    expect(s.done).toEqual(["step A"]);
    expect(s.left_undone).toEqual(["step B"]);
    expect(s.discoveries).toEqual(["bug"]);
  });

  it("returns the LAST handoff when multiple are given", () => {
    const s = latestState([
      { done: ["step A"], left_undone: ["step B"], discoveries: [] },
      { done: ["step A", "step B"], left_undone: [], discoveries: ["discovery X"] },
    ]);
    expect(s.done).toEqual(["step A", "step B"]);
    expect(s.left_undone).toEqual([]);
    expect(s.discoveries).toEqual(["discovery X"]);
  });

  it("treats null/undefined fields as empty arrays", () => {
    const s = latestState([
      { done: null as unknown as string[], left_undone: undefined as unknown as string[], discoveries: null as unknown as string[] },
    ]);
    expect(s.done).toEqual([]);
    expect(s.left_undone).toEqual([]);
    expect(s.discoveries).toEqual([]);
  });
});
