package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

func newIssueSession(t *testing.T, cb *capturingBackend) *session {
	t.Helper()
	client := cli.NewAPIClient(cb.srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)
	return sess
}

// ── create_issue ──────────────────────────────────────────────────────────────

func TestMCPCreateIssueCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "issue-1", "title": "Fix login"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "create_issue", map[string]any{
		"feature_id": "feat-1",
		"title":      "Fix login",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues" {
		t.Errorf("path = %q, want /api/issues", cb.lastPath)
	}
}

func TestMCPCreateIssueSendsFeatureID(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "issue-1"})
	sess := newIssueSession(t, cb)

	callTool(t, sess, "create_issue", map[string]any{
		"feature_id": "feat-abc",
		"title":      "My Issue",
	})
	if cb.lastBody["feature_id"] != "feat-abc" {
		t.Errorf("body.feature_id = %v, want feat-abc", cb.lastBody["feature_id"])
	}
	if cb.lastBody["title"] != "My Issue" {
		t.Errorf("body.title = %v, want My Issue", cb.lastBody["title"])
	}
}

func TestMCPCreateIssueIncludesOptionalFields(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "issue-1"})
	sess := newIssueSession(t, cb)

	callTool(t, sess, "create_issue", map[string]any{
		"feature_id":    "feat-1",
		"title":         "Issue",
		"description":   "some desc",
		"priority":      "high",
		"assignee_id":   "agent-uuid",
		"assignee_type": "agent",
	})
	if cb.lastBody["description"] != "some desc" {
		t.Errorf("body.description = %v, want some desc", cb.lastBody["description"])
	}
	if cb.lastBody["priority"] != "high" {
		t.Errorf("body.priority = %v, want high", cb.lastBody["priority"])
	}
	if cb.lastBody["assignee_id"] != "agent-uuid" {
		t.Errorf("body.assignee_id = %v, want agent-uuid", cb.lastBody["assignee_id"])
	}
	if cb.lastBody["assignee_type"] != "agent" {
		t.Errorf("body.assignee_type = %v, want agent", cb.lastBody["assignee_type"])
	}
}

func TestMCPCreateIssueWithRepo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/repos":
			json.NewEncoder(w).Encode([]any{
				map[string]any{"id": "repo-uuid", "name": "backend"},
			})
		case "/api/issues":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "issue-1"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := cli.NewAPIClient(srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "create_issue", map[string]any{
		"feature_id": "feat-1",
		"title":      "Backend work",
		"repo":       "backend",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
}

func TestMCPCreateIssueUnknownRepo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/repos":
			json.NewEncoder(w).Encode([]any{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := cli.NewAPIClient(srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "create_issue", map[string]any{
		"feature_id": "feat-1",
		"title":      "Issue",
		"repo":       "nonexistent",
	})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for unknown repo, got success")
	}
}

func TestMCPCreateIssueMissingFeatureID(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "create_issue", map[string]any{"title": "orphan"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing feature_id, got success")
	}
}

func TestMCPCreateIssueMissingTitle(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "create_issue", map[string]any{"feature_id": "feat-1"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing title, got success")
	}
}

func TestMCPCreateIssueBackendError(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid feature"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "create_issue", map[string]any{
		"feature_id": "feat-1",
		"title":      "x",
	})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error, got success")
	}
	if text == "" {
		t.Errorf("error message is empty")
	}
}

// ── update_issue ──────────────────────────────────────────────────────────────

func TestMCPUpdateIssueCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "issue-1"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "update_issue", map[string]any{
		"issue_id": "issue-1",
		"title":    "New Title",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues/issue-1" {
		t.Errorf("path = %q, want /api/issues/issue-1", cb.lastPath)
	}
	if cb.lastBody["title"] != "New Title" {
		t.Errorf("body.title = %v, want New Title", cb.lastBody["title"])
	}
}

func TestMCPUpdateIssueOmitsUnprovidedFields(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "issue-1"})
	sess := newIssueSession(t, cb)

	callTool(t, sess, "update_issue", map[string]any{
		"issue_id":    "issue-1",
		"description": "new desc",
	})
	if _, ok := cb.lastBody["title"]; ok {
		t.Errorf("title should not be in body when not provided")
	}
	if _, ok := cb.lastBody["priority"]; ok {
		t.Errorf("priority should not be in body when not provided")
	}
}

