package feature

// FeatureForBranch holds the fields from a feature row needed for branch resolution.
type FeatureForBranch struct {
	TargetBranch *string
}

// IssueForBranch holds the fields from an issue row needed for branch resolution.
type IssueForBranch struct {
	Identifier string
	Metadata   map[string]any
}

// Resolve returns the git branch an issue's task should target, and whether
// that branch is shared with sibling issues of the same feature.
//
// Priority:
//  1. feature.TargetBranch (non-empty) → shared branch
//  2. issue.Metadata["target_branch"] (non-empty string) → per-issue override
//  3. derived "issue/<Identifier>" → isolated branch
func Resolve(i IssueForBranch, f *FeatureForBranch) (branch string, shared bool) {
	if f != nil && f.TargetBranch != nil && *f.TargetBranch != "" {
		return *f.TargetBranch, true
	}
	if s, ok := i.Metadata["target_branch"].(string); ok && s != "" {
		return s, false
	}
	return "issue/" + i.Identifier, false
}
