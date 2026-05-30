package resolve_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/resolve"
)

// fakeServer is an in-memory Server. listErr forces ListWorkspaces to fail so
// tests can exercise the offline-degradation paths.
type fakeServer struct {
	workspaces []resolve.Workspace
	repoNames  map[string][]string // workspaceID -> repo names
	byRemote   map[string]resolve.Workspace

	listErr error

	createdWorkspaces []string            // slugs
	createdRepos      map[string][]string // workspaceID -> repo names created
	nextID            func(slug string) string
}

func newFakeServer() *fakeServer {
	return &fakeServer{
		repoNames:    map[string][]string{},
		byRemote:     map[string]resolve.Workspace{},
		createdRepos: map[string][]string{},
		nextID:       func(slug string) string { return "id-" + slug },
	}
}

func (f *fakeServer) ListWorkspaces(context.Context) ([]resolve.Workspace, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.workspaces, nil
}

func (f *fakeServer) CreateWorkspace(_ context.Context, slug, _ string) (resolve.Workspace, error) {
	f.createdWorkspaces = append(f.createdWorkspaces, slug)
	ws := resolve.Workspace{ID: f.nextID(slug), Slug: slug}
	f.workspaces = append(f.workspaces, ws)
	return ws, nil
}

func (f *fakeServer) ListRepoNames(_ context.Context, workspaceID string) ([]string, error) {
	return f.repoNames[workspaceID], nil
}

func (f *fakeServer) CreateRepo(_ context.Context, workspaceID string, repo resolve.RepoInput) error {
	f.createdRepos[workspaceID] = append(f.createdRepos[workspaceID], repo.Name)
	return nil
}

func (f *fakeServer) FindWorkspaceByRemote(_ context.Context, remote string) (resolve.Workspace, bool, error) {
	ws, ok := f.byRemote[remote]
	return ws, ok, nil
}

// memFS is a map-backed filesystem. files holds file contents; dirs the set of
// existing directories.
type memFS struct {
	files map[string]string
	dirs  map[string]bool
}

func (m memFS) ReadFile(path string) ([]byte, error) {
	if c, ok := m.files[path]; ok {
		return []byte(c), nil
	}
	return nil, errors.New("not found")
}

func (m memFS) DirExists(path string) bool { return m.dirs[path] }

type fakeGit map[string]string

func (f fakeGit) OriginURL(dir string) (string, bool) {
	r, ok := f[dir]
	return r, ok
}

func manifestPath(dir string) string {
	return filepath.Join(dir, ".multica", "workspace.toml")
}

func TestResolve_OverrideWinsOverManifest(t *testing.T) {
	srv := newFakeServer()
	srv.workspaces = []resolve.Workspace{{ID: "id-forced", Slug: "forced"}}

	fs := memFS{files: map[string]string{
		manifestPath("/umb"): `workspace = "from-manifest"`,
	}}

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		Override: "forced",
		StartDir: "/umb",
		FS:       fs,
		Git:      fakeGit{},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceOverride {
		t.Fatalf("Source = %q, want override", got.Source)
	}
	if got.Workspace.Slug != "forced" {
		t.Fatalf("Slug = %q, want forced", got.Workspace.Slug)
	}
}

func TestResolve_OverrideUnknownIsError(t *testing.T) {
	srv := newFakeServer()
	srv.workspaces = []resolve.Workspace{{ID: "id-a", Slug: "a"}}

	_, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		Override: "ghost",
		Git:      fakeGit{},
	})
	if err == nil {
		t.Fatal("expected error for unknown override workspace")
	}
}

func TestResolve_OverrideOfflineFallsBackToSlug(t *testing.T) {
	srv := newFakeServer()
	srv.listErr = errors.New("server down")

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		Override: "offline-ws",
		Git:      fakeGit{},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Workspace.Slug != "offline-ws" || got.Source != resolve.SourceOverride {
		t.Fatalf("got %+v, want slug offline-ws via override", got)
	}
}

func TestResolve_ManifestWalkUpResolvesSlug(t *testing.T) {
	srv := newFakeServer()
	// Workspace already exists and is in sync — nothing should be created.
	srv.workspaces = []resolve.Workspace{{ID: "id-meu", Slug: "meu-produto"}}
	srv.repoNames["id-meu"] = []string{"backend"}

	fs := memFS{
		files: map[string]string{
			manifestPath("/umb"): `
workspace = "meu-produto"
[[repo]]
name = "backend"
path = "./backend"
remote = "github.com/x/backend"
`,
		},
		dirs: map[string]bool{filepath.Join("/umb", "backend"): true},
	}

	// Start two levels deep to exercise the walk-up.
	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: filepath.Join("/umb", "backend", "src"),
		FS:       fs,
		Git:      fakeGit{},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceManifest {
		t.Fatalf("Source = %q, want manifest", got.Source)
	}
	if got.Workspace.Slug != "meu-produto" || got.Workspace.ID != "id-meu" {
		t.Fatalf("Workspace = %+v, want meu-produto/id-meu", got.Workspace)
	}
	if len(srv.createdWorkspaces) != 0 || len(srv.createdRepos) != 0 {
		t.Fatalf("in-sync manifest must create nothing; created ws=%v repos=%v", srv.createdWorkspaces, srv.createdRepos)
	}
	if len(got.ManifestRepos) != 1 {
		t.Fatalf("ManifestRepos = %v, want 1", got.ManifestRepos)
	}
}

