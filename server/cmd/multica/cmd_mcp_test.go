package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMCPSubprocessHelper is the executable side of subprocess tests.
// When MCP_TEST_SUBPROCESS is set, it replaces the test main with a real
// mcp-command run, so the calling test can inspect exit codes and stderr.
// Otherwise the test is skipped.
func TestMCPSubprocessHelper(t *testing.T) {
	if os.Getenv("MCP_TEST_SUBPROCESS") != "1" {
		t.Skip("not a subprocess helper run")
	}
	serverURL := os.Getenv("MCP_TEST_SERVER_URL")
	rootCmd.SetArgs([]string{"mcp", "--server-url", serverURL, "--profile", "nonexistent-mcp-test-profile"})
	_ = rootCmd.Execute()
}

// spawnHelper runs the current test binary as a subprocess using TestMCPSubprocessHelper.
// extra is a list of "KEY=VALUE" pairs added on top of the parent environment.
func spawnHelper(t *testing.T, serverURL string, extra ...string) *exec.Cmd {
	t.Helper()
	base := filterTestEnv(os.Environ())
	base = append(base, "MCP_TEST_SUBPROCESS=1", "MCP_TEST_SERVER_URL="+serverURL)
	base = append(base, extra...)
	cmd := exec.Command(os.Args[0], "-test.run=TestMCPSubprocessHelper", "-test.v=false")
	cmd.Env = base
	return cmd
}

// filterTestEnv returns env with Go test infrastructure variables removed so
// the subprocess is a clean slate (no inherited MULTICA_TOKEN etc.).
func filterTestEnv(env []string) []string {
	drop := map[string]bool{
		"MULTICA_TOKEN":      true,
		"MULTICA_SERVER_URL": true,
		"MULTICA_WORKSPACE_ID": true,
	}
	result := make([]string, 0, len(env))
	for _, e := range env {
		key := e
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			key = e[:idx]
		}
		if !drop[key] {
			result = append(result, e)
		}
	}
	return result
}

func TestMCPCommandHelpPrintsUsage(t *testing.T) {
	t.Parallel()
	if mcpCmd.Use != "mcp" {
		t.Errorf("mcpCmd.Use = %q, want mcp", mcpCmd.Use)
	}
	if mcpCmd.Short == "" {
		t.Error("mcpCmd.Short is empty")
	}
	if mcpCmd.RunE == nil {
		t.Error("mcpCmd.RunE is nil — command is not wired to a handler")
	}
}

func TestMCPExitCodeMissingToken(t *testing.T) {
	t.Parallel()
	// Use a running httptest server so we get past the URL check; the token
	// guard fires before the health check, so the server receives no requests.
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(fake.Close)

	cmd := spawnHelper(t, fake.URL)
	// No MULTICA_TOKEN set → should exit 2.
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v (output: %q)", err, out)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2 (output: %q)", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "MULTICA_TOKEN") {
		t.Errorf("stderr %q should mention MULTICA_TOKEN", out)
	}
}

func TestMCPExitCodeUnreachableServer(t *testing.T) {
	t.Parallel()
	// Port 1 is never open. Token is valid so we reach the health check.
	cmd := spawnHelper(t, "http://127.0.0.1:1", "MULTICA_TOKEN=valid-test-token")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v (output: %q)", err, out)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2 (output: %q)", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "unreachable") {
		t.Errorf("stderr %q should mention unreachable", out)
	}
}
