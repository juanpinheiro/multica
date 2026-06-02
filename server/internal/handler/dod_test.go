package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// makeDodAssertion inserts a DoD assertion tagged to a Milestone, returning its id.
func (f initiativeGateFixture) makeDodAssertion(featureID, milestoneID, text string, position int) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO dod_assertion (workspace_id, feature_id, milestone_id, text, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, testWorkspaceID, featureID, milestoneID, text, position).Scan(&id); err != nil {
		f.t.Fatalf("create dod assertion: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM dod_assertion WHERE id = $1`, id) })
	return id
}

// assignIssue points an issue's assignee at an agent.
func (f initiativeGateFixture) assignIssue(issueID, agentID string) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `UPDATE issue SET assignee_type = 'agent', assignee_id = $1 WHERE id = $2`, agentID, issueID); err != nil {
		f.t.Fatalf("assign issue: %v", err)
	}
}

// markIssueDone sets an issue's status to done.
func (f initiativeGateFixture) markIssueDone(issueID string) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `UPDATE issue SET status = 'done' WHERE id = $1`, issueID); err != nil {
		f.t.Fatalf("mark issue done: %v", err)
	}
}

func (f initiativeGateFixture) milestoneValidationStatus(milestoneID string) string {
	f.t.Helper()
	var status string
	if err := testPool.QueryRow(f.ctx, `SELECT validation_status FROM milestone WHERE id = $1`, milestoneID).Scan(&status); err != nil {
		f.t.Fatalf("read milestone status: %v", err)
	}
	return status
}

func (f initiativeGateFixture) countFollowUpIssues(milestoneID string) int {
	f.t.Helper()
	var n int
	if err := testPool.QueryRow(f.ctx, `
		SELECT count(*) FROM issue WHERE milestone_id = $1 AND title = 'Fix failed Definition of Done'
	`, milestoneID).Scan(&n); err != nil {
		f.t.Fatalf("count follow-up issues: %v", err)
	}
	return n
}

func (f initiativeGateFixture) countValidatorRuns(issueID string) int {
	f.t.Helper()
	var n int
	if err := testPool.QueryRow(f.ctx, `
		SELECT count(*) FROM agent_task_queue WHERE issue_id = $1 AND role = 'validator'
	`, issueID).Scan(&n); err != nil {
		f.t.Fatalf("count validator runs: %v", err)
	}
	return n
}

