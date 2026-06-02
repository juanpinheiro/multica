package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/initiative"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// notifyInitiativeReadyForReview fires an inbox notification when the Orchestrator
// advances an Initiative to in_review — all Milestones have passed their DoD.
// Prompts the human to flip the consolidated PR from draft to ready-for-review.
// Best-effort: errors are logged and swallowed.
func (h *Handler) notifyInitiativeReadyForReview(ctx context.Context, featureID pgtype.UUID) {
	feature, err := h.Queries.GetFeature(ctx, featureID)
	if err != nil {
		slog.Warn("feature ready: load feature failed", "feature_id", uuidToString(featureID), "error", err)
		return
	}
	if !feature.BranchSlug.Valid || feature.BranchSlug.String == "" {
		return
	}

	members, err := h.Queries.ListMembers(ctx, feature.WorkspaceID)
	if err != nil || len(members) == 0 {
		slog.Warn("feature ready: load workspace member failed",
			"error", err, "workspace_id", uuidToString(feature.WorkspaceID))
		return
	}

	total, err := h.Queries.CountIssuesByFeature(ctx, featureID)
	if err != nil {
		slog.Warn("feature ready: count total issues failed", "error", err)
		total = 0
	}

	body, prURL := h.buildFeatureReadyBody(ctx, feature, int(total))
	details, _ := json.Marshal(map[string]any{
		"feature_id":  uuidToString(feature.ID),
		"branch_slug": feature.BranchSlug.String,
		"issues_done": total,
		"pr_url":      prURL,
	})

	item, err := h.Queries.CreateInboxItem(ctx, db.CreateInboxItemParams{
		WorkspaceID:   feature.WorkspaceID,
		RecipientType: "member",
		RecipientID:   members[0].ID,
		Type:          "feature_ready_for_review",
		Severity:      "action_required",
		IssueID:       pgtype.UUID{},
		Title:         fmt.Sprintf("Feature '%s' ready for review", feature.Title),
		Body:          pgtype.Text{String: body, Valid: true},
		ActorType:     pgtype.Text{String: "system", Valid: true},
		ActorID:       pgtype.UUID{Valid: false},
		Details:       details,
	})
	if err != nil {
		slog.Warn("feature ready: create inbox item failed",
			"error", err, "feature_id", uuidToString(featureID))
		return
	}

	h.publish(protocol.EventInboxNew, uuidToString(feature.WorkspaceID), "system", "",
		map[string]any{"item": inboxItemToEventMap(item)})
}

// setFeatureStatus performs a validated Initiative status write. Best-effort:
// errors are logged, not returned, since it runs as a side-effect of a committed
// status transition.
func (h *Handler) setFeatureStatus(ctx context.Context, featureID pgtype.UUID, status initiative.Status) {
	if _, err := h.Queries.SetFeatureStatus(ctx, db.SetFeatureStatusParams{
		ID:     featureID,
		Status: string(status),
	}); err != nil {
		slog.Warn("set feature status failed",
			"error", err, "feature_id", uuidToString(featureID), "status", status)
	}
}

// buildFeatureReadyBody constructs the notification body. Returns (body, prURL)
// where prURL is empty when no open PR is linked to any issue in the feature.
func (h *Handler) buildFeatureReadyBody(ctx context.Context, feature db.Feature, issueCount int) (string, string) {
	word := "issues"
	if issueCount == 1 {
		word = "issue"
	}

	pr, err := h.Queries.GetFeatureOpenPR(ctx, feature.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Warn("feature ready: look up open PR failed", "error", err, "feature_id", uuidToString(feature.ID))
	}

	if pr.HtmlUrl != "" {
		return fmt.Sprintf("All %d %s completed. Consolidated PR: [#%d](%s).",
			issueCount, word, pr.PrNumber, pr.HtmlUrl), pr.HtmlUrl
	}
	return fmt.Sprintf("All %d %s completed. No PR linked yet.", issueCount, word), ""
}

// inboxItemToEventMap builds the wire-shape map for EventInboxNew, mirroring the
// shape used by the notification_listeners path. Shared by the feature-ready and
// tripwire-pause notification paths.
func inboxItemToEventMap(item db.InboxItem) map[string]any {
	return map[string]any{
		"id":             uuidToString(item.ID),
		"workspace_id":   uuidToString(item.WorkspaceID),
		"recipient_type": item.RecipientType,
		"recipient_id":   uuidToString(item.RecipientID),
		"type":           item.Type,
		"severity":       item.Severity,
		"issue_id":       uuidToPtr(item.IssueID),
		"title":          item.Title,
		"body":           textToPtr(item.Body),
		"read":           item.Read,
		"archived":       item.Archived,
		"created_at":     timestampToString(item.CreatedAt),
		"actor_type":     textToPtr(item.ActorType),
		"actor_id":       uuidToPtr(item.ActorID),
		"details":        json.RawMessage(item.Details),
		"issue_status":   "",
	}
}
