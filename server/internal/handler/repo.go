package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type RepoResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	Name          string  `json:"name"`
	RemoteURL     string  `json:"remote_url"`
	LocalPath     *string `json:"local_path"`
	DefaultBranch string  `json:"default_branch"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func repoToResponse(r db.Repo) RepoResponse {
	return RepoResponse{
		ID:            uuidToString(r.ID),
		WorkspaceID:   uuidToString(r.WorkspaceID),
		Name:          r.Name,
		RemoteURL:     r.RemoteUrl,
		LocalPath:     textToPtr(r.LocalPath),
		DefaultBranch: r.DefaultBranch,
		CreatedAt:     timestampToString(r.CreatedAt),
		UpdatedAt:     timestampToString(r.UpdatedAt),
	}
}

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	repos, err := h.Queries.ListReposInWorkspace(r.Context(), wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list repos")
		return
	}
	resp := make([]RepoResponse, len(repos))
	for i, repo := range repos {
		resp[i] = repoToResponse(repo)
	}
	writeJSON(w, http.StatusOK, resp)
}

type CreateRepoRequest struct {
	Name          string  `json:"name"`
	RemoteURL     string  `json:"remote_url"`
	LocalPath     *string `json:"local_path"`
	DefaultBranch *string `json:"default_branch"`
}

func (h *Handler) CreateRepo(w http.ResponseWriter, r *http.Request) {
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}

	var req CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.RemoteURL = strings.TrimSpace(req.RemoteURL)
	if req.Name == "" || req.RemoteURL == "" {
		writeError(w, http.StatusBadRequest, "name and remote_url are required")
		return
	}

	var defaultBranch any
	if req.DefaultBranch != nil && strings.TrimSpace(*req.DefaultBranch) != "" {
		defaultBranch = strings.TrimSpace(*req.DefaultBranch)
	}

	repo, err := h.Queries.CreateRepo(r.Context(), db.CreateRepoParams{
		WorkspaceID:   wsUUID,
		Name:          req.Name,
		RemoteUrl:     req.RemoteURL,
		LocalPath:     ptrToText(req.LocalPath),
		DefaultBranch: defaultBranch,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "a repo with this name or remote_url already exists in the workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create repo")
		return
	}
	writeJSON(w, http.StatusCreated, repoToResponse(repo))
}

func (h *Handler) GetRepo(w http.ResponseWriter, r *http.Request) {
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "repo id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	repo, err := h.Queries.GetRepoInWorkspace(r.Context(), db.GetRepoInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	writeJSON(w, http.StatusOK, repoToResponse(repo))
}

func (h *Handler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "repo id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	rows, err := h.Queries.DeleteRepo(r.Context(), db.DeleteRepoParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete repo")
		return
	}
	if rows == 0 {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
