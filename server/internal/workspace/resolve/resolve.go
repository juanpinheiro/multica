// Package resolve determines the active workspace for a CLI/MCP session.
//
// Resolution walks a fixed precedence (see Resolve) anchored on the .multica
// manifest so the user never has to pass a workspace flag: they open a session
// wherever they already are and the manifest above them, the git remote of the
// current repo, or a sensible fallback determines the context.
package resolve

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/multica-ai/multica/server/internal/workspace/manifest"
	"github.com/multica-ai/multica/server/internal/workspace/reconcile"
)

// Workspace is the minimal identity the resolver returns and the consumer
// stamps onto API requests. Slug is always set when known; ID is set when the
// server has been consulted.
type Workspace struct {
	ID   string
	Slug string
}

// RepoInput is a repo the resolver asks the server to register during a
// manifest rebuild.
type RepoInput struct {
	Name      string
	RemoteURL string
	LocalPath string
}

// Server is the subset of the API the resolver needs. The CLI provides a thin
// adapter over the real APIClient; tests provide an in-memory fake.
type Server interface {
	// ListWorkspaces returns every workspace the caller can access.
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
	// CreateWorkspace registers a new workspace and returns it with its ID.
	CreateWorkspace(ctx context.Context, slug, name string) (Workspace, error)
	// ListRepoNames returns the names of repos registered under a workspace.
	ListRepoNames(ctx context.Context, workspaceID string) ([]string, error)
	// CreateRepo registers a repo under a workspace.
	CreateRepo(ctx context.Context, workspaceID string, repo RepoInput) error
	// FindWorkspaceByRemote returns the workspace owning a repo whose remote
	// matches, if any.
	FindWorkspaceByRemote(ctx context.Context, remoteURL string) (Workspace, bool, error)
}

// FS is the filesystem the resolver reads through. It satisfies manifest.FS.
type FS interface {
	ReadFile(path string) ([]byte, error)
	DirExists(path string) bool
}

// Git reads the origin remote of the repo containing dir.
type Git interface {
	OriginURL(dir string) (string, bool)
}

// Inputs carries everything the resolver reads from the environment.
type Inputs struct {
	// Override is an explicit workspace slug/id from --workspace or
	// MULTICA_WORKSPACE. When set it wins over every other signal.
	Override string
	// StartDir is the directory resolution walks up from (typically cwd).
	StartDir string
	// LastUsed is the workspace remembered from a prior session (config).
	LastUsed string
	FS       FS
	Git      Git
	// Warnf surfaces non-fatal problems (e.g. a manifest repo missing on
	// disk). It must never panic on a nil receiver; callers pass a no-op when
	// they don't care.
	Warnf func(format string, args ...any)
}

// Source identifies which precedence rule produced the result.
type Source string

const (
	SourceOverride  Source = "override"
	SourceManifest  Source = "manifest"
	SourceGitRemote Source = "git_remote"
	SourceSingle    Source = "single"
	SourceLastUsed  Source = "last_used"
	// SourceNone means nothing resolved — the caller should start onboarding.
	SourceNone Source = "none"
)

// Result is the outcome of resolution.
type Result struct {
	Workspace Workspace
	Source    Source
	// ManifestRepos are the repos declared by the manifest, present only when
	// Source is SourceManifest.
	ManifestRepos []manifest.RepoEntry
}

// Resolve walks the precedence and returns the active workspace.
//
// Precedence, stopping at the first match:
//  1. Override          — --workspace flag or MULTICA_WORKSPACE env.
//  2. Manifest walk-up  — .multica/workspace.toml above StartDir (offline).
//  3. Git remote        — cwd repo's origin matched against the server.
//  4. Single workspace  — exactly one workspace exists.
//  5. Last-used         — the workspace from a prior session.
//  6. None              — onboarding needed.
//
// The manifest branch is offline-tolerant: it always returns the manifest's
// slug even when the server is unreachable, and reconciles (creating the
// workspace and missing repos) on a best-effort basis when it is reachable.
func Resolve(ctx context.Context, srv Server, in Inputs) (Result, error) {
	warnf := in.Warnf
	if warnf == nil {
		warnf = func(string, ...any) {}
	}

	if in.Override != "" {
		return resolveOverride(ctx, srv, in.Override)
	}

	if path, found := manifest.Find(in.StartDir, in.FS); found {
		return resolveManifest(ctx, srv, in.FS, path, warnf)
	}

	if remote, ok := in.Git.OriginURL(in.StartDir); ok {
		if ws, found, err := srv.FindWorkspaceByRemote(ctx, remote); err == nil && found {
			return Result{Workspace: ws, Source: SourceGitRemote}, nil
		}
	}

	if workspaces, err := srv.ListWorkspaces(ctx); err == nil && len(workspaces) == 1 {
		return Result{Workspace: workspaces[0], Source: SourceSingle}, nil
	}

	if in.LastUsed != "" {
		return Result{Workspace: Workspace{ID: in.LastUsed}, Source: SourceLastUsed}, nil
	}

	return Result{Source: SourceNone}, nil
}

