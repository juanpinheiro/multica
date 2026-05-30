package manifest_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/manifest"
)

// memFS is a map from absolute path to file bytes.
// Keys use filepath.Join separators (OS-native).
type memFS map[string][]byte

func (m memFS) ReadFile(path string) ([]byte, error) {
	if b, ok := m[path]; ok {
		return b, nil
	}
	return nil, errors.New("not found")
}

func TestFind(t *testing.T) {
	cases := []struct {
		name     string
		fs       memFS
		startDir string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "no manifest anywhere",
			fs:       memFS{},
			startDir: filepath.Join("C:\\", "some", "deep", "dir"),
			wantPath: "",
			wantOK:   false,
		},
		{
			name: "manifest in cwd",
			fs: memFS{
				filepath.Join("C:\\", "project", ".multica", "workspace.toml"): []byte(`workspace = "proj"`),
			},
			startDir: filepath.Join("C:\\", "project"),
			wantPath: filepath.Join("C:\\", "project", ".multica", "workspace.toml"),
			wantOK:   true,
		},
		{
			name: "manifest several levels up",
			fs: memFS{
				filepath.Join("C:\\", "umbrella", ".multica", "workspace.toml"): []byte(`workspace = "umbrella"`),
			},
			startDir: filepath.Join("C:\\", "umbrella", "a", "b", "c"),
			wantPath: filepath.Join("C:\\", "umbrella", ".multica", "workspace.toml"),
			wantOK:   true,
		},
		{
			name: "root boundary, no manifest",
			fs: memFS{
				filepath.Join("C:\\"): nil,
			},
			startDir: filepath.Join("C:\\"),
			wantPath: "",
			wantOK:   false,
		},
		{
			name: "single-repo manifest inside repo root",
			fs: memFS{
				filepath.Join("C:\\", "repo", ".multica", "workspace.toml"): []byte(`workspace = "single"`),
			},
			startDir: filepath.Join("C:\\", "repo"),
			wantPath: filepath.Join("C:\\", "repo", ".multica", "workspace.toml"),
			wantOK:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := manifest.Find(tc.startDir, tc.fs)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.wantPath {
				t.Errorf("path = %q, want %q", got, tc.wantPath)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Run("valid TOML one repo", func(t *testing.T) {
		data := []byte(`
workspace = "meu-produto"

[[repo]]
name   = "backend"
path   = "./backend"
remote = "github.com/voce/backend"
`)
		m, err := manifest.Parse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Workspace != "meu-produto" {
			t.Errorf("Workspace = %q, want %q", m.Workspace, "meu-produto")
		}
		if len(m.Repos) != 1 {
			t.Fatalf("len(Repos) = %d, want 1", len(m.Repos))
		}
		r := m.Repos[0]
		if r.Name != "backend" {
			t.Errorf("Repos[0].Name = %q, want %q", r.Name, "backend")
		}
		if r.Path != "./backend" {
			t.Errorf("Repos[0].Path = %q, want %q", r.Path, "./backend")
		}
		if r.Remote != "github.com/voce/backend" {
			t.Errorf("Repos[0].Remote = %q, want %q", r.Remote, "github.com/voce/backend")
		}
	})

	t.Run("valid TOML multiple repos", func(t *testing.T) {
		data := []byte(`
workspace = "monorepo"

[[repo]]
name   = "api"
path   = "./api"
remote = "github.com/org/api"

[[repo]]
name   = "web"
path   = "./web"
remote = "github.com/org/web"

[[repo]]
name   = "mobile"
path   = "./mobile"
remote = "github.com/org/mobile"
`)
		m, err := manifest.Parse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Workspace != "monorepo" {
			t.Errorf("Workspace = %q, want %q", m.Workspace, "monorepo")
		}
		if len(m.Repos) != 3 {
			t.Fatalf("len(Repos) = %d, want 3", len(m.Repos))
		}
		names := []string{"api", "web", "mobile"}
		for i, want := range names {
			if m.Repos[i].Name != want {
				t.Errorf("Repos[%d].Name = %q, want %q", i, m.Repos[i].Name, want)
			}
		}
	})

	t.Run("malformed TOML", func(t *testing.T) {
		data := []byte(`workspace = [unclosed`)
		_, err := manifest.Parse(data)
		if err == nil {
			t.Error("expected error for malformed TOML, got nil")
		}
	})

	t.Run("empty workspace name", func(t *testing.T) {
		data := []byte(`
[[repo]]
name = "only-repo"
path = "./only"
remote = "github.com/org/only"
`)
		m, err := manifest.Parse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Workspace != "" {
			t.Errorf("Workspace = %q, want empty string", m.Workspace)
		}
		if len(m.Repos) != 1 {
			t.Fatalf("len(Repos) = %d, want 1", len(m.Repos))
		}
	})
}

func TestManifestDir(t *testing.T) {
	manifestPath := filepath.Join("C:\\", "umbrella", ".multica", "workspace.toml")
	got := manifest.ManifestDir(manifestPath)
	want := filepath.Join("C:\\", "umbrella")
	if got != want {
		t.Errorf("ManifestDir = %q, want %q", got, want)
	}
}

func TestRepoPathResolution(t *testing.T) {
	manifestPath := filepath.Join("C:\\", "umbrella", ".multica", "workspace.toml")
	dir := manifest.ManifestDir(manifestPath)
	got := filepath.Join(dir, "./backend")
	want := filepath.Join("C:\\", "umbrella", "backend")
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
}
