package feature

// Issue holds the fields from an issue row needed for branch resolution.
type Issue struct {
	Identifier string
	Metadata   map[string]any
}

// Feature holds the fields from a feature row needed for branch resolution.
// Identifier is always non-empty (e.g. the feature UUID); BranchSlug is an
// explicit human-readable override that wins over Identifier when set.
type Feature struct {
	Identifier string
	BranchSlug *string
}

// Repo holds the repo context for branch resolution. The branch name is
// independent of the repo; Repo is carried in the signature so callers
// (e.g. the claim gate in Issue 03) have all three dimensions in one call.
type Repo struct {
	Name          string
	DefaultBranch string
}

// Resolve returns the git branch an issue's task should target, and whether
// that branch is shared with sibling issues of the same feature.
//
// Priority (highest to lowest):
//  1. issue.Metadata["target_branch"] non-empty → per-issue override, shared=false
//  2. feature != nil → "feature/<BranchSlug ?? Identifier>", shared=true
//  3. "issue/<Identifier>", shared=false
func Resolve(i Issue, f *Feature, r *Repo) (branch string, shared bool) {
	if s, ok := i.Metadata["target_branch"].(string); ok && s != "" {
		return s, false
	}
	if f != nil {
		slug := f.Identifier
		if f.BranchSlug != nil && *f.BranchSlug != "" {
			slug = *f.BranchSlug
		}
		return "feature/" + slug, true
	}
	return "issue/" + i.Identifier, false
}
