// Pure Definition-of-Done evaluation (ADR-0007), the TS lockstep mirror of the
// canonical Go module (server/internal/dod). Given a Milestone's assertions and
// the latest validator verdicts, decide whether the Milestone is satisfied.

export interface AssertionRef {
  id: string;
}

export interface AssertionVerdict {
  assertionId: string;
  passed: boolean;
}

// milestoneSatisfied reports whether every assertion has a passing verdict and
// none has a failing one. A Milestone with no assertions is vacuously satisfied.
export function milestoneSatisfied(
  assertions: AssertionRef[],
  results: AssertionVerdict[],
): boolean {
  return failedAssertions(assertions, results).length === 0;
}

// failedAssertions returns, in input order, the assertions that are not
// satisfied: those with no verdict, or with any failing verdict.
export function failedAssertions<T extends AssertionRef>(
  assertions: T[],
  results: AssertionVerdict[],
): T[] {
  const verdicts = verdictsByAssertion(results);
  return assertions.filter((a) => verdicts.get(a.id) !== true);
}

// verdictsByAssertion folds verdicts to one bool per assertion: an assertion
// passes only if it has at least one verdict and none of them failed.
function verdictsByAssertion(results: AssertionVerdict[]): Map<string, boolean> {
  const v = new Map<string, boolean>();
  for (const r of results) {
    const prev = v.get(r.assertionId);
    v.set(r.assertionId, prev === undefined ? r.passed : prev && r.passed);
  }
  return v;
}
