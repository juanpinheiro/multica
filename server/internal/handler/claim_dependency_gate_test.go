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

// TestClaimAgentTask_DependencyGate verifies that ClaimAgentTask refuses to
// dispatch an issue whose `blocks`/`blocked_by` dependencies are unsatisfied
// (blocker.status != 'done'). `related` dependencies are non-gating. Issues
// with no dependency rows behave as before.
//
// The test calls Queries.ClaimAgentTask directly to isolate the SQL claim
// behavior from per-agent capacity / runtime routing.
func TestClaimAgentTask_DependencyGate(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	queries := db.New(testPool)

	// makeAgent creates a fresh agent in the test workspace and registers
	// cleanup. Each scenario uses its own agent so per-(issue, agent)
	// serialization in ClaimAgentTask never confounds the dependency-gate
	// assertion.
	makeAgent := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent (
				workspace_id, name, description, runtime_mode, runtime_config,
				runtime_id, visibility, max_concurrent_tasks, owner_id
			)
			VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4)
			RETURNING id
		`, testWorkspaceID, name, handlerTestRuntimeID(t), testUserID).Scan(&id); err != nil {
			t.Fatalf("create agent: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, id)
		})
		return id
	}

	// makeIssue inserts an issue with the given status and registers cleanup.
	makeIssue := func(t *testing.T, status, label string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
			VALUES (
				$1, $2, $3, 'none', $4, 'member',
				(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
				0
			)
			RETURNING id
		`, testWorkspaceID, fmt.Sprintf("%s-%d", label, time.Now().UnixNano()), status, testUserID).Scan(&id); err != nil {
			t.Fatalf("create issue(status=%s): %v", status, err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
		})
		return id
	}

	// linkDependency inserts an issue_dependency row of the given type and
	// registers cleanup.
	linkDependency := func(t *testing.T, issueID, dependsOnID, depType string) {
		t.Helper()
		if _, err := testPool.Exec(ctx, `
			INSERT INTO issue_dependency (issue_id, depends_on_issue_id, type)
			VALUES ($1, $2, $3)
		`, issueID, dependsOnID, depType); err != nil {
			t.Fatalf("link dependency(%s): %v", depType, err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `
				DELETE FROM issue_dependency
				WHERE issue_id = $1 AND depends_on_issue_id = $2
			`, issueID, dependsOnID)
		})
	}

	// enqueueQueuedTask inserts a queued agent_task_queue row for the agent
	// and issue. Returns the task id.
	enqueueQueuedTask := func(t *testing.T, agentID, issueID string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (
				agent_id, runtime_id, issue_id, status, priority
			) VALUES ($1, $2, $3, 'queued', 0)
			RETURNING id
		`, agentID, handlerTestRuntimeID(t), issueID).Scan(&id); err != nil {
			t.Fatalf("enqueue task: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id)
		})
		return id
	}

	mustNotClaim := func(t *testing.T, agentID string) {
		t.Helper()
		row, err := queries.ClaimAgentTask(ctx, parseUUID(agentID))
		if err == nil {
			t.Fatalf("expected no claim, got task %s status=%s", row.ID, row.Status)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows, got %v", err)
		}
	}

	mustClaim := func(t *testing.T, agentID, wantTaskID string) {
		t.Helper()
		row, err := queries.ClaimAgentTask(ctx, parseUUID(agentID))
		if err != nil {
			t.Fatalf("expected claim, got error: %v", err)
		}
		if got := uuidToString(row.ID); got != wantTaskID {
			t.Fatalf("claimed task id = %s, want %s", got, wantTaskID)
		}
		if row.Status != "dispatched" {
			t.Errorf("claimed task status = %q, want dispatched", row.Status)
		}
	}

	t.Run("blocks dep not done: not claimable; mark done: claimable", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-blocks-pending")
		blocker := makeIssue(t, "in_progress", "blocker")
		dependent := makeIssue(t, "todo", "dependent")
		linkDependency(t, dependent, blocker, "blocks")
		taskID := enqueueQueuedTask(t, agentID, dependent)

		mustNotClaim(t, agentID)

		if _, err := testPool.Exec(ctx, `UPDATE issue SET status = 'done' WHERE id = $1`, blocker); err != nil {
			t.Fatalf("mark blocker done: %v", err)
		}
		mustClaim(t, agentID, taskID)
	})

	t.Run("multiple blocks deps all done: claimable", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-blocks-all-done")
		b1 := makeIssue(t, "done", "b1")
		b2 := makeIssue(t, "done", "b2")
		dependent := makeIssue(t, "todo", "dependent-multi")
		linkDependency(t, dependent, b1, "blocks")
		linkDependency(t, dependent, b2, "blocks")
		taskID := enqueueQueuedTask(t, agentID, dependent)

		mustClaim(t, agentID, taskID)
	})

	t.Run("multiple blocks deps one in_progress: not claimable", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-blocks-one-pending")
		bDone := makeIssue(t, "done", "b-done")
		bPending := makeIssue(t, "in_progress", "b-pending")
		dependent := makeIssue(t, "todo", "dependent-mixed")
		linkDependency(t, dependent, bDone, "blocks")
		linkDependency(t, dependent, bPending, "blocks")
		enqueueQueuedTask(t, agentID, dependent)

		mustNotClaim(t, agentID)
	})

	t.Run("related dep not done: claimable (related is non-gating)", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-related")
		other := makeIssue(t, "in_progress", "related-other")
		dependent := makeIssue(t, "todo", "dependent-related")
		linkDependency(t, dependent, other, "related")
		taskID := enqueueQueuedTask(t, agentID, dependent)

		mustClaim(t, agentID, taskID)
	})

	t.Run("no dependency rows: claimable (regression)", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-no-deps")
		dependent := makeIssue(t, "todo", "dependent-no-deps")
		taskID := enqueueQueuedTask(t, agentID, dependent)

		mustClaim(t, agentID, taskID)
	})

	t.Run("blocked_by alias also gates", func(t *testing.T) {
		agentID := makeAgent(t, "dep-gate-blocked-by")
		blocker := makeIssue(t, "in_progress", "blocker-by")
		dependent := makeIssue(t, "todo", "dependent-by")
		linkDependency(t, dependent, blocker, "blocked_by")
		enqueueQueuedTask(t, agentID, dependent)

		mustNotClaim(t, agentID)
	})
}
