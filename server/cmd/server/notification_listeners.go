package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

var emptyDetails = []byte("{}")

// terminalStatusForTaskFailedDismiss is the set of issue statuses after which
// stale task_failed inbox rows are archived so the inbox reflects current work.
var terminalStatusForTaskFailedDismiss = map[string]bool{
	"in_review": true,
	"done":      true,
	"cancelled": true,
}

// archiveStaleTaskFailedInbox archives all task_failed inbox rows for the
// given issue and notifies each affected recipient via inbox:batch-archived.
func archiveStaleTaskFailedInbox(
	ctx context.Context,
	queries *db.Queries,
	bus *events.Bus,
	workspaceID string,
	issueID string,
) {
	rows, err := queries.ArchiveInboxByIssueAndType(ctx, db.ArchiveInboxByIssueAndTypeParams{
		WorkspaceID: parseUUID(workspaceID),
		IssueID:     parseUUID(issueID),
		Type:        "task_failed",
	})
	if err != nil {
		slog.Error("auto-archive task_failed inbox: query failed",
			"workspace_id", workspaceID, "issue_id", issueID, "error", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	counts := map[string]int{}
	for _, row := range rows {
		if row.RecipientType != "member" {
			continue
		}
		counts[util.UUIDToString(row.RecipientID)]++
	}

	for recipientID, count := range counts {
		bus.Publish(events.Event{
			Type:        protocol.EventInboxBatchArchived,
			WorkspaceID: workspaceID,
			Payload: map[string]any{
				"recipient_id": recipientID,
				"count":        int64(count),
				"issue_id":     issueID,
				"reason":       "issue_status_terminal",
			},
		})
	}

	slog.Info("auto-archive task_failed inbox: archived stale rows",
		"workspace_id", workspaceID, "issue_id", issueID,
		"row_count", len(rows), "recipient_count", len(counts))
}

// registerNotificationListeners wires up event bus listeners that maintain
// the inbox. In the single-user agent-driven model, the inbox is used for
// agent activity signals (task_failed cleanup on status change).
func registerNotificationListeners(bus *events.Bus, queries *db.Queries) {
	ctx := context.Background()

	// issue:updated — archive stale task_failed rows when the issue reaches a
	// terminal status so the inbox stays a fresh-signal surface.
	bus.Subscribe(protocol.EventIssueUpdated, func(e events.Event) {
		payload, ok := e.Payload.(map[string]any)
		if !ok {
			return
		}
		issue, ok := payload["issue"].(handler.IssueResponse)
		if !ok {
			return
		}
		if statusChanged, _ := payload["status_changed"].(bool); statusChanged {
			if terminalStatusForTaskFailedDismiss[issue.Status] {
				archiveStaleTaskFailedInbox(ctx, queries, bus, e.WorkspaceID, issue.ID)
			}
		}
	})
}

// inboxItemToResponse converts a db.InboxItem into a map suitable for
// JSON-serializable event payloads.
func inboxItemToResponse(item db.InboxItem) map[string]any {
	return map[string]any{
		"id":             util.UUIDToString(item.ID),
		"workspace_id":   util.UUIDToString(item.WorkspaceID),
		"recipient_type": item.RecipientType,
		"recipient_id":   util.UUIDToString(item.RecipientID),
		"type":           item.Type,
		"severity":       item.Severity,
		"issue_id":       util.UUIDToPtr(item.IssueID),
		"title":          item.Title,
		"body":           util.TextToPtr(item.Body),
		"read":           item.Read,
		"archived":       item.Archived,
		"created_at":     util.TimestampToString(item.CreatedAt),
		"actor_type":     util.TextToPtr(item.ActorType),
		"actor_id":       util.UUIDToPtr(item.ActorID),
		"details":        json.RawMessage(item.Details),
	}
}
