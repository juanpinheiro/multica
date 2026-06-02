package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/initiative"
	"github.com/multica-ai/multica/server/internal/tripwire"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// pauseOnTripwire is the AFK safety net (ADR-0005). Consulted at the top of every
// Orchestrator reconcile: if the Initiative has exhausted its failure tolerance
// or hit a budget cap, it is moved to `blocked` and an inbox alert is raised so
// the human can intervene — and reconciliation stops, so no further Runs are
// dispatched. Returns true when the Initiative was paused.
//
// Only ready/running Initiatives can trip; terminal, blocked, and in_review ones
// are left alone (an already-paused Initiative is not re-alerted).
func (h *Handler) pauseOnTripwire(ctx context.Context, issue db.Issue, status string) bool {
	cur := initiative.Status(status)
	if cur != initiative.StatusReady && cur != initiative.StatusRunning {
		return false
	}

	feature, err := h.Queries.GetFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("tripwire: load feature failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		return false
	}
	state, ok := h.loadTripwireState(ctx, feature)
	if !ok {
		return false
	}
	pause, reason := tripwire.ShouldPause(state)
	if !pause {
		return false
	}
	if initiative.Transition(cur, initiative.StatusBlocked) != nil {
		return false
	}

	h.setFeatureStatus(ctx, feature.ID, initiative.StatusBlocked)
	h.notifyInitiativeTripwire(ctx, feature, reason)
	return true
}

// loadTripwireState assembles the pure tripwire snapshot from durable state. The
// failure count and Run count come from real queries; token and wall-clock usage
// are not yet recorded per Run, so they are sourced as 0 (their budgets stay
// inert until that tracking lands — see issue notes).
func (h *Handler) loadTripwireState(ctx context.Context, feature db.Feature) (tripwire.State, bool) {
	failures, err := h.Queries.MaxMilestoneValidationFailures(ctx, feature.ID)
	if err != nil {
		slog.Warn("tripwire: count milestone failures failed", "feature_id", uuidToString(feature.ID), "error", err)
		return tripwire.State{}, false
	}
	runs, err := h.Queries.CountRunsByFeature(ctx, feature.ID)
	if err != nil {
		slog.Warn("tripwire: count runs failed", "feature_id", uuidToString(feature.ID), "error", err)
		return tripwire.State{}, false
	}

	return tripwire.State{
		MaxMilestoneFailures: int(failures),
		FailureTolerance:     int(feature.FailureTolerance),
		TokenBudget:          feature.BudgetTokens,
		RunsUsed:             int(runs),
		RunBudget:            int(feature.BudgetRuns),
		TimeBudget:           feature.BudgetSeconds,
	}, true
}

// notifyInitiativeTripwire raises a best-effort inbox alert that the Initiative
// paused, so the human is pinged rather than silently stalled.
func (h *Handler) notifyInitiativeTripwire(ctx context.Context, feature db.Feature, reason tripwire.Reason) {
	members, err := h.Queries.ListMembers(ctx, feature.WorkspaceID)
	if err != nil || len(members) == 0 {
		slog.Warn("tripwire alert: load workspace member failed", "workspace_id", uuidToString(feature.WorkspaceID), "error", err)
		return
	}

	details, _ := json.Marshal(map[string]any{
		"feature_id": uuidToString(feature.ID),
		"reason":     string(reason),
		"mode":       feature.Mode,
	})

	item, err := h.Queries.CreateInboxItem(ctx, db.CreateInboxItemParams{
		WorkspaceID:   feature.WorkspaceID,
		RecipientType: "member",
		RecipientID:   members[0].ID,
		Type:          "initiative_tripwire",
		Severity:      "action_required",
		Title:         fmt.Sprintf("Initiative '%s' paused", feature.Title),
		Body:          pgtype.Text{String: tripwireReasonBody(reason), Valid: true},
		ActorType:     pgtype.Text{String: "system", Valid: true},
		Details:       details,
	})
	if err != nil {
		slog.Warn("tripwire alert: create inbox item failed", "feature_id", uuidToString(feature.ID), "error", err)
		return
	}

	h.publish(protocol.EventInboxNew, uuidToString(feature.WorkspaceID), "system", "",
		map[string]any{"item": inboxItemToEventMap(item)})
}

// tripwireReasonBody renders a human-readable explanation for the pause.
func tripwireReasonBody(reason tripwire.Reason) string {
	switch reason {
	case tripwire.ReasonFailureTolerance:
		return "A Milestone failed its Definition of Done too many times. The Initiative is paused for your review."
	case tripwire.ReasonTokenBudget:
		return "The Initiative reached its token budget and is paused for your review."
	case tripwire.ReasonRunBudget:
		return "The Initiative reached its Run budget and is paused for your review."
	case tripwire.ReasonTimeBudget:
		return "The Initiative reached its time budget and is paused for your review."
	default:
		return "The Initiative tripped a safety limit and is paused for your review."
	}
}
