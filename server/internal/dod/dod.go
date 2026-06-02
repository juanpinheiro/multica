// Package dod holds the pure Definition-of-Done evaluation (ADR-0007): given a
// Milestone's assertions and the latest validator verdicts, is the Milestone
// satisfied? This is the deterministic core that decides when the Gate opens —
// no I/O, no DB. The handler feeds it rows; the result drives whether a
// Milestone is marked validated or a follow-up Issue is created.
package dod

// Assertion is the minimal shape the evaluation reasons over: its identity.
type Assertion struct {
	ID string
}

// Result is one validator verdict for an assertion.
type Result struct {
	AssertionID string
	Passed      bool
}

// MilestoneSatisfied reports whether every assertion has a passing verdict and
// none has a failing one. A Milestone with no assertions is vacuously satisfied.
func MilestoneSatisfied(assertions []Assertion, results []Result) bool {
	return len(FailedAssertions(assertions, results)) == 0
}

// FailedAssertions returns, in input order, the assertions that are not
// satisfied: those with no verdict, or with any failing verdict. The
// Orchestrator turns these into follow-up Issues.
func FailedAssertions(assertions []Assertion, results []Result) []Assertion {
	verdicts := verdictsByAssertion(results)
	var failed []Assertion
	for _, a := range assertions {
		if !verdicts[a.ID] {
			failed = append(failed, a)
		}
	}
	return failed
}

// verdictsByAssertion folds results to one bool per assertion: an assertion
// passes only if it has at least one verdict and none of them failed.
func verdictsByAssertion(results []Result) map[string]bool {
	v := make(map[string]bool, len(results))
	for _, r := range results {
		if prev, seen := v[r.AssertionID]; seen {
			v[r.AssertionID] = prev && r.Passed
		} else {
			v[r.AssertionID] = r.Passed
		}
	}
	return v
}
