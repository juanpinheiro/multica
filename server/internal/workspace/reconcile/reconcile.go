package reconcile

import (
	"github.com/multica-ai/multica/server/internal/workspace/manifest"
	"github.com/multica-ai/multica/server/internal/workspace/scan"
)

// WorkspaceState is the server's current view of the workspace,
// plus disk presence information computed by the caller before invoking Reconcile.
type WorkspaceState struct {
	// WorkspaceExists is true when the manifest's workspace slug exists on the server.
	WorkspaceExists bool
	// ServerRepoNames is the set of repo names registered on the server within this workspace.
	ServerRepoNames map[string]bool
	// RepoDiskPresence maps each manifest repo's Path field to whether it exists on disk.
	// The caller fills this by checking each repo path before calling Reconcile.
	RepoDiskPresence map[string]bool
	// ScannedRepos are git repos discovered on disk by the scanner.
	ScannedRepos []scan.Candidate
}

// Plan describes the actions needed to reconcile the manifest with the server.
type Plan struct {
	// CreateWorkspace is true when the manifest's workspace slug is absent on the server.
	CreateWorkspace bool
	// ReposToCreate are manifest repos absent from the server.
	ReposToCreate []manifest.RepoEntry
	// ReposMissingOnDisk are the names of manifest repos whose Path does not exist on disk.
	ReposMissingOnDisk []string
	// ReposOrphanOnDisk are git repos found on disk that are absent from the manifest.
	ReposOrphanOnDisk []scan.Candidate
}

// Reconcile computes what actions are needed to bring the server in sync with m.
// It is a pure function: all disk and server queries must be done before calling it.
func Reconcile(m manifest.Manifest, srv WorkspaceState) Plan {
	var p Plan

	p.CreateWorkspace = !srv.WorkspaceExists

	// Build a set of manifest repo names for orphan detection.
	manifestNames := make(map[string]bool, len(m.Repos))
	for _, repo := range m.Repos {
		manifestNames[repo.Name] = true
	}

	for _, repo := range m.Repos {
		if !srv.ServerRepoNames[repo.Name] {
			p.ReposToCreate = append(p.ReposToCreate, repo)
		}
		if !srv.RepoDiskPresence[repo.Path] {
			p.ReposMissingOnDisk = append(p.ReposMissingOnDisk, repo.Name)
		}
	}

	for _, candidate := range srv.ScannedRepos {
		if !manifestNames[candidate.Name] {
			p.ReposOrphanOnDisk = append(p.ReposOrphanOnDisk, candidate)
		}
	}

	return p
}
