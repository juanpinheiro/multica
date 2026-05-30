package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInjectRuntimeConfigPreserving_PreservesExistingFile(t *testing.T) {
	workDir := t.TempDir()
	userContent := "# My project\n\nDo not touch this.\n"
	claudeMd := filepath.Join(workDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMd, []byte(userContent), 0o644); err != nil {
		t.Fatalf("seed CLAUDE.md: %v", err)
	}

	brief, wrote, err := InjectRuntimeConfigPreserving(workDir, "claude", TaskContextForEnv{AgentName: "Bot"})
	if err != nil {
		t.Fatalf("InjectRuntimeConfigPreserving: %v", err)
	}
	if wrote {
		t.Fatal("wrote = true, want false when a user file is preserved")
	}
	if brief == "" {
		t.Fatal("brief is empty; the daemon needs it to deliver context inline")
	}

	got, err := os.ReadFile(claudeMd)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if string(got) != userContent {
		t.Fatalf("CLAUDE.md was modified:\n%s", got)
	}
}

func TestInjectRuntimeConfigPreserving_WritesWhenAbsent(t *testing.T) {
	workDir := t.TempDir()

	brief, wrote, err := InjectRuntimeConfigPreserving(workDir, "claude", TaskContextForEnv{AgentName: "Bot"})
	if err != nil {
		t.Fatalf("InjectRuntimeConfigPreserving: %v", err)
	}
	if !wrote {
		t.Fatal("wrote = false, want true when no instruction file exists")
	}
	got, err := os.ReadFile(filepath.Join(workDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if string(got) != brief {
		t.Fatal("written CLAUDE.md does not match the returned brief")
	}
}

func TestInjectRuntimeConfigPreserving_PromptOnlyProviderWritesNothing(t *testing.T) {
	workDir := t.TempDir()

	brief, wrote, err := InjectRuntimeConfigPreserving(workDir, "unknown-provider", TaskContextForEnv{})
	if err != nil {
		t.Fatalf("InjectRuntimeConfigPreserving: %v", err)
	}
	if wrote {
		t.Fatal("wrote = true for a prompt-only provider, want false")
	}
	if brief == "" {
		t.Fatal("brief should still be returned for inline delivery")
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatalf("read workDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("prompt-only provider wrote %d files into the umbrella", len(entries))
	}
}
