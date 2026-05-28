package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

// capturingBackend records the last request body and method for assertion.
type capturingBackend struct {
	srv         *httptest.Server
	lastMethod  string
	lastPath    string
	lastBody    map[string]any
	statusCode  int
	responseBody any
}

func newCapturingBackend(t *testing.T, statusCode int, responseBody any) *capturingBackend {
	t.Helper()
	cb := &capturingBackend{statusCode: statusCode, responseBody: responseBody}
	cb.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		cb.lastMethod = r.Method
		cb.lastPath = r.URL.Path
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &cb.lastBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(cb.statusCode)
		json.NewEncoder(w).Encode(cb.responseBody)
	}))
	t.Cleanup(cb.srv.Close)
	return cb
}

func newFeatureSession(t *testing.T, cb *capturingBackend) *session {
	t.Helper()
	client := cli.NewAPIClient(cb.srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)
	return sess
}

// ── create_feature ─────────────────────────────────────────────────────────────

func TestMCPCreateFeatureCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "feat-1", "title": "Auth v2"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "create_feature", map[string]any{
		"title":       "Auth v2",
		"description": "Rewrite auth",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/features" {
		t.Errorf("path = %q, want /api/features", cb.lastPath)
	}
}

func TestMCPCreateFeatureForcesPlannedStatus(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "feat-1"})
	sess := newFeatureSession(t, cb)

	callTool(t, sess, "create_feature", map[string]any{
		"title":       "My Feature",
		"description": "desc",
	})
	if cb.lastBody["status"] != "planned" {
		t.Errorf("status = %v, want planned", cb.lastBody["status"])
	}
}

func TestMCPCreateFeatureIncludesOptionalFields(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "feat-1"})
	sess := newFeatureSession(t, cb)

	callTool(t, sess, "create_feature", map[string]any{
		"title":         "My Feature",
		"description":   "desc",
		"priority":      "high",
		"target_branch": "feature/auth-v2",
		"lead_id":       "agent-uuid",
	})
	if cb.lastBody["priority"] != "high" {
		t.Errorf("priority = %v, want high", cb.lastBody["priority"])
	}
	if cb.lastBody["target_branch"] != "feature/auth-v2" {
		t.Errorf("target_branch = %v, want feature/auth-v2", cb.lastBody["target_branch"])
	}
	if cb.lastBody["lead_id"] != "agent-uuid" {
		t.Errorf("lead_id = %v, want agent-uuid", cb.lastBody["lead_id"])
	}
}

func TestMCPCreateFeatureMissingTitle(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "create_feature", map[string]any{"description": "desc"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing title, got success")
	}
}

func TestMCPCreateFeatureBackendError(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid title"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "create_feature", map[string]any{
		"title":       "x",
		"description": "y",
	})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error, got success")
	}
	if text == "" {
		t.Errorf("error message is empty")
	}
}

// ── update_feature ─────────────────────────────────────────────────────────────

func TestMCPUpdateFeatureCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "feat-1"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "update_feature", map[string]any{
		"feature_id": "feat-1",
		"title":      "New Title",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/features/feat-1" {
		t.Errorf("path = %q, want /api/features/feat-1", cb.lastPath)
	}
	if cb.lastBody["title"] != "New Title" {
		t.Errorf("body.title = %v, want New Title", cb.lastBody["title"])
	}
}

func TestMCPUpdateFeatureOmitsUnprovidedFields(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "feat-1"})
	sess := newFeatureSession(t, cb)

	callTool(t, sess, "update_feature", map[string]any{
		"feature_id":  "feat-1",
		"description": "new desc",
	})
	if _, ok := cb.lastBody["title"]; ok {
		t.Errorf("title should not be in body when not provided")
	}
	if _, ok := cb.lastBody["priority"]; ok {
		t.Errorf("priority should not be in body when not provided")
	}
}

func TestMCPUpdateFeatureClearsTargetBranch(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "feat-1"})
	sess := newFeatureSession(t, cb)

	callTool(t, sess, "update_feature", map[string]any{
		"feature_id":    "feat-1",
		"target_branch": "",
	})
	if v, ok := cb.lastBody["target_branch"]; !ok || v != "" {
		t.Errorf("expected target_branch='' in body for clearing; got ok=%v value=%v", ok, v)
	}
}

// ── approve_feature ────────────────────────────────────────────────────────────

func TestMCPApproveFeatureSetsInProgress(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "feat-1", "status": "in_progress"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "approve_feature", map[string]any{"feature_id": "feat-1"})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/features/feat-1" {
		t.Errorf("path = %q, want /api/features/feat-1", cb.lastPath)
	}
	if cb.lastBody["status"] != "in_progress" {
		t.Errorf("body.status = %v, want in_progress", cb.lastBody["status"])
	}
}

func TestMCPApproveFeatureMissingID(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "approve_feature", map[string]any{})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing feature_id, got success")
	}
}

// ── set_feature_status ─────────────────────────────────────────────────────────

func TestMCPSetFeatureStatusCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "feat-1", "status": "paused"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "set_feature_status", map[string]any{
		"feature_id": "feat-1",
		"status":     "paused",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastBody["status"] != "paused" {
		t.Errorf("body.status = %v, want paused", cb.lastBody["status"])
	}
}

func TestMCPSetFeatureStatusBackendRejectsInvalidStatus(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid status"})
	sess := newFeatureSession(t, cb)

	resp := callTool(t, sess, "set_feature_status", map[string]any{
		"feature_id": "feat-1",
		"status":     "not-a-real-status",
	})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error for invalid status, got success: %s", text)
	}
}
