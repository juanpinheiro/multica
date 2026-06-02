import { describe, expect, it } from "vitest";
import { shouldPause, type TripwireState } from "./index";

const base: TripwireState = {
  maxMilestoneFailures: 0,
  failureTolerance: 0,
  tokensUsed: 0,
  tokenBudget: 0,
  runsUsed: 0,
  runBudget: 0,
  elapsedSeconds: 0,
  timeBudget: 0,
};

describe("shouldPause", () => {
  it("does not pause when nothing is configured", () => {
    expect(shouldPause(base)).toEqual({ pause: false, reason: "" });
  });

  it("trips on repeated same-milestone failure at tolerance", () => {
    expect(shouldPause({ ...base, maxMilestoneFailures: 3, failureTolerance: 3 })).toEqual({
      pause: true,
      reason: "failure_tolerance",
    });
  });

  it("does not trip below the failure tolerance", () => {
    expect(shouldPause({ ...base, maxMilestoneFailures: 2, failureTolerance: 3 })).toEqual({
      pause: false,
      reason: "",
    });
  });

  it("treats zero tolerance as a disabled failure tripwire", () => {
    expect(shouldPause({ ...base, maxMilestoneFailures: 9, failureTolerance: 0 })).toEqual({
      pause: false,
      reason: "",
    });
  });

  it("trips when the token budget is reached", () => {
    expect(shouldPause({ ...base, tokensUsed: 1000, tokenBudget: 1000 })).toEqual({
      pause: true,
      reason: "token_budget",
    });
  });

  it("trips when the run budget is reached", () => {
    expect(shouldPause({ ...base, runsUsed: 5, runBudget: 5 })).toEqual({
      pause: true,
      reason: "run_budget",
    });
  });

  it("trips when the time budget is reached", () => {
    expect(shouldPause({ ...base, elapsedSeconds: 3600, timeBudget: 3600 })).toEqual({
      pause: true,
      reason: "time_budget",
    });
  });

  it("treats zero budgets as unlimited", () => {
    expect(
      shouldPause({ ...base, tokensUsed: 1e9, runsUsed: 1e6, elapsedSeconds: 1e6 }),
    ).toEqual({ pause: false, reason: "" });
  });

  it("prioritises failure tolerance over budget caps", () => {
    expect(
      shouldPause({ ...base, maxMilestoneFailures: 3, failureTolerance: 3, runsUsed: 10, runBudget: 5 }),
    ).toEqual({ pause: true, reason: "failure_tolerance" });
  });
});
