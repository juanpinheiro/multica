package scan_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/scan"
)

// memFS represents the filesystem as a set of existing paths.
// dirs maps a directory path to its list of entries in it.
type memFS struct {
	dirs         map[string][]scan.Entry
	existingDirs map[string]bool
}

func (m *memFS) ReadDir(dir string) ([]scan.Entry, error) {
	entries, ok := m.dirs[dir]
	if !ok {
		return nil, errors.New("not found")
	}
	return entries, nil
}

func (m *memFS) DirExists(path string) bool {
	return m.existingDirs[path]
}

// fakeGit maps a directory path to its origin remote URL.
type fakeGit map[string]string

func (f fakeGit) RemoteURL(dir string) (string, error) {
	if r, ok := f[dir]; ok {
		return r, nil
	}
	return "", errors.New("no remote")
}

func TestScan(t *testing.T) {
	cases := []struct {
		name      string
		root      string
		maxDepth  int
		fs        *memFS
		git       fakeGit
		wantNames []string // expected Candidate.Name values (order-independent check done below)
		wantPaths []string
		wantMap   map[string]string // path → remote
	}{
		{
			// Case 1: flat umbrella with 3 child repos.
			name:     "flat umbrella three child repos",
			root:     filepath.Join("/", "home", "repos"),
			maxDepth: 2,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "home", "repos"): {
						{Name: "alpha", IsDir: true},
						{Name: "beta", IsDir: true},
						{Name: "gamma", IsDir: true},
					},
					filepath.Join("/", "home", "repos", "alpha"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "home", "repos", "beta"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "home", "repos", "gamma"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "home", "repos", "alpha", ".git"): true,
					filepath.Join("/", "home", "repos", "beta", ".git"):  true,
					filepath.Join("/", "home", "repos", "gamma", ".git"): true,
				},
			},
			git: fakeGit{
				filepath.Join("/", "home", "repos", "alpha"): "git@github.com:org/alpha.git",
				filepath.Join("/", "home", "repos", "beta"):  "git@github.com:org/beta.git",
				filepath.Join("/", "home", "repos", "gamma"): "git@github.com:org/gamma.git",
			},
			wantMap: map[string]string{
				filepath.Join("/", "home", "repos", "alpha"): "git@github.com:org/alpha.git",
				filepath.Join("/", "home", "repos", "beta"):  "git@github.com:org/beta.git",
				filepath.Join("/", "home", "repos", "gamma"): "git@github.com:org/gamma.git",
			},
		},
		{
			// Case 2: repo at depth 3 is not returned when maxDepth=2.
			name:     "depth bound repo at depth 3 not returned",
			root:     filepath.Join("/", "home"),
			maxDepth: 2,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "home"): {
						{Name: "projects", IsDir: true},
					},
					filepath.Join("/", "home", "projects"): {
						{Name: "deep", IsDir: true},
					},
					filepath.Join("/", "home", "projects", "deep"): {
						{Name: "repo", IsDir: true},
					},
					filepath.Join("/", "home", "projects", "deep", "repo"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "home", "projects", "deep", "repo", ".git"): true,
				},
			},
			git:     fakeGit{},
			wantMap: map[string]string{}, // no candidates expected
		},
		{
			// Case 3: skip-listed dirs are not scanned.
			name:     "skip-listed dirs node_modules vendor dist build",
			root:     filepath.Join("/", "ws"),
			maxDepth: 2,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "ws"): {
						{Name: "node_modules", IsDir: true},
						{Name: "vendor", IsDir: true},
						{Name: "dist", IsDir: true},
						{Name: "build", IsDir: true},
						{Name: "myapp", IsDir: true},
					},
					// node_modules/vendor/dist/build are listed but should never be read.
					filepath.Join("/", "ws", "node_modules"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "ws", "vendor"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "ws", "dist"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "ws", "build"): {
						{Name: ".git", IsDir: true},
					},
					filepath.Join("/", "ws", "myapp"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "ws", "myapp", ".git"): true,
				},
			},
			git: fakeGit{
				filepath.Join("/", "ws", "myapp"): "git@github.com:org/myapp.git",
			},
			wantMap: map[string]string{
				filepath.Join("/", "ws", "myapp"): "git@github.com:org/myapp.git",
			},
		},
		{
			// Case 4: does not descend into a discovered repo (nested repo is not surfaced).
			name:     "does not descend into discovered repo",
			root:     filepath.Join("/", "code"),
			maxDepth: 2,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "code"): {
						{Name: "outer", IsDir: true},
					},
					filepath.Join("/", "code", "outer"): {
						{Name: ".git", IsDir: true},
						{Name: "inner", IsDir: true},
					},
					// inner would be a repo too, but we should never read it.
					filepath.Join("/", "code", "outer", "inner"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "code", "outer", ".git"):        true,
					filepath.Join("/", "code", "outer", "inner", ".git"): true,
				},
			},
			git: fakeGit{
				filepath.Join("/", "code", "outer"): "git@github.com:org/outer.git",
			},
			wantMap: map[string]string{
				filepath.Join("/", "code", "outer"): "git@github.com:org/outer.git",
			},
		},
		{
			// Case 5: a child dir with no .git is ignored (not a candidate).
			name:     "child dir with no git is ignored",
			root:     filepath.Join("/", "base"),
			maxDepth: 1,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "base"): {
						{Name: "nogit", IsDir: true},
					},
					filepath.Join("/", "base", "nogit"): {
						{Name: "src", IsDir: true},
					},
				},
				existingDirs: map[string]bool{},
			},
			git:     fakeGit{},
			wantMap: map[string]string{},
		},
		{
			// Case 6: remote read via fake git runner is included in Candidate.Remote.
			name:     "remote included in candidate",
			root:     filepath.Join("/", "repos"),
			maxDepth: 1,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "repos"): {
						{Name: "proj", IsDir: true},
					},
					filepath.Join("/", "repos", "proj"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "repos", "proj", ".git"): true,
				},
			},
			git: fakeGit{
				filepath.Join("/", "repos", "proj"): "https://github.com/org/proj.git",
			},
			wantMap: map[string]string{
				filepath.Join("/", "repos", "proj"): "https://github.com/org/proj.git",
			},
		},
		{
			// Case 7: dir with no origin remote → Candidate.Remote is empty string.
			name:     "no origin remote yields empty Remote",
			root:     filepath.Join("/", "repos"),
			maxDepth: 1,
			fs: &memFS{
				dirs: map[string][]scan.Entry{
					filepath.Join("/", "repos"): {
						{Name: "local-only", IsDir: true},
					},
					filepath.Join("/", "repos", "local-only"): {
						{Name: ".git", IsDir: true},
					},
				},
				existingDirs: map[string]bool{
					filepath.Join("/", "repos", "local-only", ".git"): true,
				},
			},
			git: fakeGit{}, // no entry → RemoteURL returns error → Remote should be ""
			wantMap: map[string]string{
				filepath.Join("/", "repos", "local-only"): "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scan.Scan(tc.root, tc.maxDepth, tc.fs, tc.git)

			if len(got) != len(tc.wantMap) {
				t.Fatalf("got %d candidates, want %d; paths: %v", len(got), len(tc.wantMap), paths(got))
			}

			for _, c := range got {
				wantRemote, ok := tc.wantMap[c.Path]
				if !ok {
					t.Errorf("unexpected candidate path %q", c.Path)
					continue
				}
				if c.Remote != wantRemote {
					t.Errorf("candidate %q: remote = %q, want %q", c.Path, c.Remote, wantRemote)
				}
				if c.Name != filepath.Base(c.Path) {
					t.Errorf("candidate %q: Name = %q, want %q", c.Path, c.Name, filepath.Base(c.Path))
				}
			}
		})
	}
}

// paths extracts the Path field from a slice of Candidates for error messages.
func paths(cs []scan.Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Path
	}
	return out
}
