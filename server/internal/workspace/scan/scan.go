package scan

import "path/filepath"

// Entry is a directory entry returned by FS.ReadDir.
type Entry struct {
	Name  string
	IsDir bool
}

// FS abstracts filesystem operations for the scanner.
type FS interface {
	ReadDir(dir string) ([]Entry, error)
	// DirExists reports whether path is an accessible directory.
	DirExists(path string) bool
}

// GitRunner abstracts git operations.
type GitRunner interface {
	// RemoteURL returns the remote URL for origin of the git repo at dir.
	// Returns an error when the directory is not a git repo or has no origin.
	RemoteURL(dir string) (string, error)
}

// Candidate is a git repository found by the scanner.
type Candidate struct {
	Name   string // directory name (basename of Path)
	Path   string // absolute path
	Remote string // origin remote URL (empty if no origin)
}

// skipDirs is the set of directory names that are never scanned.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".git":         true,
}

// Scan walks the direct children of root up to maxDepth looking for git repos.
// It skips ignore dirs: node_modules, vendor, dist, build, .git.
// It does NOT descend into a discovered repo to find nested repos.
// A directory is a candidate when it contains a .git subdirectory.
// maxDepth=2 is the recommended default (one level = direct children, two = children's children).
// The function never panics; a directory that cannot be read is silently skipped.
func Scan(root string, maxDepth int, fs FS, git GitRunner) []Candidate {
	var results []Candidate
	walk(root, 0, maxDepth, fs, git, &results)
	return results
}

// walk recursively descends into dir at the given depth, collecting candidates.
func walk(dir string, depth, maxDepth int, fs FS, git GitRunner, results *[]Candidate) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return
	}

	// Check whether this directory is itself a git repo.
	for _, e := range entries {
		if e.Name == ".git" && e.IsDir {
			remote, _ := git.RemoteURL(dir)
			*results = append(*results, Candidate{
				Name:   filepath.Base(dir),
				Path:   dir,
				Remote: remote,
			})
			// Do not recurse further into a discovered repo.
			return
		}
	}

	// Not a repo — recurse into subdirectories if depth budget remains.
	if depth >= maxDepth {
		return
	}
	for _, e := range entries {
		if !e.IsDir || skipDirs[e.Name] {
			continue
		}
		child := filepath.Join(dir, e.Name)
		walk(child, depth+1, maxDepth, fs, git, results)
	}
}
