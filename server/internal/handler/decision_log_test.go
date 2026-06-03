package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/decisionlog"
	"github.com/multica-ai/multica/server/internal/middleware"
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

// seedDecision inserts one decision_log row for a feature with the given title.
// A retrospective task is materialised on demand so the row satisfies the
// run_id foreign key.
func (f decisionLogFixture) seedDecision(workspaceID, featureID, title string) {
	f.t.Helper()
	runID := seedDecisionRun(f.t, workspaceID, featureID)
	if _, err := testPool.Exec(f.ctx, `
		INSERT INTO decision_log (workspace_id, feature_id, run_id, title, decision, learning, adr_refs, context_terms)
		VALUES ($1, $2, $3, $4, 'd', 'l', $5, $6)
	`, workspaceID, featureID, runID, title, []string{"0004"}, []string{"Gate"}); err != nil {
		f.t.Fatalf("seed decision %q: %v", title, err)
	}
	f.t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM decision_log WHERE workspace_id = $1 AND title = $2`, workspaceID, title)
	})
}

// seedDecisionRun builds the minimal agent + issue + retrospective task chain
// needed to satisfy decision_log.run_id. Used only by Decision Log tests.
func seedDecisionRun(t *testing.T, workspaceID, featureID string) string {
	t.Helper()
	ctx := context.Background()

	var agentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4)
		RETURNING id
	`, workspaceID, fmt.Sprintf("retro-seed-%d", time.Now().UnixNano()), handlerTestRuntimeID(t), testUserID).Scan(&agentID); err != nil {
		t.Fatalf("seed retro agent: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentID) })

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, feature_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES (
			$1, $2, $3, 'done', 'none', $4, 'member',
			(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
			0
		)
		RETURNING id
	`, workspaceID, featureID, fmt.Sprintf("retro-seed-issue-%d", time.Now().UnixNano()), testUserID).Scan(&issueID); err != nil {
		t.Fatalf("seed retro issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'running', 0, 'retrospective')
		RETURNING id
	`, agentID, handlerTestRuntimeID(t), issueID).Scan(&taskID); err != nil {
		t.Fatalf("seed retro task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	return taskID
}

func callListDecisionLogWorkspace(t *testing.T, workspaceID, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	url := "/api/decisions"
	if rawQuery != "" {
		url = url + "?" + rawQuery
	}
	req := newRequest("GET", url, nil)
	req = req.WithContext(middleware.SetMemberContext(req.Context(), workspaceID, db.Member{}))
	testHandler.ListDecisionLogWorkspace(w, req)
	return w
}

func decodeDecisionListBody(t *testing.T, w *httptest.ResponseRecorder) []decisionLogResponse {
	t.Helper()
	var body struct {
		Decisions []decisionLogResponse `json:"decisions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body.Decisions
}

func TestListDecisionLogWorkspace_EmptyReturnsEmptyList(t *testing.T) {
	newDecisionLogFixture(t)

	w := callListDecisionLogWorkspace(t, testWorkspaceID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if decisions := decodeDecisionListBody(t, w); len(decisions) != 0 {
		t.Errorf("decisions = %d, want 0", len(decisions))
	}
}

func TestListDecisionLogWorkspace_AcrossFeaturesNewestFirst(t *testing.T) {
	f := newDecisionLogFixture(t)

	featureA := f.makeFeature("in_review")
	featureB := f.makeFeature("in_review")
	f.seedDecision(testWorkspaceID, featureA, "first-decision")
	time.Sleep(2 * time.Millisecond)
	f.seedDecision(testWorkspaceID, featureB, "second-decision")

	w := callListDecisionLogWorkspace(t, testWorkspaceID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	titles := titlesOf(decodeDecisionListBody(t, w))
	if !containsBoth(titles, "first-decision", "second-decision") {
		t.Fatalf("missing seeded decisions: %v", titles)
	}
	if firstIndex(titles, "second-decision") > firstIndex(titles, "first-decision") {
		t.Errorf("expected newest first; got order: %v", titles)
	}
}

func TestListDecisionLogWorkspace_PaginatesByLimitAndOffset(t *testing.T) {
	f := newDecisionLogFixture(t)
	feature := f.makeFeature("in_review")

	for i := 0; i < 3; i++ {
		f.seedDecision(testWorkspaceID, feature, fmt.Sprintf("paginated-%d-%d", time.Now().UnixNano(), i))
		time.Sleep(1 * time.Millisecond)
	}

	limited := decodeDecisionListBody(t, callListDecisionLogWorkspace(t, testWorkspaceID, "limit=1"))
	if len(limited) != 1 {
		t.Fatalf("limit=1 returned %d rows", len(limited))
	}

	offset := decodeDecisionListBody(t, callListDecisionLogWorkspace(t, testWorkspaceID, "limit=1&offset=1"))
	if len(offset) != 1 {
		t.Fatalf("limit=1&offset=1 returned %d rows", len(offset))
	}
	if offset[0].ID == limited[0].ID {
		t.Errorf("offset did not advance: same id %s", offset[0].ID)
	}
}

func TestListDecisionLogWorkspace_IsolatesAcrossWorkspaces(t *testing.T) {
	f := newDecisionLogFixture(t)

	mine := f.makeFeature("in_review")
	f.seedDecision(testWorkspaceID, mine, "mine-decision")

	otherWorkspaceID := createOtherTestWorkspace(t)
	otherFeature := seedFeatureInWorkspace(t, otherWorkspaceID)
	f.seedDecision(otherWorkspaceID, otherFeature, "leaked-decision")

	mineTitles := titlesOf(decodeDecisionListBody(t, callListDecisionLogWorkspace(t, testWorkspaceID, "")))
	if !contains(mineTitles, "mine-decision") {
		t.Errorf("mine workspace missing its own decision: %v", mineTitles)
	}
	if contains(mineTitles, "leaked-decision") {
		t.Errorf("other workspace decision leaked: %v", mineTitles)
	}
}

func seedFeatureInWorkspace(t *testing.T, workspaceID string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO feature (workspace_id, title, status)
		VALUES ($1, $2, 'in_review')
		RETURNING id
	`, workspaceID, fmt.Sprintf("ws-isolation-%d", time.Now().UnixNano())).Scan(&id); err != nil {
		t.Fatalf("create feature in other workspace: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id)
	})
	return id
}

func titlesOf(rows []decisionLogResponse) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Title
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func containsBoth(haystack []string, a, b string) bool {
	return contains(haystack, a) && contains(haystack, b)
}

func firstIndex(haystack []string, needle string) int {
	for i, h := range haystack {
		if h == needle {
			return i
		}
	}
	return -1
}
