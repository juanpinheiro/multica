package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/decisionlog"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const taskRoleRetrospective = "retrospective"

// recordRetrospectiveOnCompletion persists the Decision Log entries a
// retrospective Run recorded when reviewing a finished Initiative. Best-effort:
// errors are logged, not returned, so they never roll back the task-completion
// response.
//
// Guards: only retrospective Runs (not workers/validators) with an Issue whose
// Initiative is known, and a non-nil output.
func (h *Handler) recordRetrospectiveOnCompletion(ctx context.Context, task *db.AgentTaskQueue, input *decisionlog.Output) {
	if task == nil || input == nil {
		return
	}
	if task.Role != taskRoleRetrospective || !task.IssueID.Valid {
		return
	}

	issue, err := h.Queries.GetIssue(ctx, task.IssueID)
	if err != nil {
		slog.Warn("decision log: load issue failed", "task_id", uuidToString(task.ID), "error", err)
		return
	}
	if !issue.FeatureID.Valid {
		return
	}

	for _, e := range decisionlog.ValidEntries(input) {
		if _, err := h.Queries.CreateDecisionLogEntry(ctx, db.CreateDecisionLogEntryParams{
			WorkspaceID:  issue.WorkspaceID,
			FeatureID:    issue.FeatureID,
			RunID:        task.ID,
			Title:        e.Title,
			Decision:     e.Decision,
			Learning:     e.Learning,
			AdrRefs:      e.AdrRefs,
			ContextTerms: e.ContextTerms,
		}); err != nil {
			slog.Warn("decision log: create entry failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		}
	}
}

// dispatchRetrospective enqueues a retrospective Run for the Initiative at its
// boundary, skipping when one is already in flight. Best-effort side-effect of
// advancing the Initiative to review.
func (h *Handler) dispatchRetrospective(ctx context.Context, issue db.Issue) {
	if !issue.AssigneeID.Valid {
		return
	}
	active, err := h.Queries.CountActiveRetrospectiveRunsByFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("decision log: count active retrospectives failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		return
	}
	if active > 0 {
		return
	}
	if _, err := h.TaskService.DispatchRetrospectiveRun(ctx, issue, issue.AssigneeID); err != nil {
		slog.Warn("decision log: dispatch retrospective failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
	}
}

// ListDecisionLog returns an Initiative's recorded decisions, newest first.
func (h *Handler) ListDecisionLog(w http.ResponseWriter, r *http.Request) {
	featureID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "feature_id")
	if !ok {
		return
	}

	rows, err := h.Queries.ListDecisionLogByFeature(r.Context(), featureID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list decision log")
		return
	}

	resp := make([]decisionLogResponse, len(rows))
	for i, row := range rows {
		resp[i] = decisionLogToResponse(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": resp})
}

type decisionLogResponse struct {
	ID           string   `json:"id"`
	WorkspaceID  string   `json:"workspace_id"`
	FeatureID    string   `json:"feature_id"`
	RunID        string   `json:"run_id"`
	Title        string   `json:"title"`
	Decision     string   `json:"decision"`
	Learning     string   `json:"learning"`
	AdrRefs      []string `json:"adr_refs"`
	ContextTerms []string `json:"context_terms"`
	CreatedAt    string   `json:"created_at"`
}

func decisionLogToResponse(d db.DecisionLog) decisionLogResponse {
	return decisionLogResponse{
		ID:           uuidToString(d.ID),
		WorkspaceID:  uuidToString(d.WorkspaceID),
		FeatureID:    uuidToString(d.FeatureID),
		RunID:        uuidToString(d.RunID),
		Title:        d.Title,
		Decision:     d.Decision,
		Learning:     d.Learning,
		AdrRefs:      nonNilStrings(d.AdrRefs),
		ContextTerms: nonNilStrings(d.ContextTerms),
		CreatedAt:    timestampToString(d.CreatedAt),
	}
}
