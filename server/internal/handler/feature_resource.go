package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// FeatureResourceResponse is the JSON shape returned by the feature resource API.
type FeatureResourceResponse struct {
	ID           string          `json:"id"`
	FeatureID    string          `json:"feature_id"`
	WorkspaceID  string          `json:"workspace_id"`
	ResourceType string          `json:"resource_type"`
	ResourceRef  json.RawMessage `json:"resource_ref"`
	Label        *string         `json:"label"`
	Position     int32           `json:"position"`
	CreatedAt    string          `json:"created_at"`
	CreatedBy    *string         `json:"created_by"`
}

func featureResourceToResponse(r db.FeatureResource) FeatureResourceResponse {
	ref := json.RawMessage(r.ResourceRef)
	if len(ref) == 0 {
		ref = json.RawMessage("{}")
	}
	return FeatureResourceResponse{
		ID:           uuidToString(r.ID),
		FeatureID:    uuidToString(r.FeatureID),
		WorkspaceID:  uuidToString(r.WorkspaceID),
		ResourceType: r.ResourceType,
		ResourceRef:  ref,
		Label:        textToPtr(r.Label),
		Position:     r.Position,
		CreatedAt:    timestampToString(r.CreatedAt),
		CreatedBy:    uuidToPtr(r.CreatedBy),
	}
}

// CreateFeatureResourceRequest is the body for POST /api/features/{id}/resources.
type CreateFeatureResourceRequest struct {
	ResourceType string          `json:"resource_type"`
	ResourceRef  json.RawMessage `json:"resource_ref"`
	Label        *string         `json:"label"`
	Position     *int32          `json:"position"`
}

// validateAndNormalizeResourceRef checks the payload for a known resource_type.
// New types are added here without schema migration; unknown types are rejected
// at the API boundary so a typo can't slip through and produce a resource the
// daemon/UI doesn't understand.
func validateAndNormalizeResourceRef(resourceType string, ref json.RawMessage) (json.RawMessage, error) {
	if len(ref) == 0 {
		return nil, errors.New("resource_ref is required")
	}
	switch resourceType {
	case "github_repo":
		return validateGithubRepoRef(ref)
	default:
		return nil, fmt.Errorf("unknown resource_type %q", resourceType)
	}
}

type githubRepoRef struct {
	URL                string `json:"url"`
	DefaultBranchHint  string `json:"default_branch_hint,omitempty"`
}

func validateGithubRepoRef(ref json.RawMessage) (json.RawMessage, error) {
	var payload githubRepoRef
	if err := json.Unmarshal(ref, &payload); err != nil {
		return nil, fmt.Errorf("invalid github_repo payload: %w", err)
	}
	payload.URL = strings.TrimSpace(payload.URL)
	if payload.URL == "" {
		return nil, errors.New("github_repo: url is required")
	}
	if !isValidGitRepoURL(payload.URL) {
		return nil, errors.New("github_repo: url must be a valid http(s) or ssh git URL")
	}
	payload.DefaultBranchHint = strings.TrimSpace(payload.DefaultBranchHint)
	out, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// isValidGitRepoURL accepts the three forms a user can paste from GitHub's
// "Code" menu: https://, ssh:// (with explicit scheme), and the scp-like
// shorthand `git@host:owner/repo.git`. The check is intentionally lax — we are
// guarding against pasted garbage like "not-a-url", not enforcing a strict
// grammar — because the actual fetch happens client-side via `git clone` and
// the user gets a clearer error from git than from us.
func isValidGitRepoURL(s string) bool {
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		switch u.Scheme {
		case "http", "https", "ssh", "git":
			return true
		}
	}
	// scp-like ssh shorthand: [user@]host:path with a non-empty host and path,
	// and no spaces. Reject anything that looks like a URL with a scheme
	// (those should go through url.Parse above).
	if strings.Contains(s, " ") || strings.Contains(s, "://") {
		return false
	}
	colon := strings.Index(s, ":")
	if colon <= 0 || colon == len(s)-1 {
		return false
	}
	// In scp-like ssh shorthand `[user@]host:path`, `@` is only meaningful
	// as a user separator before the first ':'. If '@' appears at or after
	// the colon it is not the user separator — reject as malformed rather
	// than guess (and avoid a slice-bounds panic from blindly slicing).
	at := strings.Index(s, "@")
	if at >= colon {
		return false
	}
	hostStart := 0
	if at >= 0 {
		hostStart = at + 1
	}
	host := s[hostStart:colon]
	path := s[colon+1:]
	if host == "" || path == "" {
		return false
	}
	return true
}

// loadFeatureForResource resolves the feature, enforces workspace ownership,
// and returns its DB row. Used by all feature_resource handlers.
func (h *Handler) loadFeatureForResource(w http.ResponseWriter, r *http.Request, featureIDParam string) (db.Feature, bool) {
	featureUUID, ok := parseUUIDOrBadRequest(w, featureIDParam, "feature id")
	if !ok {
		return db.Feature{}, false
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace id")
	if !ok {
		return db.Feature{}, false
	}
	feature, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: featureUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return db.Feature{}, false
	}
	return feature, true
}

