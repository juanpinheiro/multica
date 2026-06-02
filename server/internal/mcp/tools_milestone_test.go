package mcp_test

import (
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

func newMilestoneSession(t *testing.T, cb *capturingBackend) *session {
	t.Helper()
	client := cli.NewAPIClient(cb.srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)
	return sess
}

// ── create_milestone ──────────────────────────────────────────────────────────

func TestMCPCreateMilestoneCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "ms-1", "title": "M1"})
	sess := newMilestoneSession(t, cb)

	resp := callTool(t, sess, "create_milestone", map[string]any{
		"feature_id": "feat-1",
		"title":      "M1",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/milestones" {
		t.Errorf("path = %q, want /api/milestones", cb.lastPath)
	}
	if cb.lastBody["feature_id"] != "feat-1" {
		t.Errorf("body.feature_id = %v, want feat-1", cb.lastBody["feature_id"])
	}
}

func TestMCPCreateMilestoneOmitsPositionWhenUnset(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "ms-1"})
	sess := newMilestoneSession(t, cb)

	callTool(t, sess, "create_milestone", map[string]any{
		"feature_id": "feat-1",
		"title":      "M1",
	})
	if _, ok := cb.lastBody["position"]; ok {
		t.Errorf("position should be omitted when not provided")
	}
}

func TestMCPCreateMilestoneSendsPosition(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "ms-1"})
	sess := newMilestoneSession(t, cb)

	callTool(t, sess, "create_milestone", map[string]any{
		"feature_id": "feat-1",
		"title":      "M1",
		"position":   2,
	})
	if cb.lastBody["position"] != float64(2) {
		t.Errorf("position = %v, want 2", cb.lastBody["position"])
	}
}

func TestMCPCreateMilestoneMissingTitle(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newMilestoneSession(t, cb)

	resp := callTool(t, sess, "create_milestone", map[string]any{"feature_id": "feat-1"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing title, got success")
	}
}

// ── update_milestone ──────────────────────────────────────────────────────────

func TestMCPUpdateMilestoneCallsPATCH(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusOK, map[string]any{"id": "ms-1"})
	sess := newMilestoneSession(t, cb)

	resp := callTool(t, sess, "update_milestone", map[string]any{
		"milestone_id": "ms-1",
		"title":        "Renamed",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", cb.lastMethod)
	}
	if cb.lastPath != "/api/milestones/ms-1" {
		t.Errorf("path = %q, want /api/milestones/ms-1", cb.lastPath)
	}
	if cb.lastBody["title"] != "Renamed" {
		t.Errorf("body.title = %v, want Renamed", cb.lastBody["title"])
	}
}

// ── create_dod_assertion ──────────────────────────────────────────────────────

func TestMCPCreateDodAssertionCallsPOST(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{"id": "dod-1", "text": "It works"})
	sess := newMilestoneSession(t, cb)

	resp := callTool(t, sess, "create_dod_assertion", map[string]any{
		"milestone_id": "ms-1",
		"text":         "It works",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	if cb.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", cb.lastMethod)
	}
	if cb.lastPath != "/api/milestones/ms-1/dod" {
		t.Errorf("path = %q, want /api/milestones/ms-1/dod", cb.lastPath)
	}
	if cb.lastBody["text"] != "It works" {
		t.Errorf("body.text = %v, want It works", cb.lastBody["text"])
	}
}

func TestMCPCreateDodAssertionMissingText(t *testing.T) {
	t.Parallel()
	cb := newCapturingBackend(t, http.StatusCreated, map[string]any{})
	sess := newMilestoneSession(t, cb)

	resp := callTool(t, sess, "create_dod_assertion", map[string]any{"milestone_id": "ms-1"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing text, got success")
	}
}
