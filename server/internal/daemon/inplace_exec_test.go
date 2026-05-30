package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/workspace/inplace"
)

func TestExecModeIsInPlace(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"in_place", true},
		{"worktree", false},
		{"", false},
		{"bogus", false},
	}
	for _, tc := range cases {
		if got := execModeIsInPlace(tc.raw); got != tc.want {
			t.Errorf("execModeIsInPlace(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// TestUmbrellaLockerSerializesSameWorkspaceParallelizesDistinct proves the core
// gate-vs-locker property at the daemon seam: two in-place tasks sharing an
// umbrella run serially (the second parks via the wait callback until the first
// releases), while tasks on distinct umbrellas never contend — the same way two
// worktree tasks, which never touch the locker at all, stay parallel. The test
// also guards against a deadlock: the second acquire must complete after release.
func TestUmbrellaLockerSerializesSameWorkspaceParallelizesDistinct(t *testing.T) {
	locker := inplace.NewLocker()
	ctx := context.Background()

	umbrellaA := t.TempDir()
	umbrellaB := t.TempDir()

	// Distinct umbrellas: both acquire immediately, no wait callback.
	relA, err := locker.Acquire(ctx, umbrellaA, "task-a", func(string) {
		t.Error("distinct umbrella should not fire the wait callback")
	})
	if err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	relB, err := locker.Acquire(ctx, umbrellaB, "task-b", func(string) {
		t.Error("distinct umbrella should not fire the wait callback")
	})
	if err != nil {
		t.Fatalf("acquire B: %v", err)
	}
	relB()

	// Same umbrella as A: must park until A releases.
	waited := make(chan string, 1)
	acquired := make(chan struct{})
	go func() {
		rel, err := locker.Acquire(ctx, umbrellaA, "task-c", func(holder string) {
			waited <- holder
		})
		if err != nil {
			t.Errorf("acquire C: %v", err)
			return
		}
		close(acquired)
		rel()
	}()

	select {
	case holder := <-waited:
		if holder != "task-a" {
			t.Fatalf("wait callback holder = %q, want task-a", holder)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second acquirer never parked behind the held umbrella")
	}

	select {
	case <-acquired:
		t.Fatal("second acquirer took the lock while A still held it")
	case <-time.After(50 * time.Millisecond):
	}

	relA()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("second acquirer never ran after release — deadlock")
	}
}

func writeManifest(t *testing.T, umbrella string) {
	t.Helper()
	dir := filepath.Join(umbrella, ".multica")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir .multica: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "workspace.toml"), []byte("workspace = \"x\"\n"), 0o644); err != nil {
		t.Fatalf("write workspace.toml: %v", err)
	}
}

func TestResolveUmbrella_FindsManifestAboveRepo(t *testing.T) {
	umbrella := t.TempDir()
	writeManifest(t, umbrella)
	repo := filepath.Join(umbrella, "backend")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	got, err := resolveUmbrella(repo)
	if err != nil {
		t.Fatalf("resolveUmbrella: %v", err)
	}
	// EvalSymlinks the expectation: t.TempDir on macOS resolves through /var symlink.
	wantResolved, _ := filepath.EvalSymlinks(umbrella)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Fatalf("umbrella = %q, want %q", got, umbrella)
	}
}

func TestResolveUmbrella_RejectsRelativePath(t *testing.T) {
	if _, err := resolveUmbrella("./backend"); err == nil {
		t.Fatal("expected error for relative repo path, got nil")
	}
}

func TestResolveUmbrella_RejectsEmptyPath(t *testing.T) {
	if _, err := resolveUmbrella(""); err == nil {
		t.Fatal("expected error for empty repo path, got nil")
	}
}

func TestResolveUmbrella_NoManifestAbove(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "backend")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if _, err := resolveUmbrella(repo); err == nil {
		t.Fatal("expected error when no manifest exists above repo, got nil")
	}
}
