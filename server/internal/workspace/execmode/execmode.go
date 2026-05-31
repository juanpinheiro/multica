// Package execmode defines a workspace's execution mode and the single
// normalization rule shared by every layer that reads it (manifest parser,
// reconciler, API handler).
//
// A workspace runs either in an isolated per-repo worktree (the default) or
// in-place in its real umbrella directory. The mode is declared only in the
// workspace manifest; everywhere else it is a value to be projected, never set.
package execmode

const (
	// Worktree runs each task in an isolated per-repo git worktree. Default.
	Worktree = "worktree"
	// InPlace runs the agent in the workspace's real umbrella directory.
	InPlace = "in_place"
)

// Normalize maps a raw mode string to a known execution mode. An empty value
// (mode absent from the manifest) and any unrecognized value both fall back to
// Worktree. The known result distinguishes the two: it is false only for a
// non-empty unrecognized value, signaling the caller to warn about a typo.
func Normalize(raw string) (mode string, known bool) {
	switch raw {
	case Worktree, InPlace:
		return raw, true
	case "":
		return Worktree, true
	default:
		return Worktree, false
	}
}
