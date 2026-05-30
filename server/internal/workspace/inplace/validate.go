// Package inplace holds the path-safety primitives for in-place execution:
// validating that a workspace's umbrella directory is safe to run an agent in,
// before the daemon checks out branches in the developer's real repos.
//
// The validator is a pure, OS-aware leaf module with no workspace, feature, or
// scheduler coupling. It takes a path and returns either nil or a typed
// *RejectedError the daemon can forward verbatim onto a task's failure comment.
package inplace

import (
	"fmt"
	"os"
	"path/filepath"
)

// Reason names why a directory was rejected as an in-place execution target.
type Reason string

const (
	// ReasonNotAbsolute is a path that is not absolute.
	ReasonNotAbsolute Reason = "path is not absolute"
	// ReasonSystemRoot is a filesystem root, Windows drive root, UNC root, or a
	// well-known system directory that holds many users' data.
	ReasonSystemRoot Reason = "path is a system or drive root"
	// ReasonHomeDir is the user's home directory itself.
	ReasonHomeDir Reason = "path is the user home directory"
	// ReasonNotExist is a path that does not exist.
	ReasonNotExist Reason = "path does not exist"
	// ReasonNotDir is a path that exists but is not a directory.
	ReasonNotDir Reason = "path is not a directory"
	// ReasonNotWritable is a directory that is not readable and writable.
	ReasonNotWritable Reason = "path is not readable and writable"
)

// RejectedError reports the first rule a candidate in-place target failed. Its
// message is safe to surface verbatim on a task's failure comment.
type RejectedError struct {
	Path   string
	Reason Reason
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("in-place target %q rejected: %s", e.Path, e.Reason)
}

// ValidateDir reports whether dir is safe to use as an in-place execution
// target. It returns nil for a legitimate, writable project directory, or a
// *RejectedError describing the first failed rule.
//
// Rejection rules, in order: a non-absolute path; a system or drive root, UNC
// root, or the user's home directory (checked both as typed and after resolving
// symlinks, so a link to a banned location is also rejected); a path that does
// not exist, is not a directory, or is not readable and writable.
func ValidateDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return &RejectedError{dir, ReasonNotAbsolute}
	}
	if reason, banned := bannedReason(filepath.Clean(dir)); banned {
		return &RejectedError{dir, reason}
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return &RejectedError{dir, ReasonNotExist}
	}
	if reason, banned := bannedReason(resolved); banned {
		return &RejectedError{dir, reason}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return &RejectedError{dir, ReasonNotExist}
	}
	if !info.IsDir() {
		return &RejectedError{dir, ReasonNotDir}
	}
	if !writable(resolved) {
		return &RejectedError{dir, ReasonNotWritable}
	}
	return nil
}

// bannedReason classifies a cleaned, absolute path as a forbidden in-place
// target. Children of these locations (e.g. a project under the home tree) are
// allowed; only the exact roots and home are rejected.
func bannedReason(p string) (Reason, bool) {
	if isRoot(p) {
		return ReasonSystemRoot, true
	}
	for _, root := range systemRoots {
		if p == root {
			return ReasonSystemRoot, true
		}
	}
	for _, home := range homeDirs() {
		if home != "" && p == home {
			return ReasonHomeDir, true
		}
	}
	return "", false
}

// isRoot reports whether p is a filesystem root, a Windows drive root, or a UNC
// share root. Such a path is its own parent.
func isRoot(p string) bool {
	if filepath.Dir(p) == p {
		return true
	}
	vol := filepath.VolumeName(p)
	return vol != "" && (p == vol || p == vol+string(os.PathSeparator))
}

// systemRoots are POSIX directories that hold many users' or the system's data;
// running an agent directly in one is never intended. They are harmless on
// Windows, where real paths never equal these literals.
var systemRoots = []string{"/Users", "/home", "/root", "/etc", "/var", "/usr"}

// homeDirs returns the user's home directory both as reported and with symlinks
// resolved, so a canonical alias of home is also caught.
func homeDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dirs := []string{filepath.Clean(home)}
	if resolved, err := filepath.EvalSymlinks(home); err == nil {
		dirs = append(dirs, resolved)
	}
	return dirs
}

// writable verifies the directory is readable and writable by creating and
// removing a transient probe file.
func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".multica-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	return os.Remove(name) == nil
}