// resolveOverride binds an explicit slug/id to a server workspace. When the
// server is unreachable the override is honored as a bare slug so it still
// works offline; when reachable but unknown it is a user error.
func resolveOverride(ctx context.Context, srv Server, override string) (Result, error) {
	workspaces, err := srv.ListWorkspaces(ctx)
	if err != nil {
		return Result{Workspace: Workspace{Slug: override}, Source: SourceOverride}, nil
	}
	for _, ws := range workspaces {
		if ws.Slug == override || ws.ID == override {
			return Result{Workspace: ws, Source: SourceOverride}, nil
		}
	}
	return Result{}, fmt.Errorf("workspace %q not found", override)
}

// resolveManifest parses the manifest, reconciles it against the server, and
// returns its workspace. Server errors degrade to warnings so resolution still
// yields the manifest's slug offline.
func resolveManifest(ctx context.Context, srv Server, fs FS, manifestPath string, warnf func(string, ...any)) (Result, error) {
	data, err := fs.ReadFile(manifestPath)
	if err != nil {
		return Result{}, fmt.Errorf("read manifest: %w", err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return Result{}, fmt.Errorf("parse manifest %s: %w", manifestPath, err)
	}

	ws := Workspace{Slug: m.Workspace}
	dir := manifest.ManifestDir(manifestPath)
	if id, err := applyManifest(ctx, srv, m, dir, fs, warnf); err != nil {
		warnf("workspace %q: reconcile skipped: %v", m.Workspace, err)
	} else {
		ws.ID = id
	}
	return Result{Workspace: ws, Source: SourceManifest, ManifestRepos: m.Repos}, nil
}

// applyManifest reconciles the manifest with the server and applies the plan:
// it creates the workspace when absent, registers missing repos, and warns on
// repos listed in the manifest but absent on disk. It never deletes anything.
// Returns the workspace ID.
func applyManifest(ctx context.Context, srv Server, m manifest.Manifest, dir string, fs FS, warnf func(string, ...any)) (string, error) {
	workspaceID, state, err := serverState(ctx, srv, m, dir, fs)
	if err != nil {
		return "", err
	}

	plan := reconcile.Reconcile(m, state)

	if plan.CreateWorkspace {
		created, err := srv.CreateWorkspace(ctx, m.Workspace, m.Workspace)
		if err != nil {
			return "", err
		}
		workspaceID = created.ID
	}

	for _, repo := range plan.ReposToCreate {
		if err := srv.CreateRepo(ctx, workspaceID, RepoInput{
			Name:      repo.Name,
			RemoteURL: repo.Remote,
			LocalPath: filepath.Join(dir, repo.Path),
		}); err != nil {
			return "", err
		}
	}

	for _, name := range plan.ReposMissingOnDisk {
		warnf("repo %q is listed in the manifest but missing on disk", name)
	}

	return workspaceID, nil
}

// serverState gathers the server's current view of the manifest's workspace
// (existence, ID, registered repo names) plus on-disk presence, ready for
// reconcile.Reconcile. The returned ID is empty when the workspace is absent.
func serverState(ctx context.Context, srv Server, m manifest.Manifest, dir string, fs FS) (string, reconcile.WorkspaceState, error) {
	workspaces, err := srv.ListWorkspaces(ctx)
	if err != nil {
		return "", reconcile.WorkspaceState{}, err
	}

	state := reconcile.WorkspaceState{
		ServerRepoNames:  map[string]bool{},
		RepoDiskPresence: diskPresence(m, dir, fs),
	}
	workspaceID := ""
	for _, ws := range workspaces {
		if ws.Slug == m.Workspace {
			state.WorkspaceExists = true
			workspaceID = ws.ID
			break
		}
	}
	if !state.WorkspaceExists {
		return "", state, nil
	}

	names, err := srv.ListRepoNames(ctx, workspaceID)
	if err != nil {
		return "", reconcile.WorkspaceState{}, err
	}
	for _, n := range names {
		state.ServerRepoNames[n] = true
	}
	return workspaceID, state, nil
}

// diskPresence reports, per manifest repo Path, whether that path exists on disk.
func diskPresence(m manifest.Manifest, dir string, fs FS) map[string]bool {
	presence := make(map[string]bool, len(m.Repos))
	for _, repo := range m.Repos {
		presence[repo.Path] = fs.DirExists(filepath.Join(dir, repo.Path))
	}
	return presence
}
