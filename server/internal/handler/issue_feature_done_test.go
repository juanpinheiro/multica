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

// featureDoneFixture groups a feature with two issues for the feature-ready
// notification tests.
type featureDoneFixture struct {
	featureID string
	issueAID  string
	issueBID  string
}

// newFeatureDoneFixture creates a shared-branch feature with two issues.
// Both issues start in_progress. branchSlug may be empty to test the
// "no branch_slug → no notification" path.
func newFeatureDoneFixture(t *testing.T, branchSlug string) featureDoneFixture {
	t.Helper()
	ctx := context.Background()

	var featureID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO feature (workspace_id, title, status, branch_slug)
		VALUES ($1, $2, 'running', NULLIF($3, ''))
		RETURNING id::text
	`, testWorkspaceID, fmt.Sprintf("feat-done-%d", time.Now().UnixNano()), branchSlug).Scan(&featureID); err != nil {
		t.Fatalf("create feature: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, featureID)
	})

	makeIssue := func(title string) string {
		t.Helper()
		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
			"title":      title,
			"status":     "in_progress",
			"feature_id": featureID,
		})
		testHandler.CreateIssue(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create issue %q: %d %s", title, w.Code, w.Body.String())
		}
		var resp IssueResponse
		json.NewDecoder(w.Body).Decode(&resp)
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, resp.ID)
		})
		return resp.ID
	}

	return featureDoneFixture{
		featureID: featureID,
		issueAID:  makeIssue(fmt.Sprintf("feat-done-A-%d", time.Now().UnixNano())),
		issueBID:  makeIssue(fmt.Sprintf("feat-done-B-%d", time.Now().UnixNano())),
	}
}

// updateIssueStatus drives UpdateIssue for a status change, asserting HTTP 200.
func updateIssueStatus(t *testing.T, issueID, status string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("PUT", "/api/issues/"+issueID, map[string]any{"status": status})
	req = withURLParam(req, "id", issueID)
	testHandler.UpdateIssue(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIssue %s → %s: %d %s", issueID, status, w.Code, w.Body.String())
	}
}

// countFeatureReadyInboxItems returns the number of feature_ready_for_review
// inbox items for the test workspace.
func countFeatureReadyInboxItems(t *testing.T, featureID string) int {
	t.Helper()
	var n int
	err := testPool.QueryRow(context.Background(), `
		SELECT count(*) FROM inbox_item
		WHERE workspace_id = $1
		  AND type = 'feature_ready_for_review'
		  AND details->>'feature_id' = $2
	`, testWorkspaceID, featureID).Scan(&n)
	if err != nil {
		t.Fatalf("count feature_ready inbox items: %v", err)
	}
	return n
}

func TestNotifyFeatureReadyForReview(t *testing.T) {
	t.Run("all issues done under shared branch fires notification", func(t *testing.T) {
		fix := newFeatureDoneFixture(t, "feature/my-branch")

		// Mark A done — B still in_progress, no notification yet.
		updateIssueStatus(t, fix.issueAID, "done")
		if n := countFeatureReadyInboxItems(t, fix.featureID); n != 0 {
			t.Errorf("after first done: want 0 inbox items, got %d", n)
		}

		// Mark B done — all siblings done, notification fires.
		updateIssueStatus(t, fix.issueBID, "done")
		if n := countFeatureReadyInboxItems(t, fix.featureID); n != 1 {
			t.Errorf("after last done: want 1 inbox item, got %d", n)
		}
	})

	t.Run("sibling still pending suppresses notification", func(t *testing.T) {
		fix := newFeatureDoneFixture(t, "feature/partial")

		// Only A reaches done; B stays in_progress.
		updateIssueStatus(t, fix.issueAID, "done")

		if n := countFeatureReadyInboxItems(t, fix.featureID); n != 0 {
			t.Errorf("want 0 inbox items while B is pending, got %d", n)
		}
	})

	t.Run("feature without branch_slug suppresses notification", func(t *testing.T) {
		// Empty branch_slug → NULL in DB → not a shared-branch feature.
		fix := newFeatureDoneFixture(t, "")

		updateIssueStatus(t, fix.issueAID, "done")
		updateIssueStatus(t, fix.issueBID, "done")

		if n := countFeatureReadyInboxItems(t, fix.featureID); n != 0 {
			t.Errorf("want 0 inbox items (no branch_slug), got %d", n)
		}
	})

	t.Run("issue without feature_id suppresses notification", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
			"title":  fmt.Sprintf("orphan-%d", time.Now().UnixNano()),
			"status": "in_progress",
		})
		testHandler.CreateIssue(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create orphan issue: %d %s", w.Code, w.Body.String())
		}
		var resp IssueResponse
		json.NewDecoder(w.Body).Decode(&resp)
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, resp.ID)
		})

		updateIssueStatus(t, resp.ID, "done")

		// No feature_id → no feature_ready_for_review items at all in workspace.
		var n int
		testPool.QueryRow(context.Background(), `
			SELECT count(*) FROM inbox_item
			WHERE workspace_id = $1 AND type = 'feature_ready_for_review'
			  AND issue_id = $2
		`, testWorkspaceID, resp.ID).Scan(&n)
		if n != 0 {
			t.Errorf("want 0 inbox items for orphan issue, got %d", n)
		}
	})

	t.Run("transition to non-done status suppresses notification", func(t *testing.T) {
		fix := newFeatureDoneFixture(t, "feature/in-progress")

		// Transition A to in_review (not done) — should not fire.
		updateIssueStatus(t, fix.issueAID, "in_review")

		if n := countFeatureReadyInboxItems(t, fix.featureID); n != 0 {
			t.Errorf("want 0 inbox items on non-done transition, got %d", n)
		}
	})

	t.Run("inbox item has correct fields", func(t *testing.T) {
		fix := newFeatureDoneFixture(t, "feature/field-check")

		updateIssueStatus(t, fix.issueAID, "done")
		updateIssueStatus(t, fix.issueBID, "done")

		var (
			title     string
			severity  string
			actorType string
		)
		err := testPool.QueryRow(context.Background(), `
			SELECT title, severity, actor_type
			FROM inbox_item
			WHERE workspace_id = $1 AND type = 'feature_ready_for_review'
			  AND details->>'feature_id' = $2
			ORDER BY created_at DESC LIMIT 1
		`, testWorkspaceID, fix.featureID).Scan(&title, &severity, &actorType)
		if err != nil {
			t.Fatalf("load inbox item: %v", err)
		}

		if severity != "action_required" {
			t.Errorf("severity = %q, want action_required", severity)
		}
		if actorType != "system" {
			t.Errorf("actor_type = %q, want system", actorType)
		}
		if title == "" {
			t.Error("inbox item title is empty")
		}
	})
}
