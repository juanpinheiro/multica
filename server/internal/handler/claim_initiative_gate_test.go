package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// initiativeGateFixture bundles the shared builders used by the Initiative
// status gate / flow tests so each scenario stays a few lines.
type initiativeGateFixture struct {
	t       *testing.T
	ctx     context.Context
	queries *db.Queries
}

func newInitiativeGateFixture(t *testing.T) initiativeGateFixture {
	t.Helper()
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	return initiativeGateFixture{t: t, ctx: context.Background(), queries: db.New(testPool)}
}

func (f initiativeGateFixture) makeAgent(name string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4)
		RETURNING id
	`, testWorkspaceID, name, handlerTestRuntimeID(f.t), testUserID).Scan(&id); err != nil {
		f.t.Fatalf("create agent: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, id) })
	return id
}

func (f initiativeGateFixture) makeFeature(status string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO feature (workspace_id, title, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`, testWorkspaceID, fmt.Sprintf("init-gate-%d", time.Now().UnixNano()), status).Scan(&id); err != nil {
		f.t.Fatalf("create feature(status=%s): %v", status, err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id) })
	return id
}

// makeIssue inserts an issue, optionally under a feature (featureID="" for none).
func (f initiativeGateFixture) makeIssue(featureID, label string) string {
	f.t.Helper()
	var featureArg any
	if featureID != "" {
		featureArg = featureID
	}
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO issue (workspace_id, feature_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES (
			$1, $2, $3, 'in_progress', 'none', $4, 'member',
			(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
			0
		)
		RETURNING id
	`, testWorkspaceID, featureArg, fmt.Sprintf("%s-%d", label, time.Now().UnixNano()), testUserID).Scan(&id); err != nil {
		f.t.Fatalf("create issue: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id) })
	return id
}

func (f initiativeGateFixture) enqueueQueuedTask(agentID, issueID string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority)
		VALUES ($1, $2, $3, 'queued', 0)
		RETURNING id
	`, agentID, handlerTestRuntimeID(f.t), issueID).Scan(&id); err != nil {
		f.t.Fatalf("enqueue task: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id) })
	return id
}

func (f initiativeGateFixture) featureStatus(featureID string) string {
	f.t.Helper()
	var status string
	if err := testPool.QueryRow(f.ctx, `SELECT status FROM feature WHERE id = $1`, featureID).Scan(&status); err != nil {
		f.t.Fatalf("read feature status: %v", err)
	}
	return status
}

// TestClaimAgentTask_InitiativeStatusGate verifies the status-driven claim:
// a task whose issue belongs to an Initiative (feature) is only claimable when
// that Initiative is 'ready' or 'running'. Issues with no Initiative are
// unaffected.
func TestClaimAgentTask_InitiativeStatusGate(t *testing.T) {
	f := newInitiativeGateFixture(t)

	mustNotClaim := func(agentID string) {
		t.Helper()
		if row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(agentID)); err == nil {
			t.Fatalf("expected no claim, got task %s", row.ID)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows, got %v", err)
		}
	}
	mustClaim := func(agentID, wantTaskID string) {
		t.Helper()
		row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(agentID))
		if err != nil {
			t.Fatalf("expected claim, got error: %v", err)
		}
		if got := uuidToString(row.ID); got != wantTaskID {
			t.Fatalf("claimed task id = %s, want %s", got, wantTaskID)
		}
	}

	t.Run("draft Initiative: not claimable", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-draft")
		issueID := f.makeIssue(f.makeFeature("draft"), "draft-issue")
		f.enqueueQueuedTask(agentID, issueID)
		mustNotClaim(agentID)
	})

	t.Run("ready Initiative: claimable", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-ready")
		issueID := f.makeIssue(f.makeFeature("ready"), "ready-issue")
		taskID := f.enqueueQueuedTask(agentID, issueID)
		mustClaim(agentID, taskID)
	})

	t.Run("running Initiative: claimable", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-running")
		issueID := f.makeIssue(f.makeFeature("running"), "running-issue")
		taskID := f.enqueueQueuedTask(agentID, issueID)
		mustClaim(agentID, taskID)
	})

	t.Run("blocked Initiative: not claimable", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-blocked")
		issueID := f.makeIssue(f.makeFeature("blocked"), "blocked-issue")
		f.enqueueQueuedTask(agentID, issueID)
		mustNotClaim(agentID)
	})

	t.Run("done Initiative: not claimable", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-done")
		issueID := f.makeIssue(f.makeFeature("done"), "done-issue")
		f.enqueueQueuedTask(agentID, issueID)
		mustNotClaim(agentID)
	})

	t.Run("no Initiative: claimable (regression)", func(t *testing.T) {
		agentID := f.makeAgent("init-gate-none")
		issueID := f.makeIssue("", "no-feature-issue")
		taskID := f.enqueueQueuedTask(agentID, issueID)
		mustClaim(agentID, taskID)
	})
}

