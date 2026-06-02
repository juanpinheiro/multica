// Pure Tripwire/Budget pause-decision (ADR-0005), the TS lockstep mirror of the
// canonical Go module (server/internal/tripwire). Given an Initiative's
// accumulated failure and budget state, decide whether it should pause and ping
// the human — the AFK safety net that stops a runaway run from burning days.

// TripwireReason names why an Initiative should pause, or "" when it should not.
export type TripwireReason =
  | ""
  | "failure_tolerance"
  | "token_budget"
  | "run_budget"
  | "time_budget";

// TripwireState is the accumulated cost/failure snapshot of one Initiative.
// Every budget is a hard ceiling; a zero (or negative) cap means "no cap" — that
// dimension never trips.
export interface TripwireState {
  // maxMilestoneFailures is the worst single Milestone's validation-failure
  // count: how many times any one Milestone has failed its Definition of Done.
  maxMilestoneFailures: number;
  // failureTolerance is the number of repeated same-Milestone failures allowed
  // before pausing. A value <= 0 disables the failure tripwire.
  failureTolerance: number;
  tokensUsed: number;
  tokenBudget: number;
  runsUsed: number;
  runBudget: number;
  elapsedSeconds: number;
  timeBudget: number;
}

export interface TripwireVerdict {
  pause: boolean;
  reason: TripwireReason;
}

// shouldPause reports whether the Initiative has tripped a safety limit and, if
// so, the first reason in a fixed precedence: failure tolerance, then token,
// run, and time budgets.
export function shouldPause(s: TripwireState): TripwireVerdict {
  const reason = tripReason(s);
  return { pause: reason !== "", reason };
}

function tripReason(s: TripwireState): TripwireReason {
  if (s.failureTolerance > 0 && s.maxMilestoneFailures >= s.failureTolerance) {
    return "failure_tolerance";
  }
  if (s.tokenBudget > 0 && s.tokensUsed >= s.tokenBudget) {
    return "token_budget";
  }
  if (s.runBudget > 0 && s.runsUsed >= s.runBudget) {
    return "run_budget";
  }
  if (s.timeBudget > 0 && s.elapsedSeconds >= s.timeBudget) {
    return "time_budget";
  }
  return "";
}