// ListFeatureResources returns the resources attached to a feature.
func (h *Handler) ListFeatureResources(w http.ResponseWriter, r *http.Request) {
	feature, ok := h.loadFeatureForResource(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	resources, err := h.Queries.ListFeatureResources(r.Context(), feature.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list feature resources")
		return
	}
	resp := make([]FeatureResourceResponse, len(resources))
	for i, res := range resources {
		resp[i] = featureResourceToResponse(res)
	}
	writeJSON(w, http.StatusOK, map[string]any{"resources": resp, "total": len(resp)})
}

// CreateFeatureResource attaches a new resource to a feature.
func (h *Handler) CreateFeatureResource(w http.ResponseWriter, r *http.Request) {
	feature, ok := h.loadFeatureForResource(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req CreateFeatureResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ResourceType = strings.TrimSpace(req.ResourceType)
	if req.ResourceType == "" {
		writeError(w, http.StatusBadRequest, "resource_type is required")
		return
	}
	normalizedRef, err := validateAndNormalizeResourceRef(req.ResourceType, req.ResourceRef)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var label pgtype.Text
	if req.Label != nil && strings.TrimSpace(*req.Label) != "" {
		label = pgtype.Text{String: strings.TrimSpace(*req.Label), Valid: true}
	}
	var position int32
	if req.Position != nil {
		position = *req.Position
	} else {
		// Append after existing resources.
		count, _ := h.Queries.CountFeatureResources(r.Context(), feature.ID)
		position = int32(count)
	}

	creator, _ := h.parseUserUUIDOrZero(userID)
	resource, err := h.Queries.CreateFeatureResource(r.Context(), db.CreateFeatureResourceParams{
		FeatureID:    feature.ID,
		WorkspaceID:  feature.WorkspaceID,
		ResourceType: req.ResourceType,
		ResourceRef:  normalizedRef,
		Label:        label,
		Position:     position,
		CreatedBy:    creator,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "this resource is already attached to the feature")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create feature resource")
		return
	}

	resp := featureResourceToResponse(resource)
	h.publish(
		protocol.EventFeatureResourceCreated,
		uuidToString(feature.WorkspaceID),
		"member",
		userID,
		map[string]any{"resource": resp, "feature_id": uuidToString(feature.ID)},
	)
	writeJSON(w, http.StatusCreated, resp)
}

// DeleteFeatureResource removes a resource from a feature.
func (h *Handler) DeleteFeatureResource(w http.ResponseWriter, r *http.Request) {
	feature, ok := h.loadFeatureForResource(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	resourceUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "resourceId"), "resource id")
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	resource, err := h.Queries.GetFeatureResourceInWorkspace(r.Context(), db.GetFeatureResourceInWorkspaceParams{
		ID: resourceUUID, WorkspaceID: feature.WorkspaceID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "feature resource not found")
		return
	}
	if uuidToString(resource.FeatureID) != uuidToString(feature.ID) {
		writeError(w, http.StatusNotFound, "feature resource not found")
		return
	}
	if err := h.Queries.DeleteFeatureResource(r.Context(), resource.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete feature resource")
		return
	}
	h.publish(
		protocol.EventFeatureResourceDeleted,
		uuidToString(feature.WorkspaceID),
		"member",
		userID,
		map[string]any{
			"feature_id":  uuidToString(feature.ID),
			"resource_id": uuidToString(resource.ID),
		},
	)
	w.WriteHeader(http.StatusNoContent)
}

// parseUserUUIDOrZero converts a user ID string to a pgtype.UUID, returning a
// zero value on any error so the caller can store NULL for created_by when the
// authenticated principal is not a workspace member (e.g. internal-server use).
func (h *Handler) parseUserUUIDOrZero(userID string) (pgtype.UUID, bool) {
	if userID == "" {
		return pgtype.UUID{}, false
	}
	u, err := parseUUIDLoose(userID)
	if err != nil {
		return pgtype.UUID{}, false
	}
	return u, true
}

// parseUUIDLoose mirrors util.ParseUUID but lives here to avoid pulling util
// into a tiny one-off helper. Keep the body minimal.
func parseUUIDLoose(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

// listFeatureResourcesForFeature is a small helper used by the daemon claim
// handler to attach feature resources to outgoing tasks.
func (h *Handler) listFeatureResourcesForFeature(ctx context.Context, featureID pgtype.UUID) []db.FeatureResource {
	if !featureID.Valid {
		return nil
	}
	rows, err := h.Queries.ListFeatureResources(ctx, featureID)
	if err != nil {
		return nil
	}
	return rows
}
