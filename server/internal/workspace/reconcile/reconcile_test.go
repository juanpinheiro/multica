package reconcile_test

import (
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/manifest"
	"github.com/multica-ai/multica/server/internal/workspace/reconcile"
	"github.com/multica-ai/multica/server/internal/workspace/scan"
)

func TestReconcile(t *testing.T) {
	repoPath := func(name string) string {
		return filepath.Join("/", "umbrella", name)
	}

	cases := []struct {
		name string
		m    manifest.Manifest
		srv  reconcile.WorkspaceState
		want reconcile.Plan
	}{
		{
			name: "workspace absent on server yields CreateWorkspace true",
			m:    manifest.Manifest{Workspace: "myws"},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  false,
				ServerRepoNames:  map[string]bool{},
				RepoDiskPresence: map[string]bool{},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{
				CreateWorkspace: true,
			},
		},
		{
			name: "workspace present yields CreateWorkspace false",
			m:    manifest.Manifest{Workspace: "myws"},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{},
				RepoDiskPresence: map[string]bool{},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{
				CreateWorkspace: false,
			},
		},
		{
			name: "manifest repo absent on server is in ReposToCreate",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "api", Path: repoPath("api"), Remote: "git@github.com:org/api.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{},
				RepoDiskPresence: map[string]bool{repoPath("api"): true},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{
				ReposToCreate: []manifest.RepoEntry{
					{Name: "api", Path: repoPath("api"), Remote: "git@github.com:org/api.git"},
				},
			},
		},
		{
			name: "manifest repo present on server is not in ReposToCreate",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "api", Path: repoPath("api"), Remote: "git@github.com:org/api.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{"api": true},
				RepoDiskPresence: map[string]bool{repoPath("api"): true},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{},
		},
		{
			name: "manifest repo path missing from disk is in ReposMissingOnDisk by name",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "web", Path: repoPath("web"), Remote: "git@github.com:org/web.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{"web": true},
				RepoDiskPresence: map[string]bool{repoPath("web"): false},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{
				ReposMissingOnDisk: []string{"web"},
			},
		},
		{
			name: "manifest repo path exists on disk is not in ReposMissingOnDisk",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "web", Path: repoPath("web"), Remote: "git@github.com:org/web.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{"web": true},
				RepoDiskPresence: map[string]bool{repoPath("web"): true},
				ScannedRepos:     nil,
			},
			want: reconcile.Plan{},
		},
		{
			name: "orphan git repo on disk not in manifest is in ReposOrphanOnDisk",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos:     []manifest.RepoEntry{},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{},
				RepoDiskPresence: map[string]bool{},
				ScannedRepos: []scan.Candidate{
					{Name: "orphan", Path: repoPath("orphan"), Remote: "git@github.com:org/orphan.git"},
				},
			},
			want: reconcile.Plan{
				ReposOrphanOnDisk: []scan.Candidate{
					{Name: "orphan", Path: repoPath("orphan"), Remote: "git@github.com:org/orphan.git"},
				},
			},
		},
		{
			name: "orphan git repo on disk that is in manifest is not in ReposOrphanOnDisk",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "known", Path: repoPath("known"), Remote: "git@github.com:org/known.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{"known": true},
				RepoDiskPresence: map[string]bool{repoPath("known"): true},
				ScannedRepos: []scan.Candidate{
					{Name: "known", Path: repoPath("known"), Remote: "git@github.com:org/known.git"},
				},
			},
			want: reconcile.Plan{},
		},
		{
			name: "fully in-sync manifest yields empty Plan",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "api", Path: repoPath("api"), Remote: "git@github.com:org/api.git"},
					{Name: "web", Path: repoPath("web"), Remote: "git@github.com:org/web.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists:  true,
				ServerRepoNames:  map[string]bool{"api": true, "web": true},
				RepoDiskPresence: map[string]bool{repoPath("api"): true, repoPath("web"): true},
				ScannedRepos: []scan.Candidate{
					{Name: "api", Path: repoPath("api"), Remote: "git@github.com:org/api.git"},
					{Name: "web", Path: repoPath("web"), Remote: "git@github.com:org/web.git"},
				},
			},
			want: reconcile.Plan{},
		},
		{
			name: "multiple repos mix of missing from server and present",
			m: manifest.Manifest{
				Workspace: "ws",
				Repos: []manifest.RepoEntry{
					{Name: "existing", Path: repoPath("existing"), Remote: "git@github.com:org/existing.git"},
					{Name: "missing", Path: repoPath("missing"), Remote: "git@github.com:org/missing.git"},
					{Name: "also-missing", Path: repoPath("also-missing"), Remote: "git@github.com:org/also-missing.git"},
				},
			},
			srv: reconcile.WorkspaceState{
				WorkspaceExists: true,
				ServerRepoNames: map[string]bool{
					"existing": true,
					// "missing" and "also-missing" are not on the server
				},
				RepoDiskPresence: map[string]bool{
					repoPath("existing"):     true,
					repoPath("missing"):      true,
					repoPath("also-missing"): true,
				},
				ScannedRepos: nil,
			},
			want: reconcile.Plan{
				ReposToCreate: []manifest.RepoEntry{
					{Name: "missing", Path: repoPath("missing"), Remote: "git@github.com:org/missing.git"},
					{Name: "also-missing", Path: repoPath("also-missing"), Remote: "git@github.com:org/also-missing.git"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reconcile.Reconcile(tc.m, tc.srv)

			if got.CreateWorkspace != tc.want.CreateWorkspace {
				t.Errorf("CreateWorkspace = %v, want %v", got.CreateWorkspace, tc.want.CreateWorkspace)
			}

			if len(got.ReposToCreate) != len(tc.want.ReposToCreate) {
				t.Errorf("len(ReposToCreate) = %d, want %d; got %v, want %v",
					len(got.ReposToCreate), len(tc.want.ReposToCreate),
					repoNames(got.ReposToCreate), repoNames(tc.want.ReposToCreate))
			} else {
				for i, r := range got.ReposToCreate {
					w := tc.want.ReposToCreate[i]
					if r.Name != w.Name || r.Path != w.Path || r.Remote != w.Remote {
						t.Errorf("ReposToCreate[%d] = %+v, want %+v", i, r, w)
					}
				}
			}

			if len(got.ReposMissingOnDisk) != len(tc.want.ReposMissingOnDisk) {
				t.Errorf("len(ReposMissingOnDisk) = %d, want %d; got %v, want %v",
					len(got.ReposMissingOnDisk), len(tc.want.ReposMissingOnDisk),
					got.ReposMissingOnDisk, tc.want.ReposMissingOnDisk)
			} else {
				for i, name := range got.ReposMissingOnDisk {
					if name != tc.want.ReposMissingOnDisk[i] {
						t.Errorf("ReposMissingOnDisk[%d] = %q, want %q", i, name, tc.want.ReposMissingOnDisk[i])
					}
				}
			}

			if len(got.ReposOrphanOnDisk) != len(tc.want.ReposOrphanOnDisk) {
				t.Errorf("len(ReposOrphanOnDisk) = %d, want %d; got %v, want %v",
					len(got.ReposOrphanOnDisk), len(tc.want.ReposOrphanOnDisk),
					candidateNames(got.ReposOrphanOnDisk), candidateNames(tc.want.ReposOrphanOnDisk))
			} else {
				for i, c := range got.ReposOrphanOnDisk {
					w := tc.want.ReposOrphanOnDisk[i]
					if c.Name != w.Name || c.Path != w.Path || c.Remote != w.Remote {
						t.Errorf("ReposOrphanOnDisk[%d] = %+v, want %+v", i, c, w)
					}
				}
			}
		})
	}
}

func repoNames(repos []manifest.RepoEntry) []string {
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}
	return names
}

func candidateNames(cs []scan.Candidate) []string {
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c.Name
	}
	return names
}
