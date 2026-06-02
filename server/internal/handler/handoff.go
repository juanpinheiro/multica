package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/handoff"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const taskRoleWorker = "worker"

// writeHandoffOnCompletion persists a structured Handoff when a worker Run
// completes. It is best-effort: errors are logged, not returned, so they
// never roll back the task-completion response.
//
// Guards: only runs for worker Runs (not validators) with an Issue and a
// non-nil HandoffInput.
func (h *Handler) writeHandoffOnCompletion(ctx context.Context, task *db.AgentTaskQueue, input *HandoffInput) {
	if task == nil || input == nil {
		return
	}
	if task.Role != taskRoleWorker {
		return
	}
	if !task.IssueID.Valid {
		return
	}

	issue, err := h.Queries.GetIssue(ctx, task.IssueID)
	if err != nil {
		slog.Warn("handoff: load issue failed",
			"task_id", uuidToString(task.ID), "issue_id", uuidToString(task.IssueID), "error", err)
		return
	}

	raw, err := serializeHandoffCommands(input.Commands)
	if err != nil {
		slog.Warn("handoff: serialize commands failed", "task_id", uuidToString(task.ID), "error", err)
		return
	}

	if _, err := h.Queries.CreateHandoff(ctx, db.CreateHandoffParams{
		WorkspaceID: issue.WorkspaceID,
		IssueID:     task.IssueID,
		RunID:       task.ID,
		Done:        nonNilStrings(input.Done),
		LeftUndone:  nonNilStrings(input.LeftUndone),
		Commands:    raw,
		Discoveries: nonNilStrings(input.Discoveries),
	}); err != nil {
		slog.Warn("handoff: create failed",
			"task_id", uuidToString(task.ID), "issue_id", uuidToString(task.IssueID), "error", err)
	}
}

// ListHandoffs returns all Handoffs for an Issue, ordered oldest-first.
func (h *Handler) ListHandoffs(w http.ResponseWriter, r *http.Request) {
	issueID := chi.URLParam(r, "id")
	issueUUID, ok := parseUUIDOrBadRequest(w, issueID, "issue_id")
	if !ok {
		return
	}

	rows, err := h.Queries.ListHandoffsByIssue(r.Context(), issueUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list handoffs")
		return
	}

	resp := make([]handoffResponse, len(rows))
	for i, row := range rows {
		resp[i] = handoffToResponse(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"handoffs": resp})
}

// handoffResponse is the JSON response shape for a Handoff.
type handoffResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	IssueID     string          `json:"issue_id"`
	RunID       string          `json:"run_id"`
	Done        []string        `json:"done"`
	LeftUndone  []string        `json:"left_undone"`
	Commands    json.RawMessage `json:"commands"`
	Discoveries []string        `json:"discoveries"`
	CreatedAt   string          `json:"created_at"`
}

func handoffToResponse(h db.Handoff) handoffResponse {
	cmds := json.RawMessage(h.Commands)
	if len(cmds) == 0 {
		cmds = json.RawMessage("[]")
	}
	return handoffResponse{
		ID:          uuidToString(h.ID),
		WorkspaceID: uuidToString(h.WorkspaceID),
		IssueID:     uuidToString(h.IssueID),
		RunID:       uuidToString(h.RunID),
		Done:        nonNilStrings(h.Done),
		LeftUndone:  nonNilStrings(h.LeftUndone),
		Commands:    cmds,
		Discoveries: nonNilStrings(h.Discoveries),
		CreatedAt:   timestampToString(h.CreatedAt),
	}
}

func serializeHandoffCommands(inputs []handoffCommandInput) ([]byte, error) {
	cmds := make([]handoff.CommandResult, len(inputs))
	for i, c := range inputs {
		cmds[i] = handoff.CommandResult{Command: c.Command, ExitCode: c.ExitCode}
	}
	return handoff.SerializeCommands(cmds)
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
