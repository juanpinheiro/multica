package service

import (
	"context"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// AgentReadiness reports whether an agent can accept new work right now.
// "Ready" means archived_at IS NULL, runtime_id IS NOT NULL, and the bound
// runtime's status is 'online'. When not ready, reason describes which gate
// failed in language suitable for autopilot_run.failure_reason.
//
// err is non-nil only on DB lookup failure for the runtime row. Callers that
// treat a transient DB error as "do not skip" (the autopilot admission gate)
// should swallow it.
//
// This is the single source of truth for agent readiness shared by:
//   - service.shouldSkipDispatch (autopilot admission gate)
//   - service.dispatchRunOnly    (autopilot runtime check)
//   - handler.isAgentAssigneeReady (issue-assign path)
//
// Touch this function, all paths move together.
func AgentReadiness(ctx context.Context, q *db.Queries, agent db.Agent) (ready bool, reason string, err error) {
	if agent.ArchivedAt.Valid {
		return false, "agent is archived", nil
	}
	if !agent.RuntimeID.Valid {
		return false, "agent has no runtime bound", nil
	}
	rt, err := q.GetAgentRuntime(ctx, agent.RuntimeID)
	if err != nil {
		return false, "", err
	}
	if rt.Status != "online" {
		return false, "agent runtime is " + rt.Status, nil
	}
	return true, "", nil
}
