package agent

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// hangAfterScript builds a POSIX fake-CLI that emits the given stdout lines and
// then hangs (sleep) without closing stdout — the exact completion-then-hang
// that froze serial Initiatives until the 30-minute idle watchdog and then
// mislabelled the successful run as blocked. `exec sleep` replaces the shell so
// the spawned process itself holds the stdout pipe open; completion-driven
// teardown must cancel the run on the completion message and resolve it within
// the grace window.
func hangAfterScript(lines ...string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	for _, line := range lines {
		b.WriteString("printf '%s\\n' '")
		b.WriteString(line)
		b.WriteString("'\n")
	}
	b.WriteString("exec sleep 30\n")
	return b.String()
}

// awaitTeardownResult drains messages and returns the Result, failing if the
// hung process was not torn down well under the idle-watchdog horizon.
func awaitTeardownResult(t *testing.T, session *Session) Result {
	t.Helper()
	go func() {
		for range session.Messages {
		}
	}()
	start := time.Now()
	select {
	case result, ok := <-session.Result:
		if !ok {
			t.Fatal("result channel closed without a value")
		}
		// Without teardown this blocks until the test timeout; with it the
		// result lands shortly after the 200ms grace, far under the watchdog.
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("result took %s; teardown must fire well under the idle watchdog", elapsed)
		}
		return result
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for result — process hang was not torn down")
		return Result{}
	}
}

func skipIfWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fixture is POSIX-only")
	}
}

func TestGeminiExecuteCompletesWhenProcessHangsAfterResult(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)

	fakePath := filepath.Join(t.TempDir(), "gemini")
	script := hangAfterScript(
		`{"type":"init","session_id":"sess-hang"}`,
		`{"type":"message","role":"assistant","content":"all done"}`,
		`{"type":"result"}`,
	)
	writeTestExecutable(t, fakePath, []byte(script))

	backend := &geminiBackend{
		cfg:                 Config{ExecutablePath: fakePath, Logger: slog.Default()},
		resultTeardownGrace: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "prompt-ignored", ExecOptions{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := awaitTeardownResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("expected status=completed, got %q (error=%q)", result.Status, result.Error)
	}
	if result.Output != "all done" {
		t.Fatalf("expected output 'all done', got %q", result.Output)
	}
	if result.SessionID != "sess-hang" {
		t.Fatalf("expected session sess-hang, got %q", result.SessionID)
	}
}

func TestAntigravityExecuteCompletesWhenProcessHangsAfterResult(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)

	fakePath := filepath.Join(t.TempDir(), "antigravity")
	script := hangAfterScript(
		`{"type":"init","session_id":"sess-hang"}`,
		`{"type":"message","role":"assistant","content":"all done"}`,
		`{"type":"result"}`,
	)
	writeTestExecutable(t, fakePath, []byte(script))

	backend := &antigravityBackend{
		cfg:                 Config{ExecutablePath: fakePath, Logger: slog.Default()},
		resultTeardownGrace: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "prompt-ignored", ExecOptions{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := awaitTeardownResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("expected status=completed, got %q (error=%q)", result.Status, result.Error)
	}
	if result.Output != "all done" {
		t.Fatalf("expected output 'all done', got %q", result.Output)
	}
	if result.SessionID != "sess-hang" {
		t.Fatalf("expected session sess-hang, got %q", result.SessionID)
	}
}

func TestCopilotExecuteCompletesWhenProcessHangsAfterResult(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)

	fakePath := filepath.Join(t.TempDir(), "copilot")
	script := hangAfterScript(fixtureSessionStart, fixtureAssistantMessage, fixtureResult)
	writeTestExecutable(t, fakePath, []byte(script))

	backend := &copilotBackend{
		cfg:                 Config{ExecutablePath: fakePath, Logger: slog.Default()},
		resultTeardownGrace: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "prompt-ignored", ExecOptions{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := awaitTeardownResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("expected status=completed, got %q (error=%q)", result.Status, result.Error)
	}
	if !strings.Contains(result.Output, "pong") {
		t.Fatalf("expected output to contain 'pong', got %q", result.Output)
	}
	if result.SessionID != "35059dc3-d928-4ffb-8616-b78938621d85" {
		t.Fatalf("unexpected session id %q", result.SessionID)
	}
}

func TestCursorExecuteCompletesWhenProcessHangsAfterResult(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)

	fakePath := filepath.Join(t.TempDir(), "cursor-agent")
	script := hangAfterScript(
		`{"type":"system","subtype":"init","session_id":"sess-hang"}`,
		`{"type":"result","subtype":"success","result":"all done","session_id":"sess-hang"}`,
	)
	writeTestExecutable(t, fakePath, []byte(script))

	backend := &cursorBackend{
		cfg:                 Config{ExecutablePath: fakePath, Logger: slog.Default()},
		resultTeardownGrace: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "prompt-ignored", ExecOptions{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := awaitTeardownResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("expected status=completed, got %q (error=%q)", result.Status, result.Error)
	}
	if result.Output != "all done" {
		t.Fatalf("expected output 'all done', got %q", result.Output)
	}
	if result.SessionID != "sess-hang" {
		t.Fatalf("expected session sess-hang, got %q", result.SessionID)
	}
}

func TestPiExecuteCompletesWhenProcessHangsAfterTurnEnd(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)

	fakePath := filepath.Join(t.TempDir(), "pi")
	script := hangAfterScript(
		`{"type":"agent_start"}`,
		`{"type":"turn_end","message":{"role":"assistant","model":"test","usage":{"input":1,"output":1,"cacheRead":0,"cacheWrite":0,"totalTokens":2}}}`,
	)
	writeTestExecutable(t, fakePath, []byte(script))

	backend := &piBackend{
		cfg:                 Config{ExecutablePath: fakePath, Logger: slog.Default()},
		resultTeardownGrace: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "prompt-ignored", ExecOptions{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := awaitTeardownResult(t, session)
	if result.Status != "completed" {
		t.Fatalf("expected status=completed, got %q (error=%q)", result.Status, result.Error)
	}
}
