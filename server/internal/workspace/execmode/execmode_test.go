package execmode_test

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/workspace/execmode"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		wantMode  string
		wantKnown bool
	}{
		{name: "absent defaults to worktree", raw: "", wantMode: execmode.Worktree, wantKnown: true},
		{name: "explicit worktree", raw: "worktree", wantMode: execmode.Worktree, wantKnown: true},
		{name: "explicit in_place", raw: "in_place", wantMode: execmode.InPlace, wantKnown: true},
		{name: "unknown falls back to worktree and is not known", raw: "isolated", wantMode: execmode.Worktree, wantKnown: false},
		{name: "case-sensitive: In_Place is unknown", raw: "In_Place", wantMode: execmode.Worktree, wantKnown: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mode, known := execmode.Normalize(tc.raw)
			if mode != tc.wantMode {
				t.Errorf("mode = %q, want %q", mode, tc.wantMode)
			}
			if known != tc.wantKnown {
				t.Errorf("known = %v, want %v", known, tc.wantKnown)
			}
		})
	}
}
