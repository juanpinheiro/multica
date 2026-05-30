package feature_test

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/feature"
)

func ptr(s string) *string { return &s }

func TestResolve(t *testing.T) {
	cases := []struct {
		name       string
		issue      feature.Issue
		f          *feature.Feature
		r          *feature.Repo
		wantBranch string
		wantShared bool
	}{
		// No feature — falls through to per-issue branch.
		{
			name:       "feature nil",
			issue:      feature.Issue{Identifier: "MUL-487", Metadata: nil},
			f:          nil,
			wantBranch: "issue/MUL-487",
			wantShared: false,
		},
		// Feature present with Identifier but no BranchSlug.
		{
			name:       "feature present, BranchSlug nil → feature/<Identifier>",
			issue:      feature.Issue{Identifier: "MUL-100", Metadata: nil},
			f:          &feature.Feature{Identifier: "some-uuid", BranchSlug: nil},
			wantBranch: "feature/some-uuid",
			wantShared: true,
		},
		// Feature with explicit BranchSlug overrides Identifier.
		{
			name:       "feature BranchSlug set → feature/<BranchSlug>",
			issue:      feature.Issue{Identifier: "MUL-300", Metadata: nil},
			f:          &feature.Feature{Identifier: "fallback-id", BranchSlug: ptr("auth-v2")},
			wantBranch: "feature/auth-v2",
			wantShared: true,
		},
		// Empty BranchSlug falls back to Identifier.
		{
			name:       "feature BranchSlug empty string → falls back to Identifier",
			issue:      feature.Issue{Identifier: "MUL-800", Metadata: nil},
			f:          &feature.Feature{Identifier: "my-feature-id", BranchSlug: ptr("")},
			wantBranch: "feature/my-feature-id",
			wantShared: true,
		},
		// Metadata override wins over everything — highest priority.
		{
			name: "issue Metadata target_branch wins over feature BranchSlug",
			issue: feature.Issue{
				Identifier: "MUL-500",
				Metadata:   map[string]any{"target_branch": "issue/per-issue"},
			},
			f:          &feature.Feature{Identifier: "feat-id", BranchSlug: ptr("shared")},
			wantBranch: "issue/per-issue",
			wantShared: false,
		},
		// Metadata override wins even when feature has no BranchSlug.
		{
			name: "issue Metadata target_branch wins over feature with no BranchSlug",
			issue: feature.Issue{
				Identifier: "MUL-400",
				Metadata:   map[string]any{"target_branch": "issue/my-override"},
			},
			f:          &feature.Feature{Identifier: "feat-id", BranchSlug: nil},
			wantBranch: "issue/my-override",
			wantShared: false,
		},
		// Empty metadata target_branch is skipped.
		{
			name: "issue Metadata target_branch empty string falls through to feature",
			issue: feature.Issue{
				Identifier: "MUL-600",
				Metadata:   map[string]any{"target_branch": ""},
			},
			f:          &feature.Feature{Identifier: "feat-id", BranchSlug: nil},
			wantBranch: "feature/feat-id",
			wantShared: true,
		},
		// Wrong type in metadata is skipped.
		{
			name: "issue Metadata target_branch wrong type falls through to feature",
			issue: feature.Issue{
				Identifier: "MUL-700",
				Metadata:   map[string]any{"target_branch": 42},
			},
			f:          &feature.Feature{Identifier: "feat-id", BranchSlug: nil},
			wantBranch: "feature/feat-id",
			wantShared: true,
		},
		// No feature and empty metadata → per-issue branch.
		{
			name: "no feature, metadata empty → issue branch",
			issue: feature.Issue{
				Identifier: "MUL-200",
				Metadata:   map[string]any{},
			},
			f:          nil,
			wantBranch: "issue/MUL-200",
			wantShared: false,
		},
		// Branch name is identical across two different repos.
		{
			name:       "repo A → same branch as repo B for same feature",
			issue:      feature.Issue{Identifier: "MUL-999", Metadata: nil},
			f:          &feature.Feature{Identifier: "cross-repo-feat", BranchSlug: ptr("auth-v2")},
			r:          &feature.Repo{Name: "backend", DefaultBranch: "main"},
			wantBranch: "feature/auth-v2",
			wantShared: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			branch, shared := feature.Resolve(tc.issue, tc.f, tc.r)
			if branch != tc.wantBranch {
				t.Errorf("branch = %q, want %q", branch, tc.wantBranch)
			}
			if shared != tc.wantShared {
				t.Errorf("shared = %v, want %v", shared, tc.wantShared)
			}
		})
	}
}

// TestResolve_SameBranchAcrossRepos verifies that the branch name is independent
// of the repo: two calls with different Repo values but the same (issue, feature)
// must return identical branch names.
func TestResolve_SameBranchAcrossRepos(t *testing.T) {
	issue := feature.Issue{Identifier: "MUL-1", Metadata: nil}
	f := &feature.Feature{Identifier: "auth-v2-id", BranchSlug: ptr("auth-v2")}

	branchA, _ := feature.Resolve(issue, f, &feature.Repo{Name: "backend", DefaultBranch: "main"})
	branchB, _ := feature.Resolve(issue, f, &feature.Repo{Name: "frontend", DefaultBranch: "main"})

	if branchA != branchB {
		t.Errorf("branch differs by repo: backend=%q frontend=%q", branchA, branchB)
	}
}
