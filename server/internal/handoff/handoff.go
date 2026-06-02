// Package handoff holds the pure domain logic for the Handoff entity — the
// structured output a worker Run records when it finishes an Issue. The
// Orchestrator (issue 10) reads the latest Handoff to resume fresh context
// after a restart without re-running work already done.
package handoff

import (
	"encoding/json"
	"fmt"
	"time"
)

// CommandResult records a single command invocation and its exit code.
type CommandResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
}

// Handoff is the structured output a worker Run records when it finishes an Issue.
type Handoff struct {
	ID          string
	IssueID     string
	RunID       string
	Done        []string
	LeftUndone  []string
	Commands    []CommandResult
	Discoveries []string
	CreatedAt   time.Time
}

// State is the current observable state of an Issue as derived from its Handoffs.
// The Orchestrator reads this on wake to know where work stands.
type State struct {
	Done        []string
	LeftUndone  []string
	Discoveries []string
}

// LatestState derives the current state from an ordered list of Handoffs. It
// returns the most recent Handoff's view: the agent accumulates history in each
// Handoff it writes, so the last entry is the authoritative current state.
// Returns a zero State (all empty non-nil slices) when handoffs is empty.
func LatestState(handoffs []Handoff) State {
	if len(handoffs) == 0 {
		return State{Done: []string{}, LeftUndone: []string{}, Discoveries: []string{}}
	}
	last := handoffs[len(handoffs)-1]
	return State{
		Done:        nonNil(last.Done),
		LeftUndone:  nonNil(last.LeftUndone),
		Discoveries: nonNil(last.Discoveries),
	}
}

// SerializeCommands encodes a commands slice to JSON bytes for storage.
// Returns a JSON empty array when cmds is nil or empty.
func SerializeCommands(cmds []CommandResult) ([]byte, error) {
	if len(cmds) == 0 {
		return []byte("[]"), nil
	}
	raw, err := json.Marshal(cmds)
	if err != nil {
		return nil, fmt.Errorf("handoff: serialize commands: %w", err)
	}
	return raw, nil
}

// ParseCommands decodes a JSON-encoded commands array from storage.
// Returns nil, nil for an empty or nil input.
func ParseCommands(raw []byte) ([]CommandResult, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var cmds []CommandResult
	if err := json.Unmarshal(raw, &cmds); err != nil {
		return nil, fmt.Errorf("handoff: parse commands: %w", err)
	}
	return cmds, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
