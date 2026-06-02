package orchestrator

import "testing"

// boundary helper: a trigger Issue that just reached done as the last open
// Issue of a pending Milestone.
func atBoundary(s State) State {
	s.TriggerIssueDone = true
	s.HasMilestone = true
	s.TriggerMilestoneValidation = "pending"
	s.TriggerMilestoneOpenSiblings = 0
	return s
}

func TestDecide_NoMilestone_NoBoundaryAction(t *testing.T) {
	s := State{
		InitiativeStatus:    "running",
		TriggerIssueDone:    true,
		HasMilestone:        false,
		FeatureOpenSiblings: 1, // siblings remain
	}
	p := Decide(s)
	if p.PassMilestone || p.DispatchValidator {
		t.Fatalf("no milestone: want no boundary action, got %+v", p)
	}
	if p.AdvanceTo != "" {
		t.Fatalf("siblings remain: want no advance, got %q", p.AdvanceTo)
	}
}

func TestDecide_BoundaryNoDoD_PassesMilestone(t *testing.T) {
	s := atBoundary(State{
		InitiativeStatus:              "running",
		TriggerMilestoneID:            "m1",
		TriggerMilestoneHasAssertions: false,
		AllMilestones:                 []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	})
	p := Decide(s)
	if !p.PassMilestone {
		t.Fatal("no-DoD boundary: want PassMilestone")
	}
	if p.DispatchValidator {
		t.Fatal("no-DoD boundary: must not dispatch a validator")
	}
}

func TestDecide_BoundaryWithDoD_DispatchesValidator(t *testing.T) {
	s := atBoundary(State{
		InitiativeStatus:                 "running",
		TriggerMilestoneID:               "m1",
		TriggerMilestoneHasAssertions:    true,
		TriggerMilestoneActiveValidators: 0,
		AllMilestones:                    []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	})
	p := Decide(s)
	if !p.DispatchValidator {
		t.Fatal("DoD boundary, no active validator: want DispatchValidator")
	}
	if p.PassMilestone {
		t.Fatal("DoD boundary: must not pass the milestone without validation")
	}
	if p.AdvanceTo != "" {
		t.Fatal("milestone not yet validated: must not advance")
	}
}

func TestDecide_BoundaryWithDoD_ActiveValidator_NoDuplicate(t *testing.T) {
	s := atBoundary(State{
		InitiativeStatus:                 "running",
		TriggerMilestoneID:               "m1",
		TriggerMilestoneHasAssertions:    true,
		TriggerMilestoneActiveValidators: 1,
		AllMilestones:                    []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	})
	if Decide(s).DispatchValidator {
		t.Fatal("a validator is already in flight: must not dispatch another")
	}
}

func TestDecide_MilestoneAlreadyPassed_NoBoundaryAction(t *testing.T) {
	s := State{
		InitiativeStatus:             "running",
		TriggerIssueDone:             true,
		HasMilestone:                 true,
		TriggerMilestoneID:           "m1",
		TriggerMilestoneValidation:   "passed",
		TriggerMilestoneOpenSiblings: 0,
		FeatureOpenSiblings:          1, // another milestone's issue still open
		AllMilestones:                []Milestone{{ID: "m1", ValidationStatus: "passed"}},
	}
	p := Decide(s)
	if p.PassMilestone || p.DispatchValidator {
		t.Fatalf("already passed: want no boundary action, got %+v", p)
	}
}

func TestDecide_TriggerIssueNotDone_NoBoundary(t *testing.T) {
	s := State{
		InitiativeStatus:             "running",
		TriggerIssueDone:             false, // worker task completed but issue not done yet
		HasMilestone:                 true,
		TriggerMilestoneID:           "m1",
		TriggerMilestoneValidation:   "pending",
		TriggerMilestoneOpenSiblings: 0,
		AllMilestones:                []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	}
	p := Decide(s)
	if p.PassMilestone || p.DispatchValidator || p.AdvanceTo != "" {
		t.Fatalf("trigger issue not done: want no action, got %+v", p)
	}
}

