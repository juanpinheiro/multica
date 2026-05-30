package feature_test

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/feature"
	"github.com/multica-ai/multica/server/internal/workspace/execmode"
)

func TestPlan(t *testing.T) {
	issue := feature.Issue{Identifier: "MUL-1", Metadata: nil}
	feat := &feature.Feature{Identifier: "feat-id", BranchSlug: ptr("auth-v2")}
	repo := &feature.Repo{Name: "backend", DefaultBranch: "main"}
	umbrella := "/home/dev/meu-produto"

	cases := []struct {
		name         string
		mode         string
		issue        feature.Issue
		feat         *feature.Feature
		repo         *feature.Repo
		umbrellaDir  string
		wantBranch   string
		wantLocation feature.Location
		wantUmbrella string
		wantParallel bool
	}{
		{
			name:         "worktree mode → isolated worktree, parallel",
			mode:         execmode.Worktree,
			issue:        issue,
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "feature/auth-v2",
			wantLocation: feature.LocationWorktree,
			wantUmbrella: "",
			wantParallel: true,
		},
		{
			name:         "in_place mode → umbrella, serial",
			mode:         execmode.InPlace,
			issue:        issue,
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "feature/auth-v2",
			wantLocation: feature.LocationUmbrella,
			wantUmbrella: umbrella,
			wantParallel: false,
		},
		{
			name:         "branch name is the same regardless of mode",
			mode:         execmode.InPlace,
			issue:        issue,
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "feature/auth-v2",
			wantLocation: feature.LocationUmbrella,
			wantUmbrella: umbrella,
			wantParallel: false,
		},
		{
			name:         "worktree with no feature → issue branch",
			mode:         execmode.Worktree,
			issue:        feature.Issue{Identifier: "MUL-99", Metadata: nil},
			feat:         nil,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "issue/MUL-99",
			wantLocation: feature.LocationWorktree,
			wantUmbrella: "",
			wantParallel: true,
		},
		{
			name:         "in_place with no feature → issue branch, serial",
			mode:         execmode.InPlace,
			issue:        feature.Issue{Identifier: "MUL-99", Metadata: nil},
			feat:         nil,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "issue/MUL-99",
			wantLocation: feature.LocationUmbrella,
			wantUmbrella: umbrella,
			wantParallel: false,
		},
		{
			name: "worktree with per-issue metadata override",
			mode: execmode.Worktree,
			issue: feature.Issue{
				Identifier: "MUL-500",
				Metadata:   map[string]any{"target_branch": "hotfix/urgent"},
			},
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "hotfix/urgent",
			wantLocation: feature.LocationWorktree,
			wantUmbrella: "",
			wantParallel: true,
		},
		{
			name: "in_place with per-issue metadata override",
			mode: execmode.InPlace,
			issue: feature.Issue{
				Identifier: "MUL-500",
				Metadata:   map[string]any{"target_branch": "hotfix/urgent"},
			},
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "hotfix/urgent",
			wantLocation: feature.LocationUmbrella,
			wantUmbrella: umbrella,
			wantParallel: false,
		},
		// Unknown mode falls back to worktree.
		{
			name:         "unknown mode defaults to worktree",
			mode:         "bogus",
			issue:        issue,
			feat:         feat,
			repo:         repo,
			umbrellaDir:  umbrella,
			wantBranch:   "feature/auth-v2",
			wantLocation: feature.LocationWorktree,
			wantUmbrella: "",
			wantParallel: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := feature.Plan(tc.mode, tc.issue, tc.feat, tc.repo, tc.umbrellaDir)
			if got.Branch != tc.wantBranch {
				t.Errorf("Branch = %q, want %q", got.Branch, tc.wantBranch)
			}
			if got.Location != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got.Location, tc.wantLocation)
			}
			if got.UmbrellaDir != tc.wantUmbrella {
				t.Errorf("UmbrellaDir = %q, want %q", got.UmbrellaDir, tc.wantUmbrella)
			}
			if got.Parallel != tc.wantParallel {
				t.Errorf("Parallel = %v, want %v", got.Parallel, tc.wantParallel)
			}
		})
	}
}

// TestPlan_BranchIndependentOfMode verifies the branch name is identical
// for both worktree and in-place modes given the same (issue, feature, repo).
func TestPlan_BranchIndependentOfMode(t *testing.T) {
	issue := feature.Issue{Identifier: "MUL-42", Metadata: nil}
	feat := &feature.Feature{Identifier: "cross-id", BranchSlug: ptr("cross-repo-auth")}
	repo := &feature.Repo{Name: "backend", DefaultBranch: "main"}

	worktree := feature.Plan(execmode.Worktree, issue, feat, repo, "/umbrella")
	inplace := feature.Plan(execmode.InPlace, issue, feat, repo, "/umbrella")

	if worktree.Branch != inplace.Branch {
		t.Errorf("branch differs by mode: worktree=%q in_place=%q", worktree.Branch, inplace.Branch)
	}
}

// TestPlan_BranchIndependentOfRepo verifies the branch name is the same
// for two different repos given the same (issue, feature) — consistent with
// feature.Resolve guaranteeing repo-independence.
func TestPlan_BranchIndependentOfRepo(t *testing.T) {
	issue := feature.Issue{Identifier: "MUL-1", Metadata: nil}
	feat := &feature.Feature{Identifier: "auth-v2-id", BranchSlug: ptr("auth-v2")}

	backend := feature.Plan(execmode.Worktree, issue, feat, &feature.Repo{Name: "backend", DefaultBranch: "main"}, "")
	frontend := feature.Plan(execmode.Worktree, issue, feat, &feature.Repo{Name: "frontend", DefaultBranch: "main"}, "")

	if backend.Branch != frontend.Branch {
		t.Errorf("branch differs by repo: backend=%q frontend=%q", backend.Branch, frontend.Branch)
	}
}
