package handler

import (
	"fmt"
	"testing"
	"time"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// countFollowUpWorkerRuns counts worker Runs enqueued for a Milestone's DoD
// follow-up Issue — the auto-dispatch that lets the Initiative self-heal.
func (f initiativeGateFixture) countFollowUpWorkerRuns(milestoneID string) int {
	f.t.Helper()
	var n int
	if err := testPool.QueryRow(f.ctx, `
		SELECT count(*) FROM agent_task_queue atq
		JOIN issue i ON i.id = atq.issue_id
		WHERE i.milestone_id = $1 AND i.title = 'Fix failed Definition of Done'
	`, milestoneID).Scan(&n); err != nil {
		f.t.Fatalf("count follow-up worker runs: %v", err)
	}
	return n
}

// orchestrateDone marks an Issue done and wakes the Orchestrator on the
// non-done → done transition, returning the reloaded Issue.
func (f initiativeGateFixture) orchestrateDone(issueID string) db.Issue {
	f.t.Helper()
	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)
	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)
	return issue
}

// TestOrchestrator_MultiMilestone_AdvancesToInReview drives a two-Milestone
// Initiative to in_review with no human input: each Milestone passes as its
// Issues complete, and the Initiative advances only once the last one is done.
func TestOrchestrator_MultiMilestone_AdvancesToInReview(t *testing.T) {
	f := newInitiativeGateFixture(t)

	worker := f.makeAgent(fmt.Sprintf("orch-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	m1 := f.makeMilestone(featureID, 0, "pending")
	m2 := f.makeMilestone(featureID, 1, "pending")
	issue1 := f.makeIssueInMilestone(featureID, m1, "orch-1")
	issue2 := f.makeIssueInMilestone(featureID, m2, "orch-2")
	f.assignIssue(issue1, worker)
	f.assignIssue(issue2, worker)

	f.orchestrateDone(issue1)
	if got := f.milestoneValidationStatus(m1); got != "passed" {
		t.Fatalf("m1 (no DoD) = %q, want passed", got)
	}
	if got := f.featureStatus(featureID); got != "running" {
		t.Fatalf("after m1, feature = %q, want running (m2 still open)", got)
	}

	f.orchestrateDone(issue2)
	if got := f.milestoneValidationStatus(m2); got != "passed" {
		t.Fatalf("m2 (no DoD) = %q, want passed", got)
	}
	if got := f.featureStatus(featureID); got != "in_review" {
		t.Fatalf("after m2, feature = %q, want in_review", got)
	}
}

// TestOrchestrator_Idempotent_NoDuplicateValidator verifies the reconcile is
// idempotent: waking it twice on the same boundary (as a restart would) does
// not dispatch a second validator. Statelessness — all state is read from the DB.
func TestOrchestrator_Idempotent_NoDuplicateValidator(t *testing.T) {
	f := newInitiativeGateFixture(t)

	worker := f.makeAgent(fmt.Sprintf("orch-worker-%d", time.Now().UnixNano()))
	f.makeAgent(fmt.Sprintf("orch-validator-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	m1 := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, m1, "orch-idem")
	f.assignIssue(issueID, worker)
	f.makeDodAssertion(featureID, m1, "tests pass", 0)

	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)
	prev := issue
	prev.Status = "in_progress"

	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)
	testHandler.orchestrateOnIssueDone(f.ctx, prev, issue) // re-wake (e.g. after a restart)

	if got := f.countValidatorRuns(issueID); got != 1 {
		t.Fatalf("validator runs = %d, want 1 (idempotent reconcile)", got)
	}
}

// TestOrchestrator_DodFailure_EnqueuesFollowUpWorkerRun verifies that a failing
// Definition of Done both creates a follow-up Issue and auto-enqueues its worker
// Run, so the Initiative self-heals without human input.
func TestOrchestrator_DodFailure_EnqueuesFollowUpWorkerRun(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("orch-worker-%d", time.Now().UnixNano()))
	validator := f.makeAgent(fmt.Sprintf("orch-validator-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	m1 := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, m1, "orch-dod")
	f.assignIssue(issueID, worker)
	a1 := f.makeDodAssertion(featureID, m1, "tests pass", 0)
	f.syncIssueCounter()
	f.markIssueDone(issueID)

	taskID := f.makeValidatorTask(validator, issueID)
	task := f.loadTask(taskID)
	testHandler.recordValidationOnCompletion(f.ctx, &task, &ValidationInput{
		Results: []validationResultInput{{AssertionID: a1, Passed: false, Detail: "still failing"}},
	})

	if got := f.milestoneValidationStatus(m1); got != "failed" {
		t.Fatalf("m1 = %q, want failed", got)
	}
	if got := f.countFollowUpWorkerRuns(m1); got != 1 {
		t.Fatalf("follow-up worker runs = %d, want 1 (auto-enqueued)", got)
	}
}
