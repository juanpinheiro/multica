package feature_test

import (
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/feature"
)

func TestValidateBranchSlug(t *testing.T) {
	cases := []struct {
		slug    string
		wantErr string // empty = expect nil error; non-empty = substring that must appear in error
	}{
		// Valid slugs
		{"", ""},
		{"auth", ""},
		{"auth-v2", ""},
		{"todo-v3", ""},
		{"123", ""},
		{"my-feature", ""},
		{"UPPER", ""},

		// Contains feature/ prefix — most specific rejection
		{"feature/x", "feature/"},
		{"feature/auth-v2", "feature/"},

		// Contains other path separators
		{"feat/x", "/"},
		{"auth/v2", "/"},
		{"a/b/c", "/"},

		// Invalid git-ref sequences
		{"auth..v2", ".."},
		{"a..b..c", ".."},
		{"auth@{v2", "@{"},

		// Starts/ends with dot
		{".auth", "dot"},
		{"auth.", "dot"},

		// Ends with .lock
		{"auth.lock", ".lock"},
		{"v2.lock", ".lock"},

		// Invalid individual characters
		{"auth v2", "invalid"},
		{"auth~v2", "invalid"},
		{"auth^v2", "invalid"},
		{"auth:v2", "invalid"},
		{"auth?v2", "invalid"},
		{"auth*v2", "invalid"},
		{"auth[v2", "invalid"},
		{"auth\\v2", "invalid"},
	}

	for _, tc := range cases {
		t.Run("slug="+tc.slug, func(t *testing.T) {
			err := feature.ValidateBranchSlug(tc.slug)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBranchSlug(%q) = %v, want nil", tc.slug, err)
				}
				return
			}
			if err == nil {
				t.Errorf("ValidateBranchSlug(%q) = nil, want error containing %q", tc.slug, tc.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ValidateBranchSlug(%q) error = %q, want it to contain %q", tc.slug, err.Error(), tc.wantErr)
			}
		})
	}
}
