package handler

import (
	"fmt"
	"testing"
	"time"
)

// setFeatureTripwire configures an Initiative's Mode/budget/failure-tolerance.
func (f initiativeGateFixture) setFeatureTripwire(featureID string, runBudget, failureTolerance int) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		UPDATE feature SET budget_runs = $2, failure_tolerance = $3 WHERE id = $1
	`, featureID, runBudget, failureTolerance); err != nil {
		f.t.Fatalf("set feature tripwire: %v", err)
	}
}

// countTripwireInboxItems counts the tripwire alerts raised for an Initiative.
func (f initiativeGateFixture) countTripwireInboxItems(featureID string) int {
	f.t.Helper()
	var n int
	if err := testPool.QueryRow(f.ctx, `
		SELECT count(*) FROM inbox_item
		WHERE workspace_id = $1
		  AND type = 'initiative_tripwire'
		  AND details->>'feature_id' = $2
	`, testWorkspaceID, featureID).Scan(&n); err != nil {
		f.t.Fatalf("count tripwire inbox items: %v", err)
	}
	return n
}

// TestTripwire_RunBudget_PausesAndAlerts: an Initiative that hits its Run budget
// is moved to blocked and the human is pinged, instead of dispatching more work.
func TestTripwire_RunBudget_PausesAndAlerts(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agent := f.makeAgent(fmt.Sprintf("tw-run-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 1, 0) // run budget 1, failure tripwire off
	issueID := f.makeIssue(featureID, "tw-run")
	f.enqueueQueuedTask(agent, issueID) // one Run → at the budget

	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)

	if got := f.featureStatus(featureID); got != "blocked" {
		t.Fatalf("feature status = %q, want blocked (run budget tripped)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 1 {
		t.Fatalf("tripwire inbox items = %d, want 1", got)
	}
}

// TestTripwire_UnderBudget_DoesNotPause: an Initiative below all caps reconciles
// normally and is never blocked.
func TestTripwire_UnderBudget_DoesNotPause(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agent := f.makeAgent(fmt.Sprintf("tw-ok-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 5, 3) // generous run budget, default tolerance
	issueA := f.makeIssue(featureID, "tw-ok-a")
	issueB := f.makeIssue(featureID, "tw-ok-b")
	f.enqueueQueuedTask(agent, issueA)

	f.markIssueDone(issueA)
	issue := f.loadIssue(issueA)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)
	_ = issueB // a sibling stays open, so the Initiative is not done either

	if got := f.featureStatus(featureID); got != "running" {
		t.Fatalf("feature status = %q, want running (under budget)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 0 {
		t.Fatalf("tripwire inbox items = %d, want 0", got)
	}
}

// TestTripwire_FailureTolerance_PausesAndAlerts: a Milestone that fails its
// Definition of Done as many times as the tolerance pauses the Initiative.
func TestTripwire_FailureTolerance_PausesAndAlerts(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("tw-fail-worker-%d", time.Now().UnixNano()))
	validator := f.makeAgent(fmt.Sprintf("tw-fail-validator-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 0, 2) // pause after the 2nd same-Milestone failure
	m1 := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, m1, "tw-fail")
	f.assignIssue(issueID, worker)
	a1 := f.makeDodAssertion(featureID, m1, "tests pass", 0)
	f.syncIssueCounter()
	f.markIssueDone(issueID)

	failValidationOnce := func() {
		taskID := f.makeValidatorTask(validator, issueID)
		task := f.loadTask(taskID)
		testHandler.recordValidationOnCompletion(f.ctx, &task, &ValidationInput{
			Results: []validationResultInput{{AssertionID: a1, Passed: false, Detail: "still failing"}},
		})
	}

	failValidationOnce()
	if got := f.featureStatus(featureID); got != "running" {
		t.Fatalf("after 1 failure, feature = %q, want running (below tolerance)", got)
	}

	failValidationOnce()
	if got := f.featureStatus(featureID); got != "blocked" {
		t.Fatalf("after 2 failures, feature = %q, want blocked (failure tolerance tripped)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 1 {
		t.Fatalf("tripwire inbox items = %d, want 1", got)
	}
}
