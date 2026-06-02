package gate

import "testing"

func ms(id string, pos int, status string) Milestone {
	return Milestone{ID: id, Position: pos, ValidationStatus: status}
}

func TestMilestoneGateOpen(t *testing.T) {
	m1pending := ms("m1", 0, "pending")
	m1passed := ms("m1", 0, ValidationPassed)
	m1failed := ms("m1", 0, "failed")
	m2 := ms("m2", 1, "pending")
	m3 := ms("m3", 2, "pending")

	cases := []struct {
		name   string
		all    []Milestone
		target string
		want   bool
	}{
		{"no milestone is always open", []Milestone{m1pending}, "", true},
		{"unknown target is open", []Milestone{m1pending}, "ghost", true},
		{"first milestone has no earlier sibling", []Milestone{m1pending, m2}, "m1", true},
		{"second blocked while first pending", []Milestone{m1pending, m2}, "m2", false},
		{"second open once first passed", []Milestone{m1passed, m2}, "m2", true},
		{"second blocked when first failed", []Milestone{m1failed, m2}, "m2", false},
		{"third blocked when an earlier is not passed", []Milestone{m1passed, m2, m3}, "m3", false},
		{"third open when all earlier passed", []Milestone{m1passed, ms("m2", 1, ValidationPassed), m3}, "m3", true},
		{"same-position sibling does not gate", []Milestone{m1pending, ms("m2", 0, "pending")}, "m2", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MilestoneGateOpen(c.all, c.target); got != c.want {
				t.Errorf("MilestoneGateOpen(%q) = %v, want %v", c.target, got, c.want)
			}
		})
	}
}

func TestClaimable(t *testing.T) {
	open := World{InitiativeActive: true, DependenciesMet: true, AgentFree: true, BranchFree: true}
	gated := []Milestone{ms("m1", 0, "pending"), ms("m2", 1, "pending")}

	cases := []struct {
		name  string
		world World
		want  Reason
	}{
		{"all gates open", open, ReasonClaimable},
		{"agent busy wins precedence", World{}, ReasonAgentBusy},
		{
			"initiative inactive",
			World{AgentFree: true, DependenciesMet: true, BranchFree: true},
			ReasonInitiativeInactive,
		},
		{
			"milestone gated",
			World{InitiativeActive: true, DependenciesMet: true, AgentFree: true, BranchFree: true, Milestones: gated, MilestoneID: "m2"},
			ReasonMilestoneGated,
		},
		{
			"dependency unmet",
			World{InitiativeActive: true, AgentFree: true, BranchFree: true},
			ReasonDependencyUnmet,
		},
		{
			"branch held",
			World{InitiativeActive: true, DependenciesMet: true, AgentFree: true},
			ReasonBranchHeld,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Claimable(c.world); got != c.want {
				t.Errorf("Claimable() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestReasonOK(t *testing.T) {
	if !ReasonClaimable.OK() {
		t.Error("ReasonClaimable.OK() = false, want true")
	}
	if ReasonMilestoneGated.OK() {
		t.Error("ReasonMilestoneGated.OK() = true, want false")
	}
}
