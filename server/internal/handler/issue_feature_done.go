package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// notifyFeatureReadyForReview fires an inbox notification when the last
// issue under a shared-branch feature transitions to done.
//
// Guards:
//   - transition must be non-done → done (prev.Status != "done" and issue.Status == "done")
//   - issue must belong to a feature (feature_id set)
//   - feature must have a branch_slug (shared-branch model only)
//   - no siblings may remain outside done status
//
// Errors are logged at warn and swallowed — this is a best-effort side-effect
// on a successful status transition and must not roll back the caller.
func (h *Handler) notifyFeatureReadyForReview(ctx context.Context, prev, issue db.Issue) {
	if prev.Status == "done" || issue.Status != "done" {
		return
	}
	if !issue.FeatureID.Valid {
		return
	}

	feature, err := h.Queries.GetFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("feature ready: load feature failed",
			"error", err,
			"issue_id", uuidToString(issue.ID),
			"feature_id", uuidToString(issue.FeatureID))
		return
	}
	if !feature.BranchSlug.Valid || feature.BranchSlug.String == "" {
		return
	}

	remaining, err := h.Queries.CountNonDoneFeatureSiblings(ctx, db.CountNonDoneFeatureSiblingsParams{
		FeatureID: issue.FeatureID,
		ID:        issue.ID,
	})
	if err != nil {
		slog.Warn("feature ready: count siblings failed",
			"error", err,
			"feature_id", uuidToString(issue.FeatureID))
		return
	}
	if remaining > 0 {
		return
	}

	// Use the first workspace member as the recipient (singleton-user workspace).
	members, err := h.Queries.ListMembers(ctx, issue.WorkspaceID)
	if err != nil || len(members) == 0 {
		slog.Warn("feature ready: load workspace member failed",
			"error", err,
			"workspace_id", uuidToString(issue.WorkspaceID))
		return
	}
	member := members[0]

	total, err := h.Queries.CountIssuesByFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("feature ready: count total issues failed", "error", err)
		total = 0
	}

	body, prURL := h.buildFeatureReadyBody(ctx, feature, int(total))
	details, _ := json.Marshal(map[string]any{
		"feature_id":    uuidToString(feature.ID),
		"branch_slug":   feature.BranchSlug.String,
		"issues_done":   total,
		"pr_url":        prURL,
	})

	item, err := h.Queries.CreateInboxItem(ctx, db.CreateInboxItemParams{
		WorkspaceID:   issue.WorkspaceID,
		RecipientType: "member",
		RecipientID:   member.ID,
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
			"error", err,
			"feature_id", uuidToString(feature.ID))
		return
	}

	h.publish(protocol.EventInboxNew, uuidToString(issue.WorkspaceID), "system", "",
		map[string]any{"item": featureReadyInboxItemToMap(item)})
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

// featureReadyInboxItemToMap builds the wire-shape map for EventInboxNew,
// mirroring the shape used by the notification_listeners path.
func featureReadyInboxItemToMap(item db.InboxItem) map[string]any {
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