// syncIssueCounter realigns the workspace issue_counter with MAX(number). The
// shared makeIssue fixtures allocate numbers directly (bypassing the counter),
// so the counter lags MAX; production always allocates through the counter. This
// keeps the counter-based follow-up allocation from colliding in the test DB.
func (f initiativeGateFixture) syncIssueCounter() {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		UPDATE workspace SET issue_counter = (
			SELECT COALESCE(MAX(number), 0) FROM issue WHERE workspace_id = $1
		) WHERE id = $1
	`, testWorkspaceID); err != nil {
		f.t.Fatalf("sync issue counter: %v", err)
	}
}

func (f initiativeGateFixture) loadIssue(issueID string) db.Issue {
	f.t.Helper()
	issue, err := f.queries.GetIssue(f.ctx, parseUUID(issueID))
	if err != nil {
		f.t.Fatalf("load issue: %v", err)
	}
	return issue
}

// TestRecordValidation_AllPass_MarksMilestonePassed: a validator Run whose every
// verdict passes marks the Milestone validated, opening the Gate.
func TestRecordValidation_AllPass_MarksMilestonePassed(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	validator := f.makeAgent(fmt.Sprintf("dod-validator-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	f.assignIssue(issueID, worker)
	a1 := f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)
	a2 := f.makeDodAssertion(featureID, milestoneID, "no regressions", 1)

	taskID := f.makeValidatorTask(validator, issueID)
	task := f.loadTask(taskID)

	testHandler.recordValidationOnCompletion(context.Background(), &task, &ValidationInput{
		Results: []validationResultInput{
			{AssertionID: a1, Passed: true},
			{AssertionID: a2, Passed: true},
		},
	})

	if got := f.milestoneValidationStatus(milestoneID); got != "passed" {
		t.Errorf("milestone validation = %q, want passed", got)
	}
	if got := f.countFollowUpIssues(milestoneID); got != 0 {
		t.Errorf("follow-up issues = %d, want 0", got)
	}
}

// TestRecordValidation_Failure_MarksFailedAndCreatesFollowUp: a failing verdict
// marks the Milestone failed (keeping the next Milestone gated) and opens a
// follow-up Issue so the Initiative self-heals.
func TestRecordValidation_Failure_MarksFailedAndCreatesFollowUp(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	validator := f.makeAgent(fmt.Sprintf("dod-validator-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	f.assignIssue(issueID, worker)
	a1 := f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)
	a2 := f.makeDodAssertion(featureID, milestoneID, "no regressions", 1)
	f.syncIssueCounter()

	taskID := f.makeValidatorTask(validator, issueID)
	task := f.loadTask(taskID)

	testHandler.recordValidationOnCompletion(context.Background(), &task, &ValidationInput{
		Results: []validationResultInput{
			{AssertionID: a1, Passed: true},
			{AssertionID: a2, Passed: false, Detail: "flaky test still failing"},
		},
	})

	if got := f.milestoneValidationStatus(milestoneID); got != "failed" {
		t.Errorf("milestone validation = %q, want failed", got)
	}
	if got := f.countFollowUpIssues(milestoneID); got != 1 {
		t.Errorf("follow-up issues = %d, want 1", got)
	}
}

// TestRecordValidation_ValidatorRoleRequired: a worker Run never records DoD
// verdicts even if a ValidationInput is supplied.
func TestRecordValidation_ValidatorRoleRequired(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	a1 := f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)

	taskID := f.makeRunningTask(worker, issueID)
	task := f.loadTask(taskID)

	testHandler.recordValidationOnCompletion(context.Background(), &task, &ValidationInput{
		Results: []validationResultInput{{AssertionID: a1, Passed: true}},
	})

	if got := f.milestoneValidationStatus(milestoneID); got != "pending" {
		t.Errorf("milestone validation = %q, want pending (worker must not validate)", got)
	}
}

// TestDispatchValidatorOnBoundary_NoDoD_MarksPassed: a Milestone with no DoD is
// vacuously satisfied and marked passed at the boundary without a validator Run.
func TestDispatchValidatorOnBoundary_NoDoD_MarksPassed(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	f.assignIssue(issueID, worker)
	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)

	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(context.Background(), prev, issue)

	if got := f.milestoneValidationStatus(milestoneID); got != "passed" {
		t.Errorf("milestone validation = %q, want passed", got)
	}
	if got := f.countValidatorRuns(issueID); got != 0 {
		t.Errorf("validator runs = %d, want 0 (no DoD to check)", got)
	}
}

// TestDispatchValidatorOnBoundary_DispatchesValidatorRun: when a Milestone with
// a DoD has all its Issues done, a validator Run is dispatched.
func TestDispatchValidatorOnBoundary_DispatchesValidatorRun(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	f.assignIssue(issueID, worker)
	f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)
	f.markIssueDone(issueID)
	issue := f.loadIssue(issueID)

	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(context.Background(), prev, issue)

	if got := f.countValidatorRuns(issueID); got != 1 {
		t.Errorf("validator runs = %d, want 1", got)
	}
	if got := f.milestoneValidationStatus(milestoneID); got != "pending" {
		t.Errorf("milestone validation = %q, want pending (validator not done yet)", got)
	}
}

// TestDispatchValidatorOnBoundary_WaitsForSiblings: the validator is not
// dispatched while other Issues in the Milestone are still open.
func TestDispatchValidatorOnBoundary_WaitsForSiblings(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	doneIssue := f.makeIssueInMilestone(featureID, milestoneID, "dod-done")
	f.makeIssueInMilestone(featureID, milestoneID, "dod-open") // stays in_progress
	f.assignIssue(doneIssue, worker)
	f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)
	f.markIssueDone(doneIssue)
	issue := f.loadIssue(doneIssue)

	prev := issue
	prev.Status = "in_progress"
	testHandler.orchestrateOnIssueDone(context.Background(), prev, issue)

	if got := f.countValidatorRuns(doneIssue); got != 0 {
		t.Errorf("validator runs = %d, want 0 (sibling still open)", got)
	}
}

// TestResolveValidatorAgent_PrefersDistinctAgent: a second agent is chosen over
// the worker to preserve the creator-verifier separation.
func TestResolveValidatorAgent_PrefersDistinctAgent(t *testing.T) {
	f := newHandoffFixture(t)

	worker := f.makeAgent(fmt.Sprintf("dod-worker-%d", time.Now().UnixNano()))
	f.makeAgent(fmt.Sprintf("dod-other-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "dod-issue")
	f.assignIssue(issueID, worker)
	issue := f.loadIssue(issueID)

	got := testHandler.resolveValidatorAgent(context.Background(), issue)
	if !got.Valid {
		t.Fatal("resolved validator agent is invalid")
	}
	if uuidToString(got) == worker {
		t.Errorf("resolved validator = worker, want a distinct agent")
	}
}
