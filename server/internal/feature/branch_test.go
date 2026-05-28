package feature_test

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/feature"
)

func ptr(s string) *string { return &s }

func TestResolve(t *testing.T) {
	cases := []struct {
		name         string
		issue        feature.IssueForBranch
		f            *feature.FeatureForBranch
		wantBranch   string
		wantShared   bool
	}{
		{
			name:       "feature is nil",
			issue:      feature.IssueForBranch{Identifier: "MUL-487", Metadata: nil},
			f:          nil,
			wantBranch: "issue/MUL-487",
			wantShared: false,
		},
		{
			name:       "feature TargetBranch nil, issue Metadata empty",
			issue:      feature.IssueForBranch{Identifier: "MUL-100", Metadata: map[string]any{}},
			f:          &feature.FeatureForBranch{TargetBranch: nil},
			wantBranch: "issue/MUL-100",
			wantShared: false,
		},
		{
			name:       "feature TargetBranch nil, issue Metadata nil",
			issue:      feature.IssueForBranch{Identifier: "MUL-200", Metadata: nil},
			f:          &feature.FeatureForBranch{TargetBranch: nil},
			wantBranch: "issue/MUL-200",
			wantShared: false,
		},
		{
			name:       "feature TargetBranch set",
			issue:      feature.IssueForBranch{Identifier: "MUL-300", Metadata: nil},
			f:          &feature.FeatureForBranch{TargetBranch: ptr("feature/auth-v2")},
			wantBranch: "feature/auth-v2",
			wantShared: true,
		},
		{
			name: "issue Metadata target_branch set, feature TargetBranch nil",
			issue: feature.IssueForBranch{
				Identifier: "MUL-400",
				Metadata:   map[string]any{"target_branch": "issue/my-override"},
			},
			f:          &feature.FeatureForBranch{TargetBranch: nil},
			wantBranch: "issue/my-override",
			wantShared: false,
		},
		{
			name: "both set — feature TargetBranch wins over issue Metadata",
			issue: feature.IssueForBranch{
				Identifier: "MUL-500",
				Metadata:   map[string]any{"target_branch": "issue/per-issue"},
			},
			f:          &feature.FeatureForBranch{TargetBranch: ptr("feature/shared")},
			wantBranch: "feature/shared",
			wantShared: true,
		},
		{
			name: "issue Metadata target_branch empty string falls through",
			issue: feature.IssueForBranch{
				Identifier: "MUL-600",
				Metadata:   map[string]any{"target_branch": ""},
			},
			f:          &feature.FeatureForBranch{TargetBranch: nil},
			wantBranch: "issue/MUL-600",
			wantShared: false,
		},
		{
			name: "issue Metadata target_branch wrong type falls through",
			issue: feature.IssueForBranch{
				Identifier: "MUL-700",
				Metadata:   map[string]any{"target_branch": 42},
			},
			f:          &feature.FeatureForBranch{TargetBranch: nil},
			wantBranch: "issue/MUL-700",
			wantShared: false,
		},
		{
			name:       "feature TargetBranch empty string falls through",
			issue:      feature.IssueForBranch{Identifier: "MUL-800", Metadata: nil},
			f:          &feature.FeatureForBranch{TargetBranch: ptr("")},
			wantBranch: "issue/MUL-800",
			wantShared: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			branch, shared := feature.Resolve(tc.issue, tc.f)
			if branch != tc.wantBranch {
				t.Errorf("branch = %q, want %q", branch, tc.wantBranch)
			}
			if shared != tc.wantShared {
				t.Errorf("shared = %v, want %v", shared, tc.wantShared)
			}
		})
	}
}
