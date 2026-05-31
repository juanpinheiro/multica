package feature

import "github.com/multica-ai/multica/server/internal/workspace/execmode"

// Location describes where a task's agent runs.
type Location string

const (
	// LocationWorktree runs the agent in an isolated per-repo git worktree.
	LocationWorktree Location = "worktree"
	// LocationUmbrella runs the agent in the workspace's real umbrella directory.
	LocationUmbrella Location = "umbrella"
)

// RunTarget is the result of execution-mode resolution: where and how a task runs.
type RunTarget struct {
	// Branch is the git branch the task targets. Identical for both modes.
	Branch string
	// Location is where the agent runs.
	Location Location
	// UmbrellaDir is the workspace umbrella root for LocationUmbrella runs.
	// Empty for LocationWorktree.
	UmbrellaDir string
	// Parallel is true when this task can run concurrently with other tasks
	// in the same workspace. Worktree tasks are parallel-eligible; in-place
	// tasks are serial (one at a time per workspace).
	Parallel bool
}

// Plan resolves where and how a task should run, extending Resolve with mode
// awareness. The Branch in the result is always identical to what Resolve
// returns for the same (i, f, r) inputs; only Location and Parallel differ
// by mode.
//
// Unknown mode values fall back to worktree behavior.
func Plan(mode string, i Issue, f *Feature, r *Repo, umbrellaDir string) RunTarget {
	branch, _ := Resolve(i, f, r)
	if mode == execmode.InPlace {
		return RunTarget{
			Branch:      branch,
			Location:    LocationUmbrella,
			UmbrellaDir: umbrellaDir,
			Parallel:    false,
		}
	}
	return RunTarget{
		Branch:   branch,
		Location: LocationWorktree,
		Parallel: true,
	}
}
