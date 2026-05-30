package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// insertTestAttachment creates an attachment row linked to the given comment and returns its UUID.
func insertTestAttachment(t *testing.T, commentID, issueID string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO attachment (workspace_id, issue_id, comment_id, uploader_type, uploader_id, filename, url, content_type, size_bytes)
		VALUES ($1, $2, $3, 'member', $4, 'test.txt', 'https://cdn.example.com/test.txt', 'text/plain', 100)
		RETURNING id
	`, testWorkspaceID, issueID, commentID, testUserID).Scan(&id); err != nil {
		t.Fatalf("insertTestAttachment: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, id)
	})
	return id
}

// insertTestComment creates a comment row and returns its UUID.
func insertTestComment(t *testing.T, issueID string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO comment (workspace_id, issue_id, author_type, author_id, content, type)
		VALUES ($1, $2, 'member', $3, 'test comment', 'comment')
		RETURNING id
	`, testWorkspaceID, issueID, testUserID).Scan(&id); err != nil {
		t.Fatalf("insertTestComment: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM comment WHERE id = $1`, id)
	})
	return id
}

// insertTestIssue creates an issue row and returns its UUID.
func insertTestIssue(t *testing.T) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO issue (workspace_id, creator_type, creator_id, title)
		VALUES ($1, 'member', $2, 'attachment test issue')
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&id); err != nil {
		t.Fatalf("insertTestIssue: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
	})
	return id
}

func TestUpdateComment_RemoveAttachmentIDs_UnlinksAttachment(t *testing.T) {
	if testHandler == nil {
		t.Skip("no database")
	}

	issueID := insertTestIssue(t)
	commentID := insertTestComment(t, issueID)
	attachmentID := insertTestAttachment(t, commentID, issueID)

	// Verify the attachment is linked to the comment before the update.
	var linkedCommentID *string
	if err := testPool.QueryRow(context.Background(),
		`SELECT comment_id::text FROM attachment WHERE id = $1`, attachmentID,
	).Scan(&linkedCommentID); err != nil {
		t.Fatalf("query attachment before update: %v", err)
	}
	if linkedCommentID == nil || *linkedCommentID != commentID {
		t.Fatalf("expected attachment linked to comment %s before update, got %v", commentID, linkedCommentID)
	}

	body, _ := json.Marshal(map[string]any{
		"content":             "updated content",
		"remove_attachment_ids": []string{attachmentID},
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("commentId", commentID)
	req := httptest.NewRequest(http.MethodPut, "/api/comments/"+commentID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	testHandler.UpdateComment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The attachment's comment_id should now be NULL.
	var afterCommentID *string
	if err := testPool.QueryRow(context.Background(),
		`SELECT comment_id::text FROM attachment WHERE id = $1`, attachmentID,
	).Scan(&afterCommentID); err != nil {
		t.Fatalf("query attachment after update: %v", err)
	}
	if afterCommentID != nil {
		t.Fatalf("expected attachment unlinked (comment_id NULL) after update, got %s", *afterCommentID)
	}
}

func TestUpdateComment_RemoveAttachmentIDs_DoesNotUnlinkOtherComments(t *testing.T) {
	if testHandler == nil {
		t.Skip("no database")
	}

	issueID := insertTestIssue(t)
	commentID := insertTestComment(t, issueID)
	otherCommentID := insertTestComment(t, issueID)
	attachmentID := insertTestAttachment(t, otherCommentID, issueID)

	// Try to remove an attachment that belongs to a different comment — should be a no-op.
	body, _ := json.Marshal(map[string]any{
		"content":             "updated content",
		"remove_attachment_ids": []string{attachmentID},
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("commentId", commentID)
	req := httptest.NewRequest(http.MethodPut, "/api/comments/"+commentID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	testHandler.UpdateComment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The other comment's attachment should remain linked.
	var linkedCommentID *string
	if err := testPool.QueryRow(context.Background(),
		`SELECT comment_id::text FROM attachment WHERE id = $1`, attachmentID,
	).Scan(&linkedCommentID); err != nil {
		t.Fatalf("query attachment after update: %v", err)
	}
	if linkedCommentID == nil || *linkedCommentID != otherCommentID {
		t.Fatalf("expected attachment still linked to %s, got %v", otherCommentID, linkedCommentID)
	}
}
