package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/workspace/execmode"
	"github.com/multica-ai/multica/server/internal/workspace/inplace"
)

// execModeIsInPlace reports whether the workspace runs in in-place mode. An
// empty or unrecognized value normalizes to worktree, so the daemon defaults to
// the isolated, parallel path unless the manifest explicitly opts in.
func execModeIsInPlace(raw string) bool {
	mode, _ := execmode.Normalize(raw)
	return mode == execmode.InPlace
}

// prepareInPlace readies an in-place run: it locates and validates the umbrella
// directory, acquires the per-umbrella lock (parking the task in
// waiting_local_directory while another in-place task holds it), and puts the
// issue's real repo on its feature/<slug> branch — failing fast over a dirty or
// off-branch tree. On success the returned ReleaseFunc must be held for the rest
// of the run so the workspace stays serial; on any error the lock is released
// before returning so a later task is never wedged.
func (d *Daemon) prepareInPlace(ctx context.Context, task Task, taskLog *slog.Logger) (string, inplace.ReleaseFunc, error) {
	umbrella, err := resolveUmbrella(task.RepoLocalPath)
	if err != nil {
		return "", nil, fmt.Errorf("in-place: %w", err)
	}
	if err := inplace.ValidateDir(umbrella); err != nil {
		return "", nil, fmt.Errorf("in-place: %w", err)
	}

	waited := false
	release, err := d.umbrellaLocker.Acquire(ctx, umbrella, task.ID, func(holder string) {
		waited = true
		if werr := d.client.WaitForLocalDirectory(ctx, task.ID, inplace.WaitReason(umbrella, holder)); werr != nil {
			taskLog.Warn("in-place: post waiting status failed", "error", werr)
		}
	})
	if err != nil {
		return "", nil, fmt.Errorf("in-place: acquire umbrella lock: %w", err)
	}
	// Clear the parked waiting state back to running once we hold the lock.
	if waited {
		if serr := d.client.StartTask(ctx, task.ID); serr != nil {
			taskLog.Warn("in-place: clear waiting status failed", "error", serr)
		}
	}

	if err := execenv.PrepareInPlaceRepo(task.RepoLocalPath, task.TargetBranch); err != nil {
		release()
		return "", nil, err
	}
	return umbrella, release, nil
}

// resolveUmbrella finds the workspace umbrella directory for an in-place run:
// the nearest ancestor of the issue's repo that holds a .multica/workspace.toml
// manifest. A feature's repos are declared as children of the umbrella, so
// walking up from any one of them lands on it. repoPath must be absolute — an
// in-place workspace's repos resolve to real on-disk paths, and a relative or
// empty path cannot anchor the walk.
func resolveUmbrella(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("in-place run needs the issue's repo path to locate the workspace umbrella")
	}
	if !filepath.IsAbs(repoPath) {
		return "", fmt.Errorf("in-place repo path %q is not absolute; cannot locate the workspace umbrella", repoPath)
	}
	dir := filepath.Clean(repoPath)
	for {
		manifest := filepath.Join(dir, ".multica", "workspace.toml")
		if fi, err := os.Stat(manifest); err == nil && !fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .multica/workspace.toml found at or above %q", repoPath)
		}
		dir = parent
	}
}
