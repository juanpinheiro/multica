import { describe, expect, it } from "vitest";
import { failedAssertions, milestoneSatisfied } from "./index";

const a = (id: string) => ({ id });
const r = (assertionId: string, passed: boolean) => ({ assertionId, passed });

describe("milestoneSatisfied", () => {
  it("is vacuously satisfied with no assertions", () => {
    expect(milestoneSatisfied([], [])).toBe(true);
  });

  it("is satisfied when every assertion has a passing verdict", () => {
    expect(milestoneSatisfied([a("x"), a("y")], [r("x", true), r("y", true)])).toBe(true);
  });

  it("is unsatisfied when an assertion has no verdict", () => {
    expect(milestoneSatisfied([a("x"), a("y")], [r("x", true)])).toBe(false);
  });

  it("is unsatisfied when an assertion fails", () => {
    expect(milestoneSatisfied([a("x")], [r("x", false)])).toBe(false);
  });

  it("treats any failing verdict for an assertion as a failure", () => {
    expect(milestoneSatisfied([a("x")], [r("x", true), r("x", false)])).toBe(false);
  });

  it("ignores stray verdicts for unknown assertions", () => {
    expect(milestoneSatisfied([a("x")], [r("x", true), r("ghost", false)])).toBe(true);
  });
});

describe("failedAssertions", () => {
  it("returns the unsatisfied assertions in input order", () => {
    expect(failedAssertions([a("x"), a("y"), a("z")], [r("y", true)])).toEqual([a("x"), a("z")]);
  });

  it("returns empty when all pass", () => {
    expect(failedAssertions([a("x")], [r("x", true)])).toEqual([]);
  });
});
