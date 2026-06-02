package handoff_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/handoff"
)

// ---------------------------------------------------------------------------
// CommandResult round-trip
// ---------------------------------------------------------------------------

func TestSerializeParseRoundTrip(t *testing.T) {
	want := []handoff.CommandResult{
		{Command: "go build ./...", ExitCode: 0},
		{Command: "go test ./...", ExitCode: 1},
	}
	raw, err := handoff.SerializeCommands(want)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	got, err := handoff.ParseCommands(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Command != want[i].Command {
			t.Errorf("[%d] command: got %q, want %q", i, got[i].Command, want[i].Command)
		}
		if got[i].ExitCode != want[i].ExitCode {
			t.Errorf("[%d] exit_code: got %d, want %d", i, got[i].ExitCode, want[i].ExitCode)
		}
	}
}

func TestSerializeEmpty(t *testing.T) {
	raw, err := handoff.SerializeCommands(nil)
	if err != nil {
		t.Fatalf("serialize nil: %v", err)
	}
	if string(raw) != "[]" {
		t.Errorf("got %q, want %q", raw, "[]")
	}
}

func TestParseEmpty(t *testing.T) {
	cmds, err := handoff.ParseCommands([]byte("[]"))
	if err != nil {
		t.Fatalf("parse []: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("got %d items, want 0", len(cmds))
	}
}

func TestParseNil(t *testing.T) {
	cmds, err := handoff.ParseCommands(nil)
	if err != nil {
		t.Fatalf("parse nil: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("got %d items, want 0", len(cmds))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := handoff.ParseCommands([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCommandResultJSONTags(t *testing.T) {
	cr := handoff.CommandResult{Command: "make test", ExitCode: 0}
	raw, _ := json.Marshal(cr)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["command"]; !ok {
		t.Error("missing json key 'command'")
	}
	if _, ok := m["exit_code"]; !ok {
		t.Error("missing json key 'exit_code'")
	}
}

// ---------------------------------------------------------------------------
// LatestState derivation
// ---------------------------------------------------------------------------

func TestLatestState_Empty(t *testing.T) {
	s := handoff.LatestState(nil)
	if len(s.Done) != 0 || len(s.LeftUndone) != 0 || len(s.Discoveries) != 0 {
		t.Errorf("expected zero state for empty input, got %+v", s)
	}
}

func TestLatestState_Single(t *testing.T) {
	h := handoff.Handoff{
		Done:        []string{"implement feature A", "write tests"},
		LeftUndone:  []string{"fix edge case B"},
		Discoveries: []string{"upstream bug found"},
	}
	s := handoff.LatestState([]handoff.Handoff{h})
	if len(s.Done) != 2 {
		t.Errorf("Done: got %d, want 2", len(s.Done))
	}
	if len(s.LeftUndone) != 1 || s.LeftUndone[0] != "fix edge case B" {
		t.Errorf("LeftUndone: got %v, want [fix edge case B]", s.LeftUndone)
	}
	if len(s.Discoveries) != 1 {
		t.Errorf("Discoveries: got %d, want 1", len(s.Discoveries))
	}
}

// LatestState returns the most recent Handoff's view — the agent accumulates
// history in each handoff so the last one is the authoritative current state.
func TestLatestState_ReturnsLastHandoff(t *testing.T) {
	earlier := handoff.Handoff{
		CreatedAt:   time.Now().Add(-time.Hour),
		Done:        []string{"step one"},
		LeftUndone:  []string{"step two"},
		Discoveries: []string{"discovery A"},
	}
	later := handoff.Handoff{
		CreatedAt:   time.Now(),
		Done:        []string{"step one", "step two"},
		LeftUndone:  []string{},
		Discoveries: []string{"discovery A", "discovery B"},
	}
	s := handoff.LatestState([]handoff.Handoff{earlier, later})
	if len(s.Done) != 2 {
		t.Errorf("Done: got %v, want [step one, step two]", s.Done)
	}
	if len(s.LeftUndone) != 0 {
		t.Errorf("LeftUndone: got %v, want []", s.LeftUndone)
	}
	if len(s.Discoveries) != 2 {
		t.Errorf("Discoveries: got %v, want 2 items", s.Discoveries)
	}
}

func TestLatestState_NilSlicesAreEmpty(t *testing.T) {
	h := handoff.Handoff{
		Done:        nil,
		LeftUndone:  nil,
		Discoveries: nil,
	}
	s := handoff.LatestState([]handoff.Handoff{h})
	if s.Done == nil || s.LeftUndone == nil || s.Discoveries == nil {
		t.Errorf("State fields must not be nil, got %+v", s)
	}
}
