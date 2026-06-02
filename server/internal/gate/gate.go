// Package gate holds the pure claim predicate — the deterministic core of
// orchestration (ADR-0004). Given a fully-resolved World, Claimable answers
// "may this Run be claimed, and if not, which gate holds it back?".
//
// This module is the canonical specification of the predicate. The atomic SQL
// implementation lives in ClaimAgentTask (server/pkg/db/queries/agent.sql),
// which owns resolution and row-locking; the Orchestrator (issue 10) consumes
// this module to reason about what to dispatch. The two are kept in lockstep:
// MilestoneGateOpen here mirrors the milestone NOT EXISTS branch in that query.
package gate

// Reason explains why a Run is or is not claimable. It is the "which reason"
// half of the predicate.
type Reason string

const (
	ReasonClaimable          Reason = "claimable"
	ReasonAgentBusy          Reason = "agent_busy"
	ReasonInitiativeInactive Reason = "initiative_inactive"
	ReasonMilestoneGated     Reason = "milestone_gated"
	ReasonDependencyUnmet    Reason = "dependency_unmet"
	ReasonBranchHeld         Reason = "branch_held"
)

// OK reports whether r means the Run is claimable.
func (r Reason) OK() bool { return r == ReasonClaimable }

// ValidationPassed is the milestone validation status that opens the gate for
// the next Milestone's Issues. Mirrors milestone.validation_status.
const ValidationPassed = "passed"

// Milestone is the minimal shape the gate reasons over: its ordinal within an
// Initiative and whether it has passed validation.
type Milestone struct {
	ID               string
	Position         int
	ValidationStatus string
}

// World is the fully-resolved state the predicate reasons over. Every field is
// a digested fact, not a raw DB row, so the gate stays pure.
type World struct {
	// InitiativeActive: the Run's Initiative is 'ready' or 'running'. True when
	// the Issue belongs to no Initiative (ungoverned work stays claimable).
	// The resolver folds in one role-aware exception: a retrospective Run
	// (issue 19) is dispatched at the 'in_review' transition, so it stays active
	// while the Initiative is 'in_review'. See the ClaimAgentTask SQL gate.
	InitiativeActive bool
	// Milestones are every Milestone in the Run's Initiative; MilestoneID is the
	// Milestone its Issue belongs to ("" for none).
	Milestones  []Milestone
	MilestoneID string
	// DependenciesMet: all blocking ('blocks'/'blocked_by') dependencies are done.
	DependenciesMet bool
	// AgentFree: no other active Run for the same agent is on this Issue.
	AgentFree bool
	// BranchFree: no other active Run holds this Run's (repo, branch).
	BranchFree bool
}

// Claimable returns ReasonClaimable when every gate is open, or the first
// failing reason in a fixed precedence.
func Claimable(w World) Reason {
	switch {
	case !w.AgentFree:
		return ReasonAgentBusy
	case !w.InitiativeActive:
		return ReasonInitiativeInactive
	case !MilestoneGateOpen(w.Milestones, w.MilestoneID):
		return ReasonMilestoneGated
	case !w.DependenciesMet:
		return ReasonDependencyUnmet
	case !w.BranchFree:
		return ReasonBranchHeld
	default:
		return ReasonClaimable
	}
}

// MilestoneGateOpen reports whether the Issue assigned to targetID may be
// claimed under milestone-gating: every Milestone ordered before the target (a
// strictly lower Position in the same Initiative) must have passed validation.
// An Issue with no Milestone (empty targetID) or no earlier siblings is open.
func MilestoneGateOpen(all []Milestone, targetID string) bool {
	if targetID == "" {
		return true
	}
	target, ok := find(all, targetID)
	if !ok {
		return true
	}
	for _, m := range all {
		if m.Position < target.Position && m.ValidationStatus != ValidationPassed {
			return false
		}
	}
	return true
}

func find(all []Milestone, id string) (Milestone, bool) {
	for _, m := range all {
		if m.ID == id {
			return m, true
		}
	}
	return Milestone{}, false
}
