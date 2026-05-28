package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TestAutopilotFeatureGate verifies that DispatchAutopilot respects
// feature.status: dispatch is skipped unless the feature is in_progress,
// and autopilots with no feature are unaffected.
func TestAutopilotFeatureGate(t *testing.T) {
	ctx := context.Background()

	var agentID string
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&agentID); err != nil {
		t.Fatalf("load test agent: %v", err)
	}
	queries := db.New(testPool)

	// makeFeature inserts a feature with the given status and registers cleanup.
	makeFeature := func(t *testing.T, status string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO feature (workspace_id, title, status)
			VALUES ($1, $2, $3)
			RETURNING id::text
		`, testWorkspaceID, fmt.Sprintf("gate-feature-%d", time.Now().UnixNano()), status).Scan(&id); err != nil {
			t.Fatalf("create feature(status=%s): %v", status, err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id)
		})
		return id
	}

	// makeAutopilot creates an autopilot via the HTTP handler and returns the
	// loaded db.Autopilot row. Pass featureID="" to create one without a feature.
	makeAutopilot := func(t *testing.T, featureID string) db.Autopilot {
		t.Helper()
		body := map[string]any{
			"title":          fmt.Sprintf("gate-ap-%d", time.Now().UnixNano()),
			"assignee_id":    agentID,
			"execution_mode": "create_issue",
		}
		if featureID != "" {
			body["feature_id"] = featureID
		}
		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/autopilots?workspace_id="+testWorkspaceID, body)
		testHandler.CreateAutopilot(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("CreateAutopilot: %d %s", w.Code, w.Body.String())
		}
		var resp AutopilotResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode autopilot: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM autopilot WHERE id = $1`, resp.ID)
		})
		ap, err := queries.GetAutopilot(ctx, parseUUID(resp.ID))
		if err != nil {
			t.Fatalf("get autopilot: %v", err)
		}
		return ap
	}

	t.Run("planned skips dispatch and records skip reason", func(t *testing.T) {
		featureID := makeFeature(t, "planned")
		ap := makeAutopilot(t, featureID)

		run, err := testHandler.AutopilotService.DispatchAutopilot(ctx, ap, pgtype.UUID{}, "manual", nil)
		if err != nil {
			t.Fatalf("DispatchAutopilot: %v", err)
		}
		if run.Status != "skipped" {
			t.Errorf("run.Status = %q, want skipped", run.Status)
		}
		if run.FailureReason.String != "feature_not_in_progress" {
			t.Errorf("run.FailureReason = %q, want feature_not_in_progress", run.FailureReason.String)
		}
		if run.IssueID.Valid {
			t.Error("run.IssueID should be unset for skipped dispatch")
		}
		var taskCount int
		testPool.QueryRow(ctx, `
			SELECT count(*) FROM agent_task_queue atq
			JOIN issue i ON atq.issue_id = i.id
			WHERE i.origin_id = $1
		`, ap.ID).Scan(&taskCount)
		if taskCount != 0 {
			t.Errorf("want 0 agent_task_queue rows for skipped dispatch, got %d", taskCount)
		}
	})

	t.Run("in_progress dispatches normally and enqueues task", func(t *testing.T) {
		featureID := makeFeature(t, "in_progress")
		ap := makeAutopilot(t, featureID)
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE origin_id = $1`, ap.ID)
		})

		run, err := testHandler.AutopilotService.DispatchAutopilot(ctx, ap, pgtype.UUID{}, "manual", nil)
		if err != nil {
			t.Fatalf("DispatchAutopilot: %v", err)
		}
		if run.Status != "issue_created" {
			t.Errorf("run.Status = %q, want issue_created", run.Status)
		}
		if !run.IssueID.Valid {
			t.Error("run.IssueID should be set for dispatched run")
			return
		}
		var taskCount int
		testPool.QueryRow(ctx, `
			SELECT count(*) FROM agent_task_queue WHERE issue_id = $1
		`, run.IssueID).Scan(&taskCount)
		if taskCount != 1 {
			t.Errorf("want 1 agent_task_queue row for dispatched run, got %d", taskCount)
		}
	})

	t.Run("completed skips dispatch", func(t *testing.T) {
		featureID := makeFeature(t, "completed")
		ap := makeAutopilot(t, featureID)

		run, err := testHandler.AutopilotService.DispatchAutopilot(ctx, ap, pgtype.UUID{}, "manual", nil)
		if err != nil {
			t.Fatalf("DispatchAutopilot: %v", err)
		}
		if run.Status != "skipped" {
			t.Errorf("run.Status = %q, want skipped", run.Status)
		}
		if run.FailureReason.String != "feature_not_in_progress" {
			t.Errorf("run.FailureReason = %q, want feature_not_in_progress", run.FailureReason.String)
		}
	})

	t.Run("no feature dispatches normally", func(t *testing.T) {
		ap := makeAutopilot(t, "") // no feature linked
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE origin_id = $1`, ap.ID)
		})

		run, err := testHandler.AutopilotService.DispatchAutopilot(ctx, ap, pgtype.UUID{}, "manual", nil)
		if err != nil {
			t.Fatalf("DispatchAutopilot: %v", err)
		}
		if run.Status != "issue_created" {
			t.Errorf("run.Status = %q, want issue_created", run.Status)
		}
		if !run.IssueID.Valid {
			t.Error("run.IssueID should be set for dispatched run")
		}
	})
}