// TestInitiative_ReadyToRunning_OnClaim verifies the execution plane flips a
// 'ready' Initiative to 'running' the first time one of its Runs is claimed.
func TestInitiative_ReadyToRunning_OnClaim(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agentID := f.makeAgent("init-flow-ready-running")
	featureID := f.makeFeature("ready")
	issueID := f.makeIssue(featureID, "flow-issue")
	f.enqueueQueuedTask(agentID, issueID)

	task, err := testHandler.TaskService.ClaimTask(f.ctx, parseUUID(agentID))
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if task == nil {
		t.Fatal("ClaimTask returned no task")
	}
	if got := f.featureStatus(featureID); got != "running" {
		t.Errorf("feature status after claim = %q, want running", got)
	}
}

// TestInitiative_RunningToInReview_OnLastIssueDone verifies the Orchestrator
// advances the Initiative to 'in_review' only once its last Issue is done — the
// completion half of the flow. The terminal step to 'done' is the PR-merge gate
// (issue 13); completion alone takes it to review.
func TestInitiative_RunningToInReview_OnLastIssueDone(t *testing.T) {
	f := newInitiativeGateFixture(t)

	featureID := f.makeFeature("running")
	issueA := f.makeIssue(featureID, "flow-a")
	issueB := f.makeIssue(featureID, "flow-b")

	markDone := func(issueID string) db.Issue {
		t.Helper()
		if _, err := testPool.Exec(f.ctx, `UPDATE issue SET status = 'done' WHERE id = $1`, issueID); err != nil {
			t.Fatalf("mark issue done: %v", err)
		}
		issue, err := f.queries.GetIssue(f.ctx, parseUUID(issueID))
		if err != nil {
			t.Fatalf("load issue: %v", err)
		}
		prev := issue
		prev.Status = "in_progress"
		testHandler.orchestrateOnIssueDone(f.ctx, prev, issue)
		return issue
	}

	markDone(issueA)
	if got := f.featureStatus(featureID); got != "running" {
		t.Errorf("after first issue done, feature = %q, want running (siblings remain)", got)
	}

	markDone(issueB)
	if got := f.featureStatus(featureID); got != "in_review" {
		t.Errorf("after last issue done, feature = %q, want in_review", got)
	}
}

func (f initiativeGateFixture) enqueueQueuedRetrospectiveTask(agentID, issueID string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'queued', 0, 'retrospective')
		RETURNING id
	`, agentID, handlerTestRuntimeID(f.t), issueID).Scan(&id); err != nil {
		f.t.Fatalf("enqueue retrospective task: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id) })
	return id
}

// TestClaimAgentTask_RetrospectiveClaimableInReview verifies the in_review
// exception to the Initiative-status gate: a retrospective Run (dispatched at
// the in_review boundary) is claimable while the Initiative is in_review, even
// though a worker Run on the same Initiative is not.
func TestClaimAgentTask_RetrospectiveClaimableInReview(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("in_review")

	workerAgent := f.makeAgent("inreview-worker")
	f.enqueueQueuedTask(workerAgent, f.makeIssue(featureID, "inreview-worker-issue"))
	if row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(workerAgent)); err == nil {
		t.Fatalf("worker task should be gated at in_review, claimed %s", row.ID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}

	retroAgent := f.makeAgent("inreview-retro")
	retroTask := f.enqueueQueuedRetrospectiveTask(retroAgent, f.makeIssue(featureID, "inreview-retro-issue"))
	row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(retroAgent))
	if err != nil {
		t.Fatalf("retrospective task should be claimable at in_review: %v", err)
	}
	if got := uuidToString(row.ID); got != retroTask {
		t.Fatalf("claimed task = %s, want retrospective %s", got, retroTask)
	}
}

// TestInitiative_RetrospectiveClaim_DoesNotRevertInReview verifies that claiming
// the retrospective Run (which happens while the Initiative is in_review) does
// not pull the Initiative back to running via advanceInitiativeToRunning.
func TestInitiative_RetrospectiveClaim_DoesNotRevertInReview(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("in_review")
	agentID := f.makeAgent("inreview-noretrovert")
	issueID := f.makeIssue(featureID, "inreview-noretrovert-issue")
	f.enqueueQueuedRetrospectiveTask(agentID, issueID)

	task, err := testHandler.TaskService.ClaimTask(f.ctx, parseUUID(agentID))
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if task == nil {
		t.Fatal("expected to claim the retrospective task at in_review")
	}
	if got := f.featureStatus(featureID); got != "in_review" {
		t.Fatalf("Initiative status = %s after retrospective claim, want in_review (must not revert to running)", got)
	}
}
