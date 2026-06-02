package tripwire

import "testing"

func TestShouldPause(t *testing.T) {
	cases := []struct {
		name       string
		state      State
		wantPause  bool
		wantReason Reason
	}{
		{
			"nothing configured never pauses",
			State{},
			false, ReasonNone,
		},
		{
			"repeated same-milestone failure trips at tolerance",
			State{MaxMilestoneFailures: 3, FailureTolerance: 3},
			true, ReasonFailureTolerance,
		},
		{
			"failures below tolerance do not trip",
			State{MaxMilestoneFailures: 2, FailureTolerance: 3},
			false, ReasonNone,
		},
		{
			"zero tolerance disables the failure tripwire",
			State{MaxMilestoneFailures: 9, FailureTolerance: 0},
			false, ReasonNone,
		},
		{
			"token budget trips when spent reaches cap",
			State{TokensUsed: 1000, TokenBudget: 1000},
			true, ReasonTokenBudget,
		},
		{
			"token usage under cap does not trip",
			State{TokensUsed: 999, TokenBudget: 1000},
			false, ReasonNone,
		},
		{
			"zero token budget means unlimited",
			State{TokensUsed: 1_000_000, TokenBudget: 0},
			false, ReasonNone,
		},
		{
			"run budget trips when run count reaches cap",
			State{RunsUsed: 5, RunBudget: 5},
			true, ReasonRunBudget,
		},
		{
			"zero run budget means unlimited",
			State{RunsUsed: 100, RunBudget: 0},
			false, ReasonNone,
		},
		{
			"time budget trips when elapsed reaches cap",
			State{ElapsedSeconds: 3600, TimeBudget: 3600},
			true, ReasonTimeBudget,
		},
		{
			"zero time budget means unlimited",
			State{ElapsedSeconds: 999_999, TimeBudget: 0},
			false, ReasonNone,
		},
		{
			"failure tolerance wins precedence over budgets",
			State{
				MaxMilestoneFailures: 3, FailureTolerance: 3,
				RunsUsed: 10, RunBudget: 5,
			},
			true, ReasonFailureTolerance,
		},
		{
			"token budget wins precedence over run and time",
			State{
				TokensUsed: 100, TokenBudget: 100,
				RunsUsed: 10, RunBudget: 5,
				ElapsedSeconds: 10, TimeBudget: 5,
			},
			true, ReasonTokenBudget,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pause, reason := ShouldPause(c.state)
			if pause != c.wantPause || reason != c.wantReason {
				t.Errorf("ShouldPause() = (%v, %q), want (%v, %q)", pause, reason, c.wantPause, c.wantReason)
			}
		})
	}
}
