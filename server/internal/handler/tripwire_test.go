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

// setFeatureCostBudgets configures an Initiative's token and time caps. A zero
// cap means "unlimited" and stays inert in the tripwire.
func (f initiativeGateFixture) setFeatureCostBudgets(featureID string, tokenBudget, timeBudgetSeconds int64) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		UPDATE feature SET budget_tokens = $2, budget_seconds = $3 WHERE id = $1
	`, featureID, tokenBudget, timeBudgetSeconds); err != nil {
		f.t.Fatalf("set feature cost budgets: %v", err)
	}
}

// recordTaskUsage inserts a task_usage row so the tripwire's token aggregation
// has something to sum. Mirrors the columns the daemon writes per Run.
func (f initiativeGateFixture) recordTaskUsage(taskID string, inputTokens, outputTokens int64) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		INSERT INTO task_usage (task_id, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, updated_at)
		VALUES ($1, 'test', 'test-model', $2, $3, 0, 0, now())
	`, taskID, inputTokens, outputTokens); err != nil {
		f.t.Fatalf("record task usage: %v", err)
	}
}

// completeTaskWithDuration marks a task terminal with explicit start/complete
// timestamps so the tripwire's elapsed-seconds aggregation has a finite duration
// to sum. The Run is completed `seconds` ago over a window of `seconds`.
func (f initiativeGateFixture) completeTaskWithDuration(taskID string, seconds int) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		UPDATE agent_task_queue
		SET status = 'completed', started_at = now() - make_interval(secs => $2), completed_at = now()
		WHERE id = $1
	`, taskID, seconds); err != nil {
		f.t.Fatalf("complete task with duration: %v", err)
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

// TestTripwire_TokenBudget_PausesAndAlerts: an Initiative whose accumulated
// task_usage tokens reach its cap is moved to blocked and the human pinged,
// proving the token aggregation in loadTripwireState reaches the pure rule.
func TestTripwire_TokenBudget_PausesAndAlerts(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agent := f.makeAgent(fmt.Sprintf("tw-tok-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 0, 0) // run + failure tripwires off — isolate tokens
	f.setFeatureCostBudgets(featureID, 1_000, 0)
	issueID := f.makeIssue(featureID, "tw-tok")
	taskID := f.enqueueQueuedTask(agent, issueID)
	f.recordTaskUsage(taskID, 600, 500) // 1_100 ≥ 1_000 → over cap

	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)

	if got := f.featureStatus(featureID); got != "blocked" {
		t.Fatalf("feature status = %q, want blocked (token budget tripped)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 1 {
		t.Fatalf("tripwire inbox items = %d, want 1", got)
	}
}

// TestTripwire_TimeBudget_PausesAndAlerts: an Initiative whose total terminal
// run-time reaches its time cap is moved to blocked, proving the elapsed-
// seconds aggregation in loadTripwireState reaches the pure rule.
func TestTripwire_TimeBudget_PausesAndAlerts(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agent := f.makeAgent(fmt.Sprintf("tw-time-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 0, 0) // run + failure tripwires off — isolate time
	f.setFeatureCostBudgets(featureID, 0, 10)
	issueID := f.makeIssue(featureID, "tw-time")
	taskID := f.enqueueQueuedTask(agent, issueID)
	f.completeTaskWithDuration(taskID, 15) // 15s ≥ 10s → over cap

	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)

	if got := f.featureStatus(featureID); got != "blocked" {
		t.Fatalf("feature status = %q, want blocked (time budget tripped)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 1 {
		t.Fatalf("tripwire inbox items = %d, want 1", got)
	}
}

// TestTripwire_CostBudgets_UnderCap_DoesNotPause: when token and time usage are
// under the caps, the Initiative keeps running. Guards against false trips that
// would block real work.
func TestTripwire_CostBudgets_UnderCap_DoesNotPause(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agent := f.makeAgent(fmt.Sprintf("tw-under-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	f.setFeatureTripwire(featureID, 0, 0)
	f.setFeatureCostBudgets(featureID, 10_000, 3_600)
	issueA := f.makeIssue(featureID, "tw-under-a")
	issueB := f.makeIssue(featureID, "tw-under-b") // sibling keeps the Initiative active
	taskID := f.enqueueQueuedTask(agent, issueA)
	f.recordTaskUsage(taskID, 100, 100) //   200 ≪ 10_000
	f.completeTaskWithDuration(taskID, 5) //   5 ≪ 3_600

	f.markIssueDone(issueA)
	issue := f.loadIssue(issueA)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)
	_ = issueB

	if got := f.featureStatus(featureID); got != "running" {
		t.Fatalf("feature status = %q, want running (under cost caps)", got)
	}
	if got := f.countTripwireInboxItems(featureID); got != 0 {
		t.Fatalf("tripwire inbox items = %d, want 0", got)
	}
}
