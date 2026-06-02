// Package initiative holds the pure domain logic for an Initiative — the
// PRD-level container that maps onto the upstream `feature` table. The status
// state machine here is the single source of truth for which lifecycle
// transitions are legal; the control plane (MCP, UI) and the execution plane
// (daemon claim, completion hooks) both route status changes through it.
package initiative

import "fmt"

// Status is an Initiative's lifecycle state.
//
//	draft → ready → running → in_review → done
//
// with blocked (a tripwire/dependency pause) and cancelled as off-ramps. done
// and cancelled are terminal.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusReady     Status = "ready"
	StatusRunning   Status = "running"
	StatusInReview  Status = "in_review"
	StatusDone      Status = "done"
	StatusBlocked   Status = "blocked"
	StatusCancelled Status = "cancelled"
)

// allowed maps each status to the states it may transition into. A status
// absent from a list is an illegal transition; terminal states have no targets.
var allowed = map[Status][]Status{
	StatusDraft:     {StatusReady, StatusCancelled},
	StatusReady:     {StatusRunning, StatusBlocked, StatusCancelled},
	StatusRunning:   {StatusInReview, StatusDone, StatusBlocked, StatusCancelled},
	StatusInReview:  {StatusDone, StatusRunning, StatusCancelled},
	StatusBlocked:   {StatusReady, StatusRunning, StatusCancelled},
	StatusDone:      {},
	StatusCancelled: {},
}

// AllStatuses returns every valid status in lifecycle order.
func AllStatuses() []Status {
	return []Status{
		StatusDraft, StatusReady, StatusRunning, StatusInReview,
		StatusDone, StatusBlocked, StatusCancelled,
	}
}

// Valid reports whether s is a known Initiative status.
func (s Status) Valid() bool {
	_, ok := allowed[s]
	return ok
}

// IsTerminal reports whether s is an end state with no outgoing transitions.
func (s Status) IsTerminal() bool {
	targets, ok := allowed[s]
	return ok && len(targets) == 0
}

// CanTransition reports whether moving from → to is legal. Unknown statuses and
// self-transitions are never legal.
func CanTransition(from, to Status) bool {
	for _, target := range allowed[from] {
		if target == to {
			return true
		}
	}
	return false
}

// Transition validates a from → to move, returning a descriptive error when it
// is illegal so callers can surface a 4xx instead of silently no-op'ing.
func Transition(from, to Status) error {
	if !from.Valid() {
		return fmt.Errorf("initiative: unknown source status %q", from)
	}
	if !to.Valid() {
		return fmt.Errorf("initiative: unknown target status %q", to)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("initiative: illegal transition %s → %s", from, to)
	}
	return nil
}
