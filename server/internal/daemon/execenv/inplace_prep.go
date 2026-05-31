package execenv

import (
	"fmt"
	"os/exec"
	"strings"
)

// InPlacePrepError reports why an in-place repo could not be safely prepared.
// Its message is safe to forward verbatim onto a task's failure comment.
type InPlacePrepError struct {
	RepoDir string
	Reason  string
}

func (e *InPlacePrepError) Error() string {
	return fmt.Sprintf("in-place repo %q cannot be prepared: %s", e.RepoDir, e.Reason)
}

// PrepareInPlaceRepo puts repoDir on the target branch for an in-place run,
// refusing to act over in-progress work. It fails fast — without switching
// branches — when the working tree is dirty or HEAD is on a branch other than
// the repo's default or the target itself. When the tree is clean and on an
// expected branch it checks out target (creating it from HEAD when absent), so
// an in-place run converges into the same feature/<slug> branch a worktree run
// would, inside the developer's real repo.
func PrepareInPlaceRepo(repoDir, target string) error {
	if target == "" {
		return &InPlacePrepError{repoDir, "no target branch resolved for this task"}
	}
	dirty, err := repoIsDirty(repoDir)
	if err != nil {
		return &InPlacePrepError{repoDir, err.Error()}
	}
	if dirty {
		return &InPlacePrepError{repoDir, "working tree has uncommitted changes; commit or stash them before running in-place"}
	}
	current, err := currentBranch(repoDir)
	if err != nil {
		return &InPlacePrepError{repoDir, err.Error()}
	}
	if current == target {
		return nil
	}
	if def := localDefaultBranch(repoDir); def == "" || current != def {
		return &InPlacePrepError{repoDir, fmt.Sprintf(
			"on branch %q, not the default branch or target %q; refusing to switch branches over in-progress work",
			current, target)}
	}
	return checkoutBranch(repoDir, target)
}

// repoIsDirty reports whether repoDir has uncommitted changes (tracked or
// untracked) in its working tree.
func repoIsDirty(repoDir string) (bool, error) {
	out, err := gitOutput(repoDir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// currentBranch returns the short name of the branch HEAD points at.
func currentBranch(repoDir string) (string, error) {
	out, err := gitOutput(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// localDefaultBranch returns the repo's default branch — origin/HEAD when a
// remote is configured, else whichever of main/master exists locally, else "".
func localDefaultBranch(repoDir string) string {
	if out, err := gitOutput(repoDir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		ref := strings.TrimSpace(out) // "origin/main"
		if i := strings.LastIndex(ref, "/"); i >= 0 {
			return ref[i+1:]
		}
		return ref
	}
	for _, b := range []string{"main", "master"} {
		if gitRun(repoDir, "rev-parse", "--verify", "--quiet", b) == nil {
			return b
		}
	}
	return ""
}

// checkoutBranch switches repoDir to branch, creating it from the current HEAD
// when it does not yet exist locally.
func checkoutBranch(repoDir, branch string) error {
	if gitRun(repoDir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch) == nil {
		if err := gitRun(repoDir, "checkout", branch); err != nil {
			return &InPlacePrepError{repoDir, fmt.Sprintf("checkout %q: %s", branch, err)}
		}
		return nil
	}
	if err := gitRun(repoDir, "checkout", "-b", branch); err != nil {
		return &InPlacePrepError{repoDir, fmt.Sprintf("create branch %q: %s", branch, err)}
	}
	return nil
}

// gitOutput runs a git command in repoDir and returns its stdout.
func gitOutput(repoDir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repoDir}, args...)...).Output()
	return string(out), err
}

// gitRun runs a git command in repoDir, discarding output.
func gitRun(repoDir string, args ...string) error {
	return exec.Command("git", append([]string{"-C", repoDir}, args...)...).Run()
}
