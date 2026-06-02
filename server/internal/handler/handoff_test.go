package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// handoffFixture extends initiativeGateFixture with task-completion helpers.
type handoffFixture struct {
	initiativeGateFixture
}

func newHandoffFixture(t *testing.T) handoffFixture {
	t.Helper()
	return handoffFixture{initiativeGateFixture: newInitiativeGateFixture(t)}
}

// makeRunningTask inserts a running (role=worker) task for the given agent/issue.
func (f handoffFixture) makeRunningTask(agentID, issueID string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'running', 0, 'worker')
		RETURNING id
	`, agentID, handlerTestRuntimeID(f.t), issueID).Scan(&id); err != nil {
		f.t.Fatalf("create running task: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id) })
	return id
}

// makeValidatorTask inserts a running task with role=validator.
func (f handoffFixture) makeValidatorTask(agentID, issueID string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'running', 0, 'validator')
		RETURNING id
	`, agentID, handlerTestRuntimeID(f.t), issueID).Scan(&id); err != nil {
		f.t.Fatalf("create validator task: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id) })
	return id
}

func (f handoffFixture) loadTask(taskID string) db.AgentTaskQueue {
	f.t.Helper()
	row, err := f.queries.GetAgentTask(f.ctx, parseUUID(taskID))
	if err != nil {
		f.t.Fatalf("load task: %v", err)
	}
	return row
}

func (f handoffFixture) listHandoffs(issueID string) []db.Handoff {
	f.t.Helper()
	rows, err := f.queries.ListHandoffsByIssue(f.ctx, parseUUID(issueID))
	if err != nil {
		f.t.Fatalf("list handoffs: %v", err)
	}
	return rows
}

// TestWriteHandoffOnCompletion_WorkerWritesHandoff verifies that completing a
// worker Run with a HandoffInput persists a handoff row.
func TestWriteHandoffOnCompletion_WorkerWritesHandoff(t *testing.T) {
	f := newHandoffFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("handoff-worker-%d", time.Now().UnixNano()))
	issueID := f.makeIssue("", "handoff-issue")
	taskID := f.makeRunningTask(agentID, issueID)
	task := f.loadTask(taskID)

	input := &HandoffInput{
		Done:        []string{"implement feature A", "write tests"},
		LeftUndone:  []string{"fix edge case B"},
		Commands:    []handoffCommandInput{{Command: "go build ./...", ExitCode: 0}},
		Discoveries: []string{"upstream bug found"},
	}
	testHandler.writeHandoffOnCompletion(context.Background(), &task, input)

	rows := f.listHandoffs(issueID)
	if len(rows) != 1 {
		t.Fatalf("handoff count = %d, want 1", len(rows))
	}
	h := rows[0]
	if len(h.Done) != 2 || h.Done[0] != "implement feature A" {
		t.Errorf("Done = %v, want [implement feature A, write tests]", h.Done)
	}
	if len(h.LeftUndone) != 1 || h.LeftUndone[0] != "fix edge case B" {
		t.Errorf("LeftUndone = %v, want [fix edge case B]", h.LeftUndone)
	}
	if len(h.Discoveries) != 1 {
		t.Errorf("Discoveries = %v, want 1 item", h.Discoveries)
	}
}

// TestWriteHandoffOnCompletion_ValidatorSkipsHandoff verifies that validator
// Runs do not write handoffs.
func TestWriteHandoffOnCompletion_ValidatorSkipsHandoff(t *testing.T) {
	f := newHandoffFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("handoff-validator-%d", time.Now().UnixNano()))
	issueID := f.makeIssue("", "handoff-validator-issue")
	taskID := f.makeValidatorTask(agentID, issueID)
	task := f.loadTask(taskID)

	input := &HandoffInput{Done: []string{"validated"}}
	testHandler.writeHandoffOnCompletion(context.Background(), &task, input)

	rows := f.listHandoffs(issueID)
	if len(rows) != 0 {
		t.Errorf("validator wrote %d handoffs, want 0", len(rows))
	}
}

// TestWriteHandoffOnCompletion_NilInputSkips verifies no panic or write when
// HandoffInput is nil.
func TestWriteHandoffOnCompletion_NilInputSkips(t *testing.T) {
	f := newHandoffFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("handoff-nil-%d", time.Now().UnixNano()))
	issueID := f.makeIssue("", "handoff-nil-issue")
	taskID := f.makeRunningTask(agentID, issueID)
	task := f.loadTask(taskID)

	testHandler.writeHandoffOnCompletion(context.Background(), &task, nil)

	rows := f.listHandoffs(issueID)
	if len(rows) != 0 {
		t.Errorf("nil input wrote %d handoffs, want 0", len(rows))
	}
}

// TestListHandoffsByIssue_OrderedOldestFirst verifies handoffs are returned
// ordered oldest-first so handoff.LatestState receives them in the right order.
func TestListHandoffsByIssue_OrderedOldestFirst(t *testing.T) {
	f := newHandoffFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("handoff-order-%d", time.Now().UnixNano()))
	issueID := f.makeIssue("", "handoff-order-issue")

	// Write two handoffs; rely on insertion order from sequential calls.
	for _, label := range []string{"first", "second"} {
		task := f.makeRunningTask(agentID, issueID)
		row := f.loadTask(task)
		testHandler.writeHandoffOnCompletion(context.Background(), &row, &HandoffInput{
			Done: []string{label},
		})
	}

	rows := f.listHandoffs(issueID)
	if len(rows) != 2 {
		t.Fatalf("handoff count = %d, want 2", len(rows))
	}
	if rows[0].Done[0] != "first" {
		t.Errorf("oldest handoff done[0] = %q, want first", rows[0].Done[0])
	}
	if rows[1].Done[0] != "second" {
		t.Errorf("newest handoff done[0] = %q, want second", rows[1].Done[0])
	}

	// GetLatestHandoffByIssue should return the most recent one.
	latest, err := f.queries.GetLatestHandoffByIssue(f.ctx, parseUUID(issueID))
	if err != nil && err != pgx.ErrNoRows {
		t.Fatalf("GetLatestHandoffByIssue: %v", err)
	}
	if latest.Done[0] != "second" {
		t.Errorf("latest handoff done[0] = %q, want second", latest.Done[0])
	}
}
