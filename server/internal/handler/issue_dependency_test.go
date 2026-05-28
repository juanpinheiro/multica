package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCreateIssueDependency verifies the POST /api/issues/{id}/dependencies
// handler: row is created, response is correct, and duplicate calls are
// idempotent (same ID returned, 200 status).
func TestCreateIssueDependency(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	makeIssue := func(t *testing.T, label string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
			VALUES (
				$1, $2, 'todo', 'none', $3, 'member',
				(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
				0
			)
			RETURNING id::text
		`, testWorkspaceID, fmt.Sprintf("dep-test-%s-%d", label, time.Now().UnixNano()), testUserID).Scan(&id); err != nil {
			t.Fatalf("create issue(%s): %v", label, err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
		})
		return id
	}

	t.Run("creates dependency and returns 201", func(t *testing.T) {
		issueID := makeIssue(t, "src")
		targetID := makeIssue(t, "target")

		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/issues/"+issueID+"/dependencies", map[string]any{
			"depends_on_issue_id": targetID,
			"type":                "blocks",
		})
		req = withURLParam(req, "id", issueID)
		testHandler.CreateIssueDependency(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp IssueDependencyResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.ID == "" {
			t.Error("response ID is empty")
		}
		if resp.IssueID != issueID {
			t.Errorf("IssueID = %q, want %q", resp.IssueID, issueID)
		}
		if resp.DependsOnIssueID != targetID {
			t.Errorf("DependsOnIssueID = %q, want %q", resp.DependsOnIssueID, targetID)
		}
		if resp.Type != "blocks" {
			t.Errorf("Type = %q, want blocks", resp.Type)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue_dependency WHERE id = $1`, resp.ID)
		})
	})

	t.Run("duplicate call is idempotent", func(t *testing.T) {
		issueID := makeIssue(t, "idem-src")
		targetID := makeIssue(t, "idem-target")

		call := func() IssueDependencyResponse {
			w := httptest.NewRecorder()
			req := newRequest("POST", "/api/issues/"+issueID+"/dependencies", map[string]any{
				"depends_on_issue_id": targetID,
				"type":                "related",
			})
			req = withURLParam(req, "id", issueID)
			testHandler.CreateIssueDependency(w, req)
			if w.Code != http.StatusCreated && w.Code != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
			}
			var resp IssueDependencyResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			return resp
		}

		first := call()
		second := call()

		if first.ID != second.ID {
			t.Errorf("idempotency failed: first ID = %q, second ID = %q", first.ID, second.ID)
		}
		if second.Type != "related" {
			t.Errorf("second Type = %q, want related", second.Type)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue_dependency WHERE id = $1`, first.ID)
		})
	})

	t.Run("rejects invalid type", func(t *testing.T) {
		issueID := makeIssue(t, "type-src")
		targetID := makeIssue(t, "type-target")

		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/issues/"+issueID+"/dependencies", map[string]any{
			"depends_on_issue_id": targetID,
			"type":                "blocked_by",
		})
		req = withURLParam(req, "id", issueID)
		testHandler.CreateIssueDependency(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for type=blocked_by, got %d", w.Code)
		}
	})

	t.Run("rejects depends_on_issue_id from different workspace", func(t *testing.T) {
		issueID := makeIssue(t, "ws-src")

		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/issues/"+issueID+"/dependencies", map[string]any{
			"depends_on_issue_id": "00000000-0000-0000-0000-000000000001",
			"type":                "blocks",
		})
		req = withURLParam(req, "id", issueID)
		testHandler.CreateIssueDependency(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for unknown target, got %d", w.Code)
		}
	})
}
