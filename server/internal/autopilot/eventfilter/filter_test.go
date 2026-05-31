package eventfilter

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		event   string
		filters []string
		want    bool
	}{
		{
			name:    "empty filters matches everything",
			event:   "github.push",
			filters: []string{},
			want:    true,
		},
		{
			name:    "nil filters matches everything",
			event:   "github.push",
			filters: nil,
			want:    true,
		},
		{
			name:    "exact match",
			event:   "github.push",
			filters: []string{"github.push"},
			want:    true,
		},
		{
			name:    "event not in filter",
			event:   "github.push",
			filters: []string{"github.pull_request.opened", "github.issues.opened"},
			want:    false,
		},
		{
			name:    "glob matches pull_request subtype",
			event:   "github.pull_request.opened",
			filters: []string{"github.pull_request.*"},
			want:    true,
		},
		{
			name:    "glob does not match different top-level event",
			event:   "github.push",
			filters: []string{"github.pull_request.*"},
			want:    false,
		},
		{
			name:    "event matches second pattern in filter",
			event:   "github.push",
			filters: []string{"github.pull_request.*", "github.push"},
			want:    true,
		},
		{
			name:    "wildcard alone matches any event",
			event:   "anything.at.all",
			filters: []string{"*"},
			want:    true,
		},
		{
			name:    "malformed pattern skipped, no valid match returns false",
			event:   "github.push",
			filters: []string{"[", "github.pull_request.*"},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Match(tc.event, tc.filters)
			if got != tc.want {
				t.Errorf("Match(%q, %v) = %v, want %v", tc.event, tc.filters, got, tc.want)
			}
		})
	}
}