func TestMCPUpdateIssueMissingID(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "update_issue", map[string]any{"title": "x"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing issue_id, got success")
	}
}

// ── set_issue_status ──────────────────────────────────────────────────────────

func TestMCPSetIssueStatusCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "issue-1", "status": "done"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "set_issue_status", map[string]any{
		"issue_id": "issue-1",
		"status":   "done",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues/issue-1" {
		t.Errorf("path = %q, want /api/issues/issue-1", cb.lastPath)
	}
	if cb.lastBody["status"] != "done" {
		t.Errorf("body.status = %v, want done", cb.lastBody["status"])
	}
}

func TestMCPSetIssueStatusBackendRejectsInvalid(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid status"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "set_issue_status", map[string]any{
		"issue_id": "issue-1",
		"status":   "not-a-real-status",
	})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error for invalid status, got success")
	}
}

// ── assign_issue ──────────────────────────────────────────────────────────────

func TestMCPAssignIssueCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "issue-1"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "assign_issue", map[string]any{
		"issue_id":      "issue-1",
		"assignee_id":   "agent-uuid",
		"assignee_type": "agent",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues/issue-1" {
		t.Errorf("path = %q, want /api/issues/issue-1", cb.lastPath)
	}
	if cb.lastBody["assignee_id"] != "agent-uuid" {
		t.Errorf("body.assignee_id = %v, want agent-uuid", cb.lastBody["assignee_id"])
	}
	if cb.lastBody["assignee_type"] != "agent" {
		t.Errorf("body.assignee_type = %v, want agent", cb.lastBody["assignee_type"])
	}
}

func TestMCPAssignIssueMissingAssigneeType(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "assign_issue", map[string]any{
		"issue_id":    "issue-1",
		"assignee_id": "agent-uuid",
	})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing assignee_type, got success")
	}
}

// ── comment_on_issue ──────────────────────────────────────────────────────────

func TestMCPCommentOnIssueCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "comment-1"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "comment_on_issue", map[string]any{
		"issue_id": "issue-1",
		"body":     "Looks good!",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues/issue-1/comments" {
		t.Errorf("path = %q, want /api/issues/issue-1/comments", cb.lastPath)
	}
	if cb.lastBody["content"] != "Looks good!" {
		t.Errorf("body.content = %v, want Looks good!", cb.lastBody["content"])
	}
}

func TestMCPCommentOnIssueMissingBody(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "comment_on_issue", map[string]any{"issue_id": "issue-1"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing body, got success")
	}
}

// ── link_issue_dependency ─────────────────────────────────────────────────────

func TestMCPLinkIssueDependencyCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "dep-1"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "link_issue_dependency", map[string]any{
		"issue_id":            "issue-1",
		"depends_on_issue_id": "issue-2",
		"type":                "blocks",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/issues/issue-1/dependencies" {
		t.Errorf("path = %q, want /api/issues/issue-1/dependencies", cb.lastPath)
	}
	if cb.lastBody["depends_on_issue_id"] != "issue-2" {
		t.Errorf("body.depends_on_issue_id = %v, want issue-2", cb.lastBody["depends_on_issue_id"])
	}
	if cb.lastBody["type"] != "blocks" {
		t.Errorf("body.type = %v, want blocks", cb.lastBody["type"])
	}
}

func TestMCPLinkIssueDependencyRelatedType(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "dep-1"})
	sess := newIssueSession(t, cb)

	callTool(t, sess, "link_issue_dependency", map[string]any{
		"issue_id":            "issue-1",
		"depends_on_issue_id": "issue-2",
		"type":                "related",
	})
	if cb.lastBody["type"] != "related" {
		t.Errorf("body.type = %v, want related", cb.lastBody["type"])
	}
}

func TestMCPLinkIssueDependencyMissingType(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "link_issue_dependency", map[string]any{
		"issue_id":            "issue-1",
		"depends_on_issue_id": "issue-2",
	})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing type, got success")
	}
}

func TestMCPLinkIssueDependencyBackendError(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusBadRequest, map[string]any{"error": "issue not found"})
	sess := newIssueSession(t, cb)

	resp := callTool(t, sess, "link_issue_dependency", map[string]any{
		"issue_id":            "issue-1",
		"depends_on_issue_id": "bad-uuid",
		"type":                "blocks",
	})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if text == "" {
		t.Errorf("error message is empty")
	}
}
