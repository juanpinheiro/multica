package main

import (
	"context"
	"testing"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// Shared integration test helpers for cmd/server package tests.

func createTestIssue(t *testing.T, workspaceID, creatorID string) string {
	t.Helper()
	ctx := context.Background()
	var issueID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_type, creator_id, position, number)
		VALUES ($1, 'test issue', 'todo', 'medium', 'member', $2, 0,
		        (SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1))
		RETURNING id
	`, workspaceID, creatorID).Scan(&issueID)
	if err != nil {
		t.Fatalf("createTestIssue: %v", err)
	}
	return issueID
}

func createTestUser(t *testing.T, email string) string {
	t.Helper()
	ctx := context.Background()
	var userID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO "user" (name, email)
		VALUES ('Test User', $1)
		RETURNING id
	`, email).Scan(&userID)
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	return userID
}

func cleanupTestIssue(t *testing.T, issueID string) {
	t.Helper()
	testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
}

func cleanupTestUser(t *testing.T, email string) {
	t.Helper()
	testPool.Exec(context.Background(), `DELETE FROM "user" WHERE email = $1`, email)
}

// inboxItemsForRecipient returns all non-archived inbox items for a given recipient.
func inboxItemsForRecipient(t *testing.T, queries *db.Queries, recipientID string) []db.ListInboxItemsRow {
	t.Helper()
	items, err := queries.ListInboxItems(context.Background(), db.ListInboxItemsParams{
		WorkspaceID:   util.MustParseUUID(testWorkspaceID),
		RecipientType: "member",
		RecipientID:   util.MustParseUUID(recipientID),
	})
	if err != nil {
		t.Fatalf("ListInboxItems: %v", err)
	}
	return items
}

