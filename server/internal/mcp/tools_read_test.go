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

	callTool(t, sess, "list_features", map[string]any{"status": "draft"})

	if capturedQuery != "status=draft" {
		t.Errorf("query = %q, want status=draft", capturedQuery)
	}
}

func TestMCPGetFeature(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":                       "ok",
		"/api/features/feat-123":        map[string]any{"id": "feat-123", "title": "My Feature"},
		"/api/features/feat-123/issues": map[string]any{"ready_now": []any{}, "blocked": []any{}, "pull_requests": []any{}},
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
	if _, ok := out["issues_by_repo"]; !ok {
		t.Errorf("result missing 'issues_by_repo' key: %v", out)
	}
	if _, ok := out["pull_requests_by_repo"]; !ok {
		t.Errorf("result missing 'pull_requests_by_repo' key: %v", out)
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

func TestMCPGetFeatureGroupsByRepo(t *testing.T) {
	t.Parallel()
	backend := map[string]any{
		"/health":              "ok",
		"/api/features/feat-1": map[string]any{"id": "feat-1", "title": "Auth v2"},
		"/api/features/feat-1/issues": map[string]any{
			"ready_now": []any{
				map[string]any{"id": "i1", "identifier": "MUL-1", "title": "Backend impl", "status": "todo", "priority": "high", "repo_id": "r1", "repo_name": "backend"},
				map[string]any{"id": "i2", "identifier": "MUL-2", "title": "Frontend impl", "status": "todo", "priority": "high", "repo_id": "r2", "repo_name": "frontend"},
			},
			"blocked": []any{},
			"pull_requests": []any{
				map[string]any{"number": float64(1), "html_url": "https://github.com/x/backend/pull/1", "state": "open", "title": "Auth backend", "repo_id": "r1"},
			},
		},
	}
	fake := newToolsBackend(backend)
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "get_feature", map[string]any{"feature_id": "feat-1"})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}

	byRepo, ok := out["issues_by_repo"].(map[string]any)
	if !ok {
		t.Fatalf("issues_by_repo is not a map: %T %v", out["issues_by_repo"], out["issues_by_repo"])
	}
	if _, ok := byRepo["backend"]; !ok {
		t.Errorf("issues_by_repo missing 'backend' group: %v", byRepo)
	}
	if _, ok := byRepo["frontend"]; !ok {
		t.Errorf("issues_by_repo missing 'frontend' group: %v", byRepo)
	}

	prsByRepo, ok := out["pull_requests_by_repo"].(map[string]any)
	if !ok {
		t.Fatalf("pull_requests_by_repo is not a map: %T", out["pull_requests_by_repo"])
	}
	if prs, ok := prsByRepo["r1"].([]any); !ok || len(prs) != 1 {
		t.Errorf("expected 1 PR under repo r1, got: %v", prsByRepo["r1"])
	}
}

func TestMCPListRepos(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{
		"/health":    "ok",
		"/api/repos": []any{map[string]any{"id": "r1", "name": "backend", "remote_url": "github.com/x/backend"}},
	})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "list_repos", map[string]any{})
	text, isError := toolResult(t, resp)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}
	var out []any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result not valid JSON array: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("expected 1 repo, got %d", len(out))
	}
}

func TestMCPListReposBackendError(t *testing.T) {
	t.Parallel()
	fake := newToolsBackend(map[string]any{"/health": "ok"})
	defer fake.Close()

	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test")
	sess := newSession(t, s)
	initialize(t, sess)

	resp := callTool(t, sess, "list_repos", map[string]any{})
	_, isError := toolResult(t, resp)
	if !isError {
		t.Errorf("expected error when backend returns 404, got success")
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

