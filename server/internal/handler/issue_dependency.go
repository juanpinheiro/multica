package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// CreateIssueDependencyRequest is the body for POST /api/issues/{id}/dependencies.
type CreateIssueDependencyRequest struct {
	DependsOnIssueID string `json:"depends_on_issue_id"`
	Type             string `json:"type"`
}

// IssueDependencyResponse is the JSON response for a created issue dependency.
type IssueDependencyResponse struct {
	ID               string `json:"id"`
	IssueID          string `json:"issue_id"`
	DependsOnIssueID string `json:"depends_on_issue_id"`
	Type             string `json:"type"`
}

// CreateIssueDependency handles POST /api/issues/{id}/dependencies.
// Accepted types are "blocks" and "related"; "blocked_by" is the redundant
// inverse and is hidden from callers. Duplicate links are idempotent: a
// second POST with the same (issue_id, depends_on_issue_id, type) returns
// the existing row with 200 instead of inserting again.
func (h *Handler) CreateIssueDependency(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, id)
	if !ok {
		return
	}

	var req CreateIssueDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DependsOnIssueID == "" {
		writeError(w, http.StatusBadRequest, "depends_on_issue_id is required")
		return
	}
	if req.Type != "blocks" && req.Type != "related" {
		writeError(w, http.StatusBadRequest, "type must be 'blocks' or 'related'")
		return
	}

	targetID, ok := parseUUIDOrBadRequest(w, req.DependsOnIssueID, "depends_on_issue_id")
	if !ok {
		return
	}

	if _, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
		ID:          targetID,
		WorkspaceID: issue.WorkspaceID,
	}); err != nil {
		writeError(w, http.StatusBadRequest, "depends_on_issue_id not found in this workspace")
		return
	}

	if h.DB == nil {
		writeError(w, http.StatusInternalServerError, "failed to create dependency")
		return
	}

	var existingID pgtype.UUID
	if err := h.DB.QueryRow(r.Context(), `
		SELECT id FROM issue_dependency
		WHERE issue_id = $1 AND depends_on_issue_id = $2 AND type = $3
		LIMIT 1
	`, issue.ID, targetID, req.Type).Scan(&existingID); err == nil {
		writeJSON(w, http.StatusOK, IssueDependencyResponse{
			ID:               uuidToString(existingID),
			IssueID:          uuidToString(issue.ID),
			DependsOnIssueID: uuidToString(targetID),
			Type:             req.Type,
		})
		return
	}

	var newID pgtype.UUID
	if err := h.DB.QueryRow(r.Context(), `
		INSERT INTO issue_dependency (issue_id, depends_on_issue_id, type)
		VALUES ($1, $2, $3)
		RETURNING id
	`, issue.ID, targetID, req.Type).Scan(&newID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create dependency")
		return
	}

	writeJSON(w, http.StatusCreated, IssueDependencyResponse{
		ID:               uuidToString(newID),
		IssueID:          uuidToString(issue.ID),
		DependsOnIssueID: uuidToString(targetID),
		Type:             req.Type,
	})
}
