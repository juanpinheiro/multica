package agent

import (
	"log/slog"
	"testing"
)

func TestNewReturnsAntigravityBackend(t *testing.T) {
	t.Parallel()
	b, err := New("antigravity", Config{ExecutablePath: "/nonexistent/antigravity"})
	if err != nil {
		t.Fatalf("New(antigravity) error: %v", err)
	}
	if _, ok := b.(*antigravityBackend); !ok {
		t.Fatalf("expected *antigravityBackend, got %T", b)
	}
}

func TestBuildAntigravityArgsBaseline(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("write a haiku", ExecOptions{}, slog.Default())

	wantContains := [][]string{
		{"-p", "write a haiku"},
		{"--output-format", "stream-json"},
		{"--yolo"},
	}
	for _, pair := range wantContains {
		assertArgsContainSequence(t, args, pair)
	}
}

func TestBuildAntigravityArgsWithModel(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{Model: "ag-pro"}, slog.Default())
	assertArgsContainSequence(t, args, []string{"--model", "ag-pro"})
}

func TestBuildAntigravityArgsOmitsModelWhenEmpty(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{}, slog.Default())
	for _, a := range args {
		if a == "--model" {
			t.Fatalf("expected no --model flag when Model is empty, got args=%v", args)
		}
	}
}

func TestBuildAntigravityArgsWithSystemPrompt(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{SystemPrompt: "be concise"}, slog.Default())
	assertArgsContainSequence(t, args, []string{"--system-prompt", "be concise"})
}

func TestBuildAntigravityArgsWithResume(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{ResumeSessionID: "session-abc"}, slog.Default())
	assertArgsContainSequence(t, args, []string{"--resume", "session-abc"})
}

func TestBuildAntigravityArgsFiltersBlockedCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{
		CustomArgs: []string{"--output-format", "text", "--sandbox"},
	}, slog.Default())

	for i, a := range args {
		if a == "--output-format" && i+1 < len(args) && args[i+1] == "text" {
			t.Fatalf("blocked --output-format text should have been filtered: %v", args)
		}
	}
	assertArgsContainSequence(t, args, []string{"--sandbox"})
}

func TestBuildAntigravityArgsPassesThroughCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildAntigravityArgs("hi", ExecOptions{
		CustomArgs: []string{"--sandbox"},
	}, slog.Default())

	assertArgsContainSequence(t, args, []string{"--sandbox"})
}

func assertArgsContainSequence(t *testing.T, args []string, seq []string) {
	t.Helper()
	for i := 0; i <= len(args)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if args[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Fatalf("expected args to contain %v, got %v", seq, args)
}
