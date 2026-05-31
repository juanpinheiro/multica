package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
	"github.com/multica-ai/multica/server/internal/workspace/resolve"
)

// applyResolvedWorkspace runs manifest-driven resolution and stamps the result
// onto client. It is a no-op when the workspace was already set explicitly
// (legacy --workspace-id / MULTICA_WORKSPACE_ID) or inside an agent task, where
// the daemon is the sole authority on workspace identity.
//
// Resolution failures degrade silently: the client keeps whatever workspace it
// had (none), so the downstream command surfaces its own "workspace required"
// error rather than a confusing resolution error.
func applyResolvedWorkspace(cmd *cobra.Command, client *cli.APIClient) {
	serverURL := client.BaseURL
	token := client.Token
	if serverURL == "" || token == "" {
		return
	}

	startDir, err := os.Getwd()
	if err != nil {
		return
	}

	profile := resolveProfile(cmd)
	cfg, _ := cli.LoadCLIConfigForProfile(profile)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := resolve.Resolve(ctx, &serverAdapter{baseURL: serverURL, token: token}, resolve.Inputs{
		Override: cli.FlagOrEnv(cmd, "workspace", "MULTICA_WORKSPACE", ""),
		StartDir: startDir,
		LastUsed: cfg.WorkspaceID,
		FS:       osFS{},
		Git:      osGit{},
		Warnf:    func(format string, args ...any) { fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...) },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: workspace resolution failed: %v\n", err)
		return
	}
	if res.Source == resolve.SourceNone {
		return
	}

	client.WorkspaceID = res.Workspace.ID
	client.WorkspaceSlug = res.Workspace.Slug

	persistLastUsed(profile, cfg, res.Workspace.ID)
}

// persistLastUsed records the resolved workspace ID so a later session outside
// any repo/umbrella can fall back to it. Only writes when the ID is known and
// changed, to avoid churning the config file on every command.
func persistLastUsed(profile string, cfg cli.CLIConfig, workspaceID string) {
	if workspaceID == "" || cfg.WorkspaceID == workspaceID {
		return
	}
	cfg.WorkspaceID = workspaceID
	_ = cli.SaveCLIConfigForProfile(cfg, profile)
}

// serverAdapter implements resolve.Server over the REST API.
type serverAdapter struct {
	baseURL string
	token   string
}

func (a *serverAdapter) client(workspaceID string) *cli.APIClient {
	return cli.NewAPIClient(a.baseURL, workspaceID, a.token)
}

func (a *serverAdapter) ListWorkspaces(ctx context.Context) ([]resolve.Workspace, error) {
	var raw []workspaceSummary
	if err := a.client("").GetJSON(ctx, "/api/workspaces", &raw); err != nil {
		return nil, err
	}
	out := make([]resolve.Workspace, len(raw))
	for i, ws := range raw {
		out[i] = resolve.Workspace{ID: ws.ID, Slug: ws.Slug, Mode: ws.Mode}
	}
	return out, nil
}

func (a *serverAdapter) CreateWorkspace(ctx context.Context, slug, name, mode string) (resolve.Workspace, error) {
	body := map[string]string{"name": name, "slug": slug, "mode": mode}
	var ws workspaceSummary
	if err := a.client("").PostJSON(ctx, "/api/workspaces", body, &ws); err != nil {
		return resolve.Workspace{}, err
	}
	return resolve.Workspace{ID: ws.ID, Slug: ws.Slug, Mode: ws.Mode}, nil
}

func (a *serverAdapter) UpdateWorkspaceMode(ctx context.Context, workspaceID, mode string) error {
	body := map[string]string{"mode": mode}
	return a.client(workspaceID).PatchJSON(ctx, "/api/workspaces/"+workspaceID, body, nil)
}

func (a *serverAdapter) ListRepoNames(ctx context.Context, workspaceID string) ([]string, error) {
	repos, err := a.listRepos(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}
	return names, nil
}

func (a *serverAdapter) CreateRepo(ctx context.Context, workspaceID string, repo resolve.RepoInput) error {
	body := map[string]any{"name": repo.Name, "remote_url": repo.RemoteURL}
	if repo.LocalPath != "" {
		body["local_path"] = repo.LocalPath
	}
	return a.client(workspaceID).PostJSON(ctx, "/api/repos", body, nil)
}

func (a *serverAdapter) FindWorkspaceByRemote(ctx context.Context, remoteURL string) (resolve.Workspace, bool, error) {
	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return resolve.Workspace{}, false, err
	}
	want := normalizeRemote(remoteURL)
	for _, ws := range workspaces {
		repos, err := a.listRepos(ctx, ws.ID)
		if err != nil {
			continue
		}
		for _, r := range repos {
			if normalizeRemote(r.RemoteURL) == want {
				return ws, true, nil
			}
		}
	}
	return resolve.Workspace{}, false, nil
}

type repoSummary struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
}

func (a *serverAdapter) listRepos(ctx context.Context, workspaceID string) ([]repoSummary, error) {
	var repos []repoSummary
	if err := a.client(workspaceID).GetJSON(ctx, "/api/repos", &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// normalizeRemote canonicalizes a git remote so equivalent forms compare equal:
// it drops the scheme, a trailing ".git", a trailing slash, and lowercases.
func normalizeRemote(remote string) string {
	r := strings.TrimSpace(strings.ToLower(remote))
	for _, prefix := range []string{"https://", "http://", "ssh://", "git://"} {
		r = strings.TrimPrefix(r, prefix)
	}
	if i := strings.LastIndex(r, "@"); i != -1 {
		r = r[i+1:] // drop user@ in scp-style URLs
	}
	r = strings.TrimSuffix(r, "/")
	r = strings.TrimSuffix(r, ".git")
	return strings.Replace(r, ":", "/", 1) // git@host:owner/repo → host/owner/repo
}

// osFS is the production resolve.FS backed by the real filesystem.
type osFS struct{}

func (osFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (osFS) DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// osGit is the production resolve.Git that shells out to git.
type osGit struct{}

func (osGit) OriginURL(dir string) (string, bool) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	url := strings.TrimSpace(string(out))
	return url, url != ""
}
