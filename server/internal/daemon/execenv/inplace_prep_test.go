package execenv

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initRepo creates a git repo with one commit on a known default branch and
// returns its path. The branch is forced to "main" so the default-branch logic
// is deterministic regardless of the host's init.defaultBranch setting.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main", dir)
	writeFile(t, filepath.Join(dir, "README.md"), "hello")
	runGit(t, dir, "-C", dir, "add", ".")
	commit(t, dir)
	return dir
}

func runGit(t *testing.T, _ string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, out, err)
	}
}

func commit(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "-C", dir, "commit", "--allow-empty", "-m", "snapshot")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func headBranch(t *testing.T, dir string) string {
	t.Helper()
	b, err := currentBranch(dir)
	if err != nil {
		t.Fatalf("currentBranch: %v", err)
	}
	return b
}

func TestPrepareInPlaceRepo_CleanDefaultCreatesTarget(t *testing.T) {
	dir := initRepo(t)
	if err := PrepareInPlaceRepo(dir, "feature/login"); err != nil {
		t.Fatalf("PrepareInPlaceRepo: %v", err)
	}
	if got := headBranch(t, dir); got != "feature/login" {
		t.Fatalf("HEAD = %q, want feature/login", got)
	}
}

func TestPrepareInPlaceRepo_AlreadyOnTargetIsNoOp(t *testing.T) {
	dir := initRepo(t)
	runGit(t, dir, "-C", dir, "checkout", "-b", "feature/login")
	if err := PrepareInPlaceRepo(dir, "feature/login"); err != nil {
		t.Fatalf("PrepareInPlaceRepo: %v", err)
	}
	if got := headBranch(t, dir); got != "feature/login" {
		t.Fatalf("HEAD = %q, want feature/login", got)
	}
}

func TestPrepareInPlaceRepo_ExistingTargetIsCheckedOut(t *testing.T) {
	dir := initRepo(t)
	runGit(t, dir, "-C", dir, "branch", "feature/login")
	if err := PrepareInPlaceRepo(dir, "feature/login"); err != nil {
		t.Fatalf("PrepareInPlaceRepo: %v", err)
	}
	if got := headBranch(t, dir); got != "feature/login" {
		t.Fatalf("HEAD = %q, want feature/login", got)
	}
}

func TestPrepareInPlaceRepo_DirtyTreeFailsFast(t *testing.T) {
	dir := initRepo(t)
	writeFile(t, filepath.Join(dir, "uncommitted.txt"), "wip")

	err := PrepareInPlaceRepo(dir, "feature/login")
	if err == nil {
		t.Fatal("expected error for dirty tree, got nil")
	}
	var prepErr *InPlacePrepError
	if !errors.As(err, &prepErr) {
		t.Fatalf("error = %T, want *InPlacePrepError", err)
	}
	if got := headBranch(t, dir); got != "main" {
		t.Fatalf("HEAD switched to %q on a dirty tree; must stay on main", got)
	}
}

func TestPrepareInPlaceRepo_OffExpectedBranchFailsFast(t *testing.T) {
	dir := initRepo(t)
	runGit(t, dir, "-C", dir, "checkout", "-b", "wip-experiment")

	err := PrepareInPlaceRepo(dir, "feature/login")
	if err == nil {
		t.Fatal("expected error for off-branch repo, got nil")
	}
	var prepErr *InPlacePrepError
	if !errors.As(err, &prepErr) {
		t.Fatalf("error = %T, want *InPlacePrepError", err)
	}
	if got := headBranch(t, dir); got != "wip-experiment" {
		t.Fatalf("HEAD switched to %q; must stay on wip-experiment", got)
	}
}

func TestPrepareInPlaceRepo_EmptyTargetFails(t *testing.T) {
	dir := initRepo(t)
	if err := PrepareInPlaceRepo(dir, ""); err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}
