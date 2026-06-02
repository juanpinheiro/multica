package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// MilestoneResponse is the JSON response for a milestone.
type MilestoneResponse struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspace_id"`
	FeatureID        string `json:"feature_id"`
	Title            string `json:"title"`
	Position         int32  `json:"position"`
	ValidationStatus string `json:"validation_status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func milestoneToResponse(m db.Milestone) MilestoneResponse {
	return MilestoneResponse{
		ID:               uuidToString(m.ID),
		WorkspaceID:      uuidToString(m.WorkspaceID),
		FeatureID:        uuidToString(m.FeatureID),
		Title:            m.Title,
		Position:         m.Position,
		ValidationStatus: m.ValidationStatus,
		CreatedAt:        timestampToString(m.CreatedAt),
		UpdatedAt:        timestampToString(m.UpdatedAt),
	}
}

// ListMilestones returns the workspace's milestones, ordered by Initiative then
// position. The board resolves an Issue's milestone title from this list, the
// same way it resolves its feature.
func (h *Handler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	var featureFilter pgtype.UUID
	if f := r.URL.Query().Get("feature_id"); f != "" {
		id, ok := parseUUIDOrBadRequest(w, f, "feature_id")
		if !ok {
			return
		}
		featureFilter = id
	}

	var milestones []db.Milestone
	var err error
	if featureFilter.Valid {
		milestones, err = h.Queries.ListMilestonesByFeature(r.Context(), featureFilter)
	} else {
		milestones, err = h.Queries.ListMilestonesByWorkspace(r.Context(), wsUUID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list milestones")
		return
	}

	resp := make([]MilestoneResponse, len(milestones))
	for i, m := range milestones {
		resp[i] = milestoneToResponse(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestones": resp, "total": len(resp)})
}

// CreateMilestoneRequest is the control-plane (MCP / UI) payload. Position is
// optional — omitted, the Milestone is appended to the end of its Initiative.
type CreateMilestoneRequest struct {
	FeatureID string `json:"feature_id"`
	Title     string `json:"title"`
	Position  *int32 `json:"position"`
}

// CreateMilestone adds a Milestone to an Initiative. The control plane owns
// creation (issue 14); validation_status always starts at 'pending' and is
// later advanced by the validator Run.
func (h *Handler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	var req CreateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	featureUUID, ok := parseUUIDOrBadRequest(w, req.FeatureID, "feature_id")
	if !ok {
		return
	}
	if _, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: featureUUID, WorkspaceID: wsUUID,
	}); err != nil {
		writeError(w, http.StatusBadRequest, "feature not found in this workspace")
		return
	}

	position := int32(0)
	if req.Position != nil {
		position = *req.Position
	} else if count, err := h.Queries.CountMilestonesByFeature(r.Context(), featureUUID); err == nil {
		position = int32(count)
	}

	milestone, err := h.Queries.CreateMilestone(r.Context(), db.CreateMilestoneParams{
		WorkspaceID: wsUUID,
		FeatureID:   featureUUID,
		Title:       req.Title,
		Position:    position,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create milestone")
		return
	}
	writeJSON(w, http.StatusCreated, milestoneToResponse(milestone))
}

// UpdateMilestoneRequest carries the control-plane editable fields. Only
// provided fields change; validation_status is execution-plane owned.
type UpdateMilestoneRequest struct {
	Title    *string `json:"title"`
	Position *int32  `json:"position"`
}

// UpdateMilestone edits a Milestone's title and ordering.
func (h *Handler) UpdateMilestone(w http.ResponseWriter, r *http.Request) {
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "milestone_id")
	if !ok {
		return
	}
	existing, err := h.Queries.GetMilestone(r.Context(), idUUID)
	if err != nil || existing.WorkspaceID != wsUUID {
		writeError(w, http.StatusNotFound, "milestone not found")
		return
	}

	var req UpdateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	milestone, err := h.Queries.UpdateMilestone(r.Context(), db.UpdateMilestoneParams{
		ID:       existing.ID,
		Title:    ptrToText(req.Title),
		Position: ptrToInt4(req.Position),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update milestone")
		return
	}
	writeJSON(w, http.StatusOK, milestoneToResponse(milestone))
}
