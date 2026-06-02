// Package orchestrator holds the pure decision core of the Orchestrator — the
// "COO" of one in-flight Initiative (ADR-0004). Given a fully-resolved snapshot
// of an Initiative taken right after a Run completes, Decide returns the actions
// to apply: pass a Milestone with no Definition of Done, dispatch a validator at
// a Milestone boundary, or advance the Initiative toward review.
//
// Like gate and dod, this module is pure (no I/O): the runtime scaffold feeds it
// digested facts and applies the resulting Plan through the dispatch service.
// Statelessness is the point — every wake rebuilds State from the DB, so a
// process restart mid-run resumes with nothing lost.
package orchestrator

// validationPassed mirrors milestone.validation_status (and gate.ValidationPassed):
// the value that marks a Milestone validated.
const validationPassed = "passed"

// Review is the status an Initiative advances to once every Milestone is
// validated and all its Issues are done — the single human review gate. The
// running→done step is owned by the PR-merge gate (issue 13), not by completion.
const Review = "in_review"

// Milestone is the minimal shape Decide reasons over for the "all validated"
// roll-up: its identity and whether it has passed validation.
type Milestone struct {
	ID               string
	ValidationStatus string
}

// State is the fully-resolved snapshot of one Initiative after a Run completes,
// centered on the Issue whose Run triggered the wake. Every field is a digested
// fact, not a raw DB row, so the decision stays pure.
type State struct {
	// InitiativeStatus is the Initiative's current lifecycle status.
	InitiativeStatus string
	// TriggerIssueDone reports whether the triggering Issue has actually reached
	// done. A worker Run can complete before its Issue is marked done, so a
	// boundary is only real once this is true.
	TriggerIssueDone bool

	// HasMilestone reports whether the triggering Issue belongs to a Milestone.
	HasMilestone bool
	// TriggerMilestoneID identifies the triggering Issue's Milestone.
	TriggerMilestoneID string
	// TriggerMilestoneValidation is that Milestone's validation_status.
	TriggerMilestoneValidation string
	// TriggerMilestoneHasAssertions reports whether it carries a DoD.
	TriggerMilestoneHasAssertions bool
	// TriggerMilestoneOpenSiblings counts non-done Issues in that Milestone,
	// excluding the triggering Issue. Zero means the Milestone's work is complete.
	TriggerMilestoneOpenSiblings int
	// TriggerMilestoneActiveValidators counts in-flight validator Runs for it, so
	// the boundary trigger never dispatches a duplicate.
	TriggerMilestoneActiveValidators int

	// FeatureOpenSiblings counts non-done Issues across the whole Initiative,
	// excluding the triggering Issue. Zero is a precondition for advancing.
	FeatureOpenSiblings int
	// AllMilestones is every Milestone in the Initiative (any order).
	AllMilestones []Milestone
}

// Plan is the set of actions the runtime applies through the dispatch service.
// Zero values mean "nothing to do".
type Plan struct {
	// PassMilestone marks the triggering Milestone validated (no DoD to check).
	PassMilestone bool
	// DispatchValidator enqueues a validator Run for the triggering Milestone.
	DispatchValidator bool
	// AdvanceTo is the Initiative's next status, or "" when it should not move.
	AdvanceTo string
}

// Decide returns the orchestration actions for the snapshot.
func Decide(s State) Plan {
	var p Plan
	if s.atBoundary() {
		switch {
		case !s.TriggerMilestoneHasAssertions:
			p.PassMilestone = true
		case s.TriggerMilestoneActiveValidators == 0:
			p.DispatchValidator = true
		}
	}
	if s.canAdvance(p.PassMilestone) {
		p.AdvanceTo = Review
	}
	return p
}

// atBoundary reports whether the triggering Issue is the last open Issue of a
// not-yet-validated Milestone — the point where validation is due.
func (s State) atBoundary() bool {
	return s.TriggerIssueDone &&
		s.HasMilestone &&
		s.TriggerMilestoneValidation != validationPassed &&
		s.TriggerMilestoneOpenSiblings == 0
}

// canAdvance reports whether the Initiative may move to review: the triggering
// Issue is done, no Issues remain, every Milestone is validated, and the
// Initiative is still in an advanceable state. triggerPassed folds in a pass
// being applied this same cycle (a no-DoD Milestone) so the roll-up does not lag
// a decision by one wake.
func (s State) canAdvance(triggerPassed bool) bool {
	if !s.TriggerIssueDone || s.FeatureOpenSiblings != 0 {
		return false
	}
	if s.InitiativeStatus != "ready" && s.InitiativeStatus != "running" {
		return false
	}
	for _, m := range s.AllMilestones {
		passed := m.ValidationStatus == validationPassed
		if triggerPassed && m.ID == s.TriggerMilestoneID {
			passed = true
		}
		if !passed {
			return false
		}
	}
	return true
}