func TestResolve_ManifestRebuildsAbsentWorkspaceAndRepos(t *testing.T) {
	srv := newFakeServer() // no workspaces — simulates fresh/wiped DB

	fs := memFS{
		files: map[string]string{
			manifestPath("/umb"): `
workspace = "meu-produto"
[[repo]]
name = "backend"
path = "./backend"
remote = "github.com/x/backend"
[[repo]]
name = "frontend"
path = "./frontend"
remote = "github.com/x/frontend"
`,
		},
		dirs: map[string]bool{
			filepath.Join("/umb", "backend"): true,
			// frontend intentionally missing on disk → warning, not deletion.
		},
	}

	var warnings []string
	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: "/umb",
		FS:       fs,
		Git:      fakeGit{},
		Warnf:    func(format string, args ...any) { warnings = append(warnings, format) },
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(srv.createdWorkspaces) != 1 || srv.createdWorkspaces[0] != "meu-produto" {
		t.Fatalf("createdWorkspaces = %v, want [meu-produto]", srv.createdWorkspaces)
	}
	created := srv.createdRepos[got.Workspace.ID]
	if len(created) != 2 {
		t.Fatalf("created repos = %v, want backend+frontend", created)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one missing-on-disk warning", warnings)
	}
}

func TestResolve_ManifestRebuildIsIdempotent(t *testing.T) {
	srv := newFakeServer()
	fs := memFS{
		files: map[string]string{
			manifestPath("/umb"): `
workspace = "meu-produto"
[[repo]]
name = "backend"
path = "./backend"
remote = "github.com/x/backend"
`,
		},
		dirs: map[string]bool{filepath.Join("/umb", "backend"): true},
	}
	in := resolve.Inputs{StartDir: "/umb", FS: fs, Git: fakeGit{}}

	first, err := resolve.Resolve(context.Background(), srv, in)
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	// Simulate the server now knowing about the repo the first run created.
	srv.repoNames[first.Workspace.ID] = []string{"backend"}

	second, err := resolve.Resolve(context.Background(), srv, in)
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if first.Workspace.ID != second.Workspace.ID {
		t.Fatalf("ids differ across runs: %q vs %q", first.Workspace.ID, second.Workspace.ID)
	}
	if len(srv.createdWorkspaces) != 1 {
		t.Fatalf("re-run created workspaces again: %v", srv.createdWorkspaces)
	}
	if got := len(srv.createdRepos[first.Workspace.ID]); got != 1 {
		t.Fatalf("re-run created repos again: total %d, want 1", got)
	}
}

func TestResolve_GitRemoteLookup(t *testing.T) {
	srv := newFakeServer()
	srv.byRemote["github.com/x/detached"] = resolve.Workspace{ID: "id-d", Slug: "detached-ws"}

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: "/worktree",
		FS:       memFS{}, // no manifest above
		Git:      fakeGit{"/worktree": "github.com/x/detached"},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceGitRemote || got.Workspace.Slug != "detached-ws" {
		t.Fatalf("got %+v, want git_remote/detached-ws", got)
	}
}

func TestResolve_SingleWorkspace(t *testing.T) {
	srv := newFakeServer()
	srv.workspaces = []resolve.Workspace{{ID: "only", Slug: "only-ws"}}

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: "/nowhere",
		FS:       memFS{},
		Git:      fakeGit{},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceSingle || got.Workspace.ID != "only" {
		t.Fatalf("got %+v, want single/only", got)
	}
}

func TestResolve_LastUsedFallback(t *testing.T) {
	srv := newFakeServer()
	// Two workspaces → single-workspace rule does not apply.
	srv.workspaces = []resolve.Workspace{{ID: "a", Slug: "a"}, {ID: "b", Slug: "b"}}

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: "/nowhere",
		FS:       memFS{},
		Git:      fakeGit{},
		LastUsed: "remembered-id",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceLastUsed || got.Workspace.ID != "remembered-id" {
		t.Fatalf("got %+v, want last_used/remembered-id", got)
	}
}

func TestResolve_NoneTriggersOnboarding(t *testing.T) {
	srv := newFakeServer() // zero workspaces

	got, err := resolve.Resolve(context.Background(), srv, resolve.Inputs{
		StartDir: "/nowhere",
		FS:       memFS{},
		Git:      fakeGit{},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != resolve.SourceNone {
		t.Fatalf("Source = %q, want none", got.Source)
	}
}
