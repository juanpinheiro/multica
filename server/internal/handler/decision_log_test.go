package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/decisionlog"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// decisionLogFixture extends initiativeGateFixture with retrospective helpers.
type decisionLogFixture struct {
	initiativeGateFixture
}

func newDecisionLogFixture(t *testing.T) decisionLogFixture {
	t.Helper()
	return decisionLogFixture{initiativeGateFixture: newInitiativeGateFixture(t)}
}

// makeRetrospectiveTask inserts a running task with role=retrospective.
func (f decisionLogFixture) makeRetrospectiveTask(agentID, issueID string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'running', 0, 'retrospective')
		RETURNING id
	`, agentID, handlerTestRuntimeID(f.t), issueID).Scan(&id); err != nil {
		f.t.Fatalf("create retrospective task: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id) })
	return id
}

// makeAssignedIssue inserts an issue under a feature, assigned to an agent.
func (f decisionLogFixture) makeAssignedIssue(featureID, agentID, label string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO issue (workspace_id, feature_id, title, status, priority, creator_id, creator_type, assignee_id, assignee_type, number, position)
		VALUES (
			$1, $2, $3, 'done', 'none', $4, 'member', $5, 'agent',
			(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
			0
		)
		RETURNING id
	`, testWorkspaceID, featureID, fmt.Sprintf("%s-%d", label, time.Now().UnixNano()), testUserID, agentID).Scan(&id); err != nil {
		f.t.Fatalf("create assigned issue: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id) })
	return id
}

func (f decisionLogFixture) loadRetroTask(taskID string) db.AgentTaskQueue {
	f.t.Helper()
	row, err := f.queries.GetAgentTask(f.ctx, parseUUID(taskID))
	if err != nil {
		f.t.Fatalf("load task: %v", err)
	}
	return row
}

func (f decisionLogFixture) listDecisions(featureID string) []db.DecisionLog {
	f.t.Helper()
	rows, err := f.queries.ListDecisionLogByFeature(f.ctx, parseUUID(featureID))
	if err != nil {
		f.t.Fatalf("list decision log: %v", err)
	}
	return rows
}

// TestRecordRetrospectiveOnCompletion_PersistsEntries verifies a retrospective
// Run's Decision Log entries are persisted, with refs/terms preserved.
func TestRecordRetrospectiveOnCompletion_PersistsEntries(t *testing.T) {
	f := newDecisionLogFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("retro-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "retro-issue")
	taskID := f.makeRetrospectiveTask(agentID, issueID)
	task := f.loadRetroTask(taskID)

	out := &decisionlog.Output{Entries: []decisionlog.Entry{{
		Title:        "Keep the Gate thin",
		Decision:     "SQL enforces, Go specifies",
		Learning:     "two layers stayed in sync",
		AdrRefs:      []string{"0004"},
		ContextTerms: []string{"Gate"},
	}}}
	testHandler.recordRetrospectiveOnCompletion(context.Background(), &task, out)

	rows := f.listDecisions(featureID)
	if len(rows) != 1 {
		t.Fatalf("decision count = %d, want 1", len(rows))
	}
	d := rows[0]
	if d.Title != "Keep the Gate thin" || d.Decision != "SQL enforces, Go specifies" {
		t.Errorf("entry mismatch: %+v", d)
	}
	if len(d.AdrRefs) != 1 || d.AdrRefs[0] != "0004" {
		t.Errorf("adr_refs = %v", d.AdrRefs)
	}
	if len(d.ContextTerms) != 1 || d.ContextTerms[0] != "Gate" {
		t.Errorf("context_terms = %v", d.ContextTerms)
	}
}

// TestRecordRetrospectiveOnCompletion_WorkerSkips verifies non-retrospective
// Runs do not write Decision Log entries.
func TestRecordRetrospectiveOnCompletion_WorkerSkips(t *testing.T) {
	f := newDecisionLogFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("retro-worker-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("running")
	issueID := f.makeIssue(featureID, "retro-worker-issue")

	var taskID string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'running', 0, 'worker') RETURNING id
	`, agentID, handlerTestRuntimeID(t), issueID).Scan(&taskID); err != nil {
		t.Fatalf("create worker task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })
	task := f.loadRetroTask(taskID)

	testHandler.recordRetrospectiveOnCompletion(context.Background(), &task, &decisionlog.Output{
		Entries: []decisionlog.Entry{{Title: "t", Decision: "d"}},
	})

	if rows := f.listDecisions(featureID); len(rows) != 0 {
		t.Errorf("worker wrote %d decisions, want 0", len(rows))
	}
}

// TestRecordRetrospectiveOnCompletion_NilInputSkips verifies no write or panic
// when the output is nil.
func TestRecordRetrospectiveOnCompletion_NilInputSkips(t *testing.T) {
	f := newDecisionLogFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("retro-nil-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "retro-nil-issue")
	task := f.loadRetroTask(f.makeRetrospectiveTask(agentID, issueID))

	testHandler.recordRetrospectiveOnCompletion(context.Background(), &task, nil)

	if rows := f.listDecisions(featureID); len(rows) != 0 {
		t.Errorf("nil input wrote %d decisions, want 0", len(rows))
	}
}

// TestRecordRetrospectiveOnCompletion_DropsInvalidEntries verifies entries
// missing a title or decision are not persisted.
func TestRecordRetrospectiveOnCompletion_DropsInvalidEntries(t *testing.T) {
	f := newDecisionLogFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("retro-drop-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "retro-drop-issue")
	task := f.loadRetroTask(f.makeRetrospectiveTask(agentID, issueID))

	testHandler.recordRetrospectiveOnCompletion(context.Background(), &task, &decisionlog.Output{Entries: []decisionlog.Entry{
		{Title: "", Decision: "no title"},
		{Title: "no decision", Decision: ""},
		{Title: "valid", Decision: "kept"},
	}})

	rows := f.listDecisions(featureID)
	if len(rows) != 1 {
		t.Fatalf("decision count = %d, want 1 (only the valid one)", len(rows))
	}
	if rows[0].Title != "valid" {
		t.Errorf("kept wrong entry: %+v", rows[0])
	}
}

// TestDispatchRetrospective_EnqueuesOnceAtBoundary verifies dispatchRetrospective
// enqueues a retrospective Run and does not duplicate when one is in flight.
func TestDispatchRetrospective_EnqueuesOnceAtBoundary(t *testing.T) {
	f := newDecisionLogFixture(t)

	agentID := f.makeAgent(fmt.Sprintf("retro-dispatch-%d", time.Now().UnixNano()))
	featureID := f.makeFeature("in_review")
	issueID := f.makeAssignedIssue(featureID, agentID, "retro-dispatch-issue")
	issue, err := f.queries.GetIssue(f.ctx, parseUUID(issueID))
	if err != nil {
		t.Fatalf("load issue: %v", err)
	}

	testHandler.dispatchRetrospective(context.Background(), issue)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
	})

	count, err := f.queries.CountActiveRetrospectiveRunsByFeature(f.ctx, parseUUID(featureID))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("after first dispatch, active retrospectives = %d, want 1", count)
	}

	// A second dispatch while one is in flight must be a no-op.
	testHandler.dispatchRetrospective(context.Background(), issue)
	count, err = f.queries.CountActiveRetrospectiveRunsByFeature(f.ctx, parseUUID(featureID))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("after second dispatch, active retrospectives = %d, want 1 (deduped)", count)
	}
}