func TestDecide_OpenSiblings_WaitsForThem(t *testing.T) {
	s := atBoundary(State{
		InitiativeStatus:              "running",
		TriggerMilestoneID:            "m1",
		TriggerMilestoneHasAssertions: true,
		AllMilestones:                 []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	})
	s.TriggerMilestoneOpenSiblings = 1
	if Decide(s).DispatchValidator {
		t.Fatal("a sibling is still open: must not dispatch a validator")
	}
}

func TestDecide_AllMilestonesPassed_AdvancesToInReview(t *testing.T) {
	// Last milestone already validated (passed in DB), trigger issue done,
	// no feature siblings left → the Initiative is ready for review.
	s := State{
		InitiativeStatus:             "running",
		TriggerIssueDone:             true,
		HasMilestone:                 true,
		TriggerMilestoneID:           "m2",
		TriggerMilestoneValidation:   "passed",
		TriggerMilestoneOpenSiblings: 0,
		FeatureOpenSiblings:          0,
		AllMilestones: []Milestone{
			{ID: "m1", ValidationStatus: "passed"},
			{ID: "m2", ValidationStatus: "passed"},
		},
	}
	if got := Decide(s).AdvanceTo; got != "in_review" {
		t.Fatalf("all milestones passed + no siblings: want advance in_review, got %q", got)
	}
}

func TestDecide_NoDoDLastMilestone_PassesAndAdvances(t *testing.T) {
	// A single no-DoD milestone reaching its boundary should both be passed and
	// (treating that pass as effective) advance the Initiative in one decision.
	s := atBoundary(State{
		InitiativeStatus:              "running",
		TriggerMilestoneID:            "m1",
		TriggerMilestoneHasAssertions: false,
		FeatureOpenSiblings:           0,
		AllMilestones:                 []Milestone{{ID: "m1", ValidationStatus: "pending"}},
	})
	p := Decide(s)
	if !p.PassMilestone {
		t.Fatal("want PassMilestone")
	}
	if p.AdvanceTo != "in_review" {
		t.Fatalf("no-DoD last milestone: want advance in_review, got %q", p.AdvanceTo)
	}
}

func TestDecide_NoMilestones_AdvancesWhenFeatureDone(t *testing.T) {
	// Tracer flow: an Initiative with no Milestones advances once all its Issues
	// are done. Vacuously "all milestones passed".
	s := State{
		InitiativeStatus:    "running",
		TriggerIssueDone:    true,
		HasMilestone:        false,
		FeatureOpenSiblings: 0,
		AllMilestones:       nil,
	}
	if got := Decide(s).AdvanceTo; got != "in_review" {
		t.Fatalf("no milestones, feature done: want in_review, got %q", got)
	}
}

func TestDecide_PendingEarlierMilestone_NoAdvance(t *testing.T) {
	s := State{
		InitiativeStatus:           "running",
		TriggerIssueDone:           true,
		HasMilestone:               true,
		TriggerMilestoneID:         "m2",
		TriggerMilestoneValidation: "passed",
		FeatureOpenSiblings:        0,
		AllMilestones: []Milestone{
			{ID: "m1", ValidationStatus: "pending"},
			{ID: "m2", ValidationStatus: "passed"},
		},
	}
	if got := Decide(s).AdvanceTo; got != "" {
		t.Fatalf("an earlier milestone is unvalidated: want no advance, got %q", got)
	}
}

func TestDecide_TerminalInitiative_NoAdvance(t *testing.T) {
	for _, status := range []string{"in_review", "done", "cancelled", "blocked", "draft"} {
		s := State{
			InitiativeStatus:    status,
			TriggerIssueDone:    true,
			FeatureOpenSiblings: 0,
		}
		if got := Decide(s).AdvanceTo; got != "" {
			t.Fatalf("status %q: want no advance, got %q", status, got)
		}
	}
}
