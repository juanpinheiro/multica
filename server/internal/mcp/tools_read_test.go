package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

// newToolsBackend creates a fake backend that serves canned JSON responses
// keyed by request path. Unknown paths return 404.
func newToolsBackend(routes map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}))
}

func callTool(t *testing.T, sess *session, toolName string, args map[string]any) map[string]any {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	msg := `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"` + toolName + `","arguments":` + string(argsJSON) + `}}`
	sess.sendLine(t, msg)
	resp := sess.readLine(t)
	return resp
}

func toolResult(t *testing.T, resp map[string]any) (string, bool) {
	t.Helper()
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result, got: %v", resp)
	}
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("empty content in result: %v", result)
	}
	first, _ := content[0].(map[string]any)
	text, _ := first["text"].(string)
	isError, _ := result["isError"].(bool)
	return text, isError
}

func TestMCPListFeatures(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":      "ok",
		"/api/features": map[string]any{"features": []any{}, "total": 0},
	})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "list_features", map[string]any{})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if _, ok := out["features"]; !ok {
		t.Errorf("result missing 'features' key: %v", out)
	}
}

func TestMCPListFeaturesStatusFilter(t *testing.T) {
	t.Parallel()
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"features": []any{}, "total": 0})
	}))
	defer srv.Close()

	client := cli.NewAPIClient(srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	callTool(t, sess, "list_features", map[string]any{"status": "planned"})

	if capturedQuery != "status=planned" {
		t.Errorf("query = %q, want status=planned", capturedQuery)
	}
}

func TestMCPGetFeature(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":                     "ok",
		"/api/features/feat-123":      map[string]any{"id": "feat-123", "title": "My Feature"},
		"/api/features/feat-123/issues": map[string]any{"ready_now": []any{}, "blocked": []any{}},
	})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_feature", map[string]any{"feature_id": "feat-123"})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if _, ok := out["feature"]; !ok {
		t.Errorf("result missing 'feature' key: %v", out)
	}
	if _, ok := out["issues"]; !ok {
		t.Errorf("result missing 'issues' key: %v", out)
	}
}

func TestMCPGetFeatureMissingID(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{"/health": "ok"})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_feature", map[string]any{})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing feature_id, got success: %s", text)
	}
}

func TestMCPGetFeatureBackendError(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{"/health": "ok"})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_feature", map[string]any{"feature_id": "does-not-exist"})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing feature, got success")
	}
}

func TestMCPListIssues(t *testing.T) {
	t.Parallel()
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": []any{}, "total": 0})
	}))
	defer srv.Close()

	client := cli.NewAPIClient(srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "list_issues", map[string]any{
		"feature_id": "feat-456",
		"status":     "todo",
	})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	q, _ := url.ParseQuery(capturedQuery)
	if q.Get("feature_id") != "feat-456" {
		t.Errorf("feature_id not forwarded: %q", capturedQuery)
	}
	if q.Get("status") != "todo" {
		t.Errorf("status not forwarded: %q", capturedQuery)
	}
}

func TestMCPGetIssue(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":          "ok",
		"/api/issues/iss-1": map[string]any{"id": "iss-1", "title": "Do the thing"},
	})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_issue", map[string]any{"issue_id": "iss-1"})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if out["id"] != "iss-1" {
		t.Errorf("id = %v, want iss-1", out["id"])
	}
}

func TestMCPGetIssueMissingID(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{"/health": "ok"})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_issue", map[string]any{})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error for missing issue_id, got success")
	}
}

func TestMCPListAgents(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":     "ok",
		"/api/agents": []any{map[string]any{"id": "agent-1", "name": "Claude"}},
	})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "list_agents", map[string]any{})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out []any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON array: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("got %d agents, want 1", len(out))
	}
}

func TestMCPToolErrorIncludesHTTPStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":"invalid uuid"}`))
	}))
	defer srv.Close()

	client := cli.NewAPIClient(srv.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_issue", map[string]any{"issue_id": "not-a-uuid"})
	text, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected tool error, got success")
	}
	if text == "" {
		t.Errorf("error message is empty")
	}
}

