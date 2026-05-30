package mcp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/cli"
	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

// fakeBackend returns an httptest.Server that handles /health only.
func fakeBackend(t *testing.T) *httptest.Server {
	t.Helper()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(fake.Close)
	return fake
}

// session drives a full MCP JSON-RPC exchange over io.Pipe.
// It starts the server in a goroutine and returns write/read halves.
type session struct {
	stdin  *io.PipeWriter
	lines  *bufio.Scanner
	done   chan error
	cancel context.CancelFunc
}

func newSession(t *testing.T, s *multicamcp.Server) *session {
	t.Helper()
	pr, pw := io.Pipe()
	outpr, outpw := io.Pipe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	done := make(chan error, 1)
	go func() {
		err := s.ServeReadWriter(ctx, pr, outpw)
		_ = outpw.Close()
		done <- err
	}()

	t.Cleanup(func() {
		cancel()
		_ = pw.Close()
		<-done
	})

	return &session{
		stdin:  pw,
		lines:  bufio.NewScanner(outpr),
		done:   done,
		cancel: cancel,
	}
}

func (s *session) sendLine(t *testing.T, msg string) {
	t.Helper()
	if _, err := io.WriteString(s.stdin, msg+"\n"); err != nil {
		t.Fatalf("write to server stdin: %v", err)
	}
}

func (s *session) readLine(t *testing.T) map[string]any {
	t.Helper()
	if !s.lines.Scan() {
		if err := s.lines.Err(); err != nil {
			t.Fatalf("scan response: %v", err)
		}
		t.Fatal("server closed stdout before sending a response")
	}
	var out map[string]any
	if err := json.Unmarshal(s.lines.Bytes(), &out); err != nil {
		t.Fatalf("parse response %q: %v", s.lines.Text(), err)
	}
	return out
}

// initialize performs the required MCP handshake and returns the initialize response.
func initialize(t *testing.T, sess *session) map[string]any {
	t.Helper()
	const initReq = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	sess.sendLine(t, initReq)
	resp := sess.readLine(t)
	// ACK
	sess.sendLine(t, `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`)
	return resp
}

func TestMCPServerInitialize(t *testing.T) {
	t.Parallel()
	fake := fakeBackend(t)
	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test-version")

	sess := newSession(t, s)
	resp := initialize(t, sess)

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("result is nil; got %v", resp)
	}
	serverInfo, _ := result["serverInfo"].(map[string]any)
	if serverInfo == nil {
		t.Fatalf("result.serverInfo is nil; got %v", result)
	}
	if serverInfo["name"] != "multica" {
		t.Errorf("serverInfo.name = %v, want multica", serverInfo["name"])
	}
}

func TestMCPServerToolsList(t *testing.T) {
	t.Parallel()
	fake := fakeBackend(t)
	client := cli.NewAPIClient(fake.URL, "ws-test", "test-token")
	s := multicamcp.New(client, "test-version")

	sess := newSession(t, s)
	initialize(t, sess)

	sess.sendLine(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	resp := sess.readLine(t)

	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("result is nil; got %v", resp)
	}
	tools, _ := result["tools"].([]any)
	if tools == nil {
		tools = []any{}
	}
	const wantTools = 16
	if len(tools) != wantTools {
		t.Errorf("tools/list returned %d tools, want %d", len(tools), wantTools)
	}

	names := make(map[string]bool, len(tools))
	for _, t := range tools {
		if m, ok := t.(map[string]any); ok {
			names[m["name"].(string)] = true
		}
	}
	for _, want := range []string{
		"list_features", "get_feature", "list_issues", "get_issue", "list_agents", "list_repos",
		"create_feature", "update_feature", "approve_feature", "set_feature_status",
		"create_issue", "update_issue", "set_issue_status", "assign_issue",
		"comment_on_issue", "link_issue_dependency",
	} {
		if !names[want] {
			t.Errorf("tool %q not found in tools/list", want)
		}
	}
}