// seedTaskFailedInbox directly inserts a task_failed inbox row for a member recipient.
func seedTaskFailedInbox(t *testing.T, issueID, recipientID string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO inbox_item (workspace_id, recipient_type, recipient_id, type, severity, issue_id, title, details)
		VALUES ($1, 'member', $2, 'task_failed', 'action_required', $3, 'task failed', '{}')
	`, testWorkspaceID, recipientID, issueID)
	if err != nil {
		t.Fatalf("seedTaskFailedInbox: %v", err)
	}
}

// cleanupInboxForIssue deletes all inbox items related to a given issue.
func cleanupInboxForIssue(t *testing.T, issueID string) {
	t.Helper()
	testPool.Exec(context.Background(), `DELETE FROM inbox_item WHERE issue_id = $1`, issueID)
}

// countInboxByTypeForRecipient returns active and archived counts for a given type and recipient.
func countInboxByTypeForRecipient(t *testing.T, recipientID, notifType string) (active, archived int) {
	t.Helper()
	rows, err := testPool.Query(context.Background(), `
		SELECT archived FROM inbox_item
		WHERE workspace_id = $1 AND recipient_type = 'member' AND recipient_id = $2 AND type = $3
	`, testWorkspaceID, recipientID, notifType)
	if err != nil {
		t.Fatalf("countInboxByTypeForRecipient: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var isArchived bool
		if err := rows.Scan(&isArchived); err != nil {
			t.Fatalf("countInboxByTypeForRecipient scan: %v", err)
		}
		if isArchived {
			archived++
		} else {
			active++
		}
	}
	return active, archived
}

// publishStatusChange fires the issue:updated event for a status-only transition.
func publishStatusChange(bus *events.Bus, issueID, newStatus, prevStatus string) {
	bus.Publish(events.Event{
		Type:        protocol.EventIssueUpdated,
		WorkspaceID: testWorkspaceID,
		ActorType:   "member",
		ActorID:     testUserID,
		Payload: map[string]any{
			"issue": handler.IssueResponse{
				ID:          issueID,
				WorkspaceID: testWorkspaceID,
				Title:       "task_failed dismiss test",
				Status:      newStatus,
				Priority:    "medium",
				CreatorType: "member",
				CreatorID:   testUserID,
			},
			"assignee_changed": false,
			"status_changed":   true,
			"prev_status":      prevStatus,
		},
	})
}

// TestNotification_StatusChange_ArchivesStaleTaskFailed verifies that a
// terminal status transition archives existing task_failed inbox rows and
// fires inbox:batch-archived per recipient.
func TestNotification_StatusChange_ArchivesStaleTaskFailed(t *testing.T) {
	queries := db.New(testPool)
	bus := events.New()
	registerNotificationListeners(bus, queries)

	secondEmail := "notif-archive-task-failed-sub@multica.ai"
	secondID := createTestUser(t, secondEmail)
	t.Cleanup(func() { cleanupTestUser(t, secondEmail) })

	issueID := createTestIssue(t, testWorkspaceID, testUserID)
	t.Cleanup(func() {
		cleanupInboxForIssue(t, issueID)
		cleanupTestIssue(t, issueID)
	})

	// Seed two task_failed rows for each recipient.
	for i := 0; i < 2; i++ {
		seedTaskFailedInbox(t, issueID, testUserID)
		seedTaskFailedInbox(t, issueID, secondID)
	}

	// Seed a sibling non-task-failed notification to verify narrow archive scope.
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO inbox_item (workspace_id, recipient_type, recipient_id, type, severity, issue_id, title, details)
		VALUES ($1, 'member', $2, 'new_comment', 'info', $3, 'sibling notification', '{}')
	`, testWorkspaceID, testUserID, issueID)
	if err != nil {
		t.Fatalf("seed sibling notification: %v", err)
	}

	if active, _ := countInboxByTypeForRecipient(t, testUserID, "task_failed"); active != 2 {
		t.Fatalf("precondition: expected 2 active task_failed rows for creator, got %d", active)
	}
	if active, _ := countInboxByTypeForRecipient(t, secondID, "task_failed"); active != 2 {
		t.Fatalf("precondition: expected 2 active task_failed rows for second user, got %d", active)
	}

	var batchArchived []events.Event
	bus.Subscribe(protocol.EventInboxBatchArchived, func(e events.Event) {
		batchArchived = append(batchArchived, e)
	})

	publishStatusChange(bus, issueID, "in_review", "in_progress")

	for _, recipientID := range []string{testUserID, secondID} {
		active, archived := countInboxByTypeForRecipient(t, recipientID, "task_failed")
		if active != 0 {
			t.Fatalf("recipient %s: expected 0 active task_failed rows, got %d", recipientID, active)
		}
		if archived != 2 {
			t.Fatalf("recipient %s: expected 2 archived task_failed rows, got %d", recipientID, archived)
		}
	}

	// Sibling notification untouched.
	if active, _ := countInboxByTypeForRecipient(t, testUserID, "new_comment"); active != 1 {
		t.Fatalf("expected sibling new_comment to remain active, got %d", active)
	}

	if len(batchArchived) != 2 {
		t.Fatalf("expected 2 inbox:batch-archived events, got %d", len(batchArchived))
	}
	seenRecipients := map[string]bool{}
	for _, e := range batchArchived {
		payload, ok := e.Payload.(map[string]any)
		if !ok {
			t.Fatalf("unexpected payload type %T", e.Payload)
		}
		recipientID, _ := payload["recipient_id"].(string)
		if recipientID == "" {
			t.Fatalf("missing recipient_id in batch-archived payload")
		}
		if payload["issue_id"] != issueID {
			t.Fatalf("expected issue_id %q, got %v", issueID, payload["issue_id"])
		}
		if payload["reason"] != "issue_status_terminal" {
			t.Fatalf("expected reason 'issue_status_terminal', got %v", payload["reason"])
		}
		if count, _ := payload["count"].(int64); count != 2 {
			t.Fatalf("expected count=2 for recipient %s, got %v", recipientID, payload["count"])
		}
		seenRecipients[recipientID] = true
	}
	if !seenRecipients[testUserID] || !seenRecipients[secondID] {
		t.Fatalf("expected batch-archived events for both recipients, got %v", seenRecipients)
	}
}

// TestNotification_StatusChange_NonTerminalKeepsTaskFailed verifies that a
// transition to a non-terminal status does NOT archive task_failed rows.
func TestNotification_StatusChange_NonTerminalKeepsTaskFailed(t *testing.T) {
	queries := db.New(testPool)
	bus := events.New()
	registerNotificationListeners(bus, queries)

	issueID := createTestIssue(t, testWorkspaceID, testUserID)
	t.Cleanup(func() {
		cleanupInboxForIssue(t, issueID)
		cleanupTestIssue(t, issueID)
	})

	seedTaskFailedInbox(t, issueID, testUserID)

	if active, _ := countInboxByTypeForRecipient(t, testUserID, "task_failed"); active != 1 {
		t.Fatalf("precondition: expected 1 active task_failed row, got %d", active)
	}

	publishStatusChange(bus, issueID, "in_progress", "todo")

	active, archived := countInboxByTypeForRecipient(t, testUserID, "task_failed")
	if active != 1 || archived != 0 {
		t.Fatalf("expected task_failed row to remain active after non-terminal transition, got active=%d archived=%d", active, archived)
	}
}
