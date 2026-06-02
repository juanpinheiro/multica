// Package tripwire holds the pure pause-decision for an Initiative (ADR-0005):
// given its accumulated failure and budget state, ShouldPause answers "should
// this Initiative pause and ping the human, and if so, why?". It is the AFK
// safety net — what stops an imperfect Definition of Done from burning days.
//
// Like gate, dod, and initiative, this module is pure (no I/O): the orchestrator
// scaffold feeds it counts and, when it trips, moves the Initiative to blocked
// and raises an inbox alert. Mode (HITL/AFK) is the planning-time autonomy
// choice and lives on the entity; the runtime safety is this tripwire.
package tripwire

// Reason names why an Initiative should pause, or ReasonNone when it should not.
type Reason string

const (
	ReasonNone             Reason = ""
	ReasonFailureTolerance Reason = "failure_tolerance"
	ReasonTokenBudget      Reason = "token_budget"
	ReasonRunBudget        Reason = "run_budget"
	ReasonTimeBudget       Reason = "time_budget"
)

// State is the accumulated cost/failure snapshot of one Initiative. Every budget
// is a hard ceiling; a zero (or negative) cap means "no cap" — that dimension
// never trips, so an Initiative with nothing configured is unguarded except by
// any failure tolerance it carries.
type State struct {
	// MaxMilestoneFailures is the worst single Milestone's validation-failure
	// count: how many times any one Milestone has failed its Definition of Done.
	MaxMilestoneFailures int
	// FailureTolerance is the number of repeated same-Milestone failures allowed
	// before pausing. A value <= 0 disables the failure tripwire.
	FailureTolerance int

	// TokensUsed / TokenBudget cap total tokens spent across the Initiative.
	TokensUsed  int64
	TokenBudget int64

	// RunsUsed / RunBudget cap the number of Runs the Initiative may dispatch.
	RunsUsed  int
	RunBudget int

	// ElapsedSeconds / TimeBudget cap the wall-clock the Initiative may run.
	ElapsedSeconds int64
	TimeBudget     int64
}

// ShouldPause reports whether the Initiative has tripped a safety limit and, if
// so, the first reason in a fixed precedence: failure tolerance, then token, run,
// and time budgets.
func ShouldPause(s State) (bool, Reason) {
	r := s.reason()
	return r != ReasonNone, r
}

func (s State) reason() Reason {
	switch {
	case s.FailureTolerance > 0 && s.MaxMilestoneFailures >= s.FailureTolerance:
		return ReasonFailureTolerance
	case s.TokenBudget > 0 && s.TokensUsed >= s.TokenBudget:
		return ReasonTokenBudget
	case s.RunBudget > 0 && s.RunsUsed >= s.RunBudget:
		return ReasonRunBudget
	case s.TimeBudget > 0 && s.ElapsedSeconds >= s.TimeBudget:
		return ReasonTimeBudget
	default:
		return ReasonNone
	}
}
