package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/dod"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	taskRoleValidator       = "validator"
	milestoneValidationPass = "passed"
	milestoneValidationFail = "failed"
)

// ValidationInput is the verdict set a validator Run submits on completion: one
// pass/fail result per DoD assertion it checked. All fields are optional.
type ValidationInput struct {
	Results []validationResultInput `json:"results"`
}

type validationResultInput struct {
	AssertionID string `json:"assertion_id"`
	Passed      bool   `json:"passed"`
	Detail      string `json:"detail"`
}

// recordValidationOnCompletion persists a validator Run's verdicts and then
// evaluates the Milestone's Definition of Done. Best-effort: errors are logged,
// not returned, so they never roll back the task-completion response.
//
// Guards: only validator Runs (not workers) with an Issue and a non-nil input.
func (h *Handler) recordValidationOnCompletion(ctx context.Context, task *db.AgentTaskQueue, input *ValidationInput) {
	if task == nil || input == nil {
		return
	}
	if task.Role != taskRoleValidator {
		return
	}
	if !task.IssueID.Valid {
		return
	}

	issue, err := h.Queries.GetIssue(ctx, task.IssueID)
	if err != nil {
		slog.Warn("dod: load issue failed", "task_id", uuidToString(task.ID), "error", err)
		return
	}
	if !issue.MilestoneID.Valid {
		return
	}

	h.persistValidatorVerdicts(ctx, issue.WorkspaceID, task.ID, input.Results)
	h.evaluateMilestoneDoD(ctx, issue)
	// The verdicts are now persisted and the Milestone's validation status is
	// settled, so the Orchestrator can safely reconcile: advance the Initiative
	// if every Milestone is green, or hold the gate while a follow-up is redone.
	h.orchestrateIssue(ctx, issue)
}

// persistValidatorVerdicts writes one DoD result row per verdict, skipping any
// whose assertion id is not a valid UUID (defensive against agent output).
func (h *Handler) persistValidatorVerdicts(ctx context.Context, workspaceID, runID pgtype.UUID, results []validationResultInput) {
	for _, res := range results {
		assertionID, err := util.ParseUUID(res.AssertionID)
		if err != nil {
			slog.Warn("dod: skip result with invalid assertion id", "run_id", uuidToString(runID), "assertion_id", res.AssertionID)
			continue
		}
		if _, err := h.Queries.CreateDodAssertionResult(ctx, db.CreateDodAssertionResultParams{
			WorkspaceID: workspaceID,
			AssertionID: assertionID,
			RunID:       runID,
			Passed:      res.Passed,
			Detail:      res.Detail,
		}); err != nil {
			slog.Warn("dod: create result failed", "run_id", uuidToString(runID), "assertion_id", res.AssertionID, "error", err)
		}
	}
}

// evaluateMilestoneDoD checks the Milestone's accumulated verdicts against its
// assertions. On green the Milestone is marked validated and the Gate opens; on
// failure it is marked failed (keeping the next Milestone gated) and a follow-up
// Issue is created so the Initiative self-heals.
func (h *Handler) evaluateMilestoneDoD(ctx context.Context, issue db.Issue) {
	assertions, err := h.Queries.ListDodAssertionsByMilestone(ctx, issue.MilestoneID)
	if err != nil {
		slog.Warn("dod: list assertions failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return
	}
	results, err := h.Queries.ListLatestDodResultsByMilestone(ctx, issue.MilestoneID)
	if err != nil {
		slog.Warn("dod: list results failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return
	}

	da, dr := toDodAssertions(assertions), toDodResults(results)
	if dod.MilestoneSatisfied(da, dr) {
		h.setMilestoneValidation(ctx, issue.MilestoneID, milestoneValidationPass)
		return
	}

	h.setMilestoneValidation(ctx, issue.MilestoneID, milestoneValidationFail)
	h.createDodFollowUpIssue(ctx, issue, dod.FailedAssertions(da, dr), assertions)
}

// createDodFollowUpIssue opens a worker Issue in the Milestone listing the failed
// assertions, assigned to the agent that owns the Milestone's work, and enqueues
// its worker Run so the Initiative self-heals without human input. It keeps the
// Milestone non-done so the Gate holds until the work is redone and re-validated.
func (h *Handler) createDodFollowUpIssue(ctx context.Context, issue db.Issue, failed []dod.Assertion, all []db.DodAssertion) {
	if len(failed) == 0 || !issue.AssigneeID.Valid {
		return
	}

	number, err := h.Queries.IncrementIssueCounter(ctx, issue.WorkspaceID)
	if err != nil {
		slog.Warn("dod: increment issue counter failed", "workspace_id", uuidToString(issue.WorkspaceID), "error", err)
		return
	}

	followUp, err := h.Queries.CreateDodFollowUpIssue(ctx, db.CreateDodFollowUpIssueParams{
		WorkspaceID: issue.WorkspaceID,
		FeatureID:   issue.FeatureID,
		MilestoneID: issue.MilestoneID,
		Title:       "Fix failed Definition of Done",
		Description: pgtype.Text{String: followUpBody(failed, all), Valid: true},
		AssigneeID:  issue.AssigneeID,
		Number:      number,
	})
	if err != nil {
		slog.Warn("dod: create follow-up issue failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return
	}

	if _, err := h.TaskService.EnqueueTaskForIssue(ctx, followUp); err != nil {
		slog.Warn("dod: enqueue follow-up worker run failed", "issue_id", uuidToString(followUp.ID), "error", err)
	}
}

// followUpBody renders the failed assertions as a checklist, resolving each id
// back to its assertion text from the Milestone's assertion rows.
func followUpBody(failed []dod.Assertion, all []db.DodAssertion) string {
	text := make(map[string]string, len(all))
	for _, a := range all {
		text[uuidToString(a.ID)] = a.Text
	}
	var b strings.Builder
	b.WriteString("These Definition-of-Done assertions failed validation and must be satisfied:\n\n")
	for _, f := range failed {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", text[f.ID]))
	}
	return b.String()
}

// resolveValidatorAgent picks an agent distinct from the worker to keep the
// creator-verifier separation, falling back to the worker agent (run with a
// fresh session) when no other agent exists.
func (h *Handler) resolveValidatorAgent(ctx context.Context, issue db.Issue) pgtype.UUID {
	if issue.AssigneeID.Valid {
		other, err := h.Queries.GetValidatorAgent(ctx, db.GetValidatorAgentParams{
			WorkspaceID: issue.WorkspaceID,
			ID:          issue.AssigneeID,
		})
		if err == nil {
			return other
		}
		if !isNotFound(err) {
			slog.Warn("dod: resolve validator agent failed", "workspace_id", uuidToString(issue.WorkspaceID), "error", err)
		}
	}
	return issue.AssigneeID
}

// setMilestoneValidation performs a validation-status write. Best-effort: errors
// are logged, not returned, since it runs as a side-effect of a completed Run.
func (h *Handler) setMilestoneValidation(ctx context.Context, milestoneID pgtype.UUID, status string) {
	if _, err := h.Queries.SetMilestoneValidationStatus(ctx, db.SetMilestoneValidationStatusParams{
		ID:               milestoneID,
		ValidationStatus: status,
	}); err != nil {
		slog.Warn("dod: set milestone validation failed", "milestone_id", uuidToString(milestoneID), "status", status, "error", err)
	}
}

// ListMilestoneDoD returns a Milestone's assertions, each carrying its latest
// validation status (passed/failed/pending) — the monitor's DoD pass/fail view.
func (h *Handler) ListMilestoneDoD(w http.ResponseWriter, r *http.Request) {
	milestoneID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "milestone_id")
	if !ok {
		return
	}

	assertions, err := h.Queries.ListDodAssertionsByMilestone(r.Context(), milestoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list dod assertions")
		return
	}
	results, err := h.Queries.ListLatestDodResultsByMilestone(r.Context(), milestoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list dod results")
		return
	}

	latest := make(map[string]db.DodAssertionResult, len(results))
	for _, res := range results {
		latest[uuidToString(res.AssertionID)] = res
	}

	resp := make([]dodAssertionResponse, len(assertions))
	for i, a := range assertions {
		resp[i] = dodAssertionToResponse(a, latest[uuidToString(a.ID)])
	}
	writeJSON(w, http.StatusOK, map[string]any{"assertions": resp})
}

// ListIssueDoD returns the Definition of Done derived for an Issue: the
// assertions of its Milestone (the per-Issue Acceptance Criteria view).
func (h *Handler) ListIssueDoD(w http.ResponseWriter, r *http.Request) {
	issueID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "issue_id")
	if !ok {
		return
	}

	assertions, err := h.Queries.ListDodAssertionsByIssue(r.Context(), issueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list dod assertions")
		return
	}

	resp := make([]dodAssertionResponse, len(assertions))
	for i, a := range assertions {
		resp[i] = dodAssertionToResponse(a, db.DodAssertionResult{})
	}
	writeJSON(w, http.StatusOK, map[string]any{"assertions": resp})
}

// CreateDodAssertionRequest is the control-plane payload for writing a single
// Definition-of-Done assertion under a Milestone. Position is optional —
// omitted, the assertion is appended to the Milestone's list.
type CreateDodAssertionRequest struct {
	Text     string `json:"text"`
	Position *int32 `json:"position"`
}

// CreateDodAssertion writes a DoD assertion tagged to the Milestone in the URL.
// The feature is derived from the Milestone so the control plane only supplies
// the assertion text.
func (h *Handler) CreateDodAssertion(w http.ResponseWriter, r *http.Request) {
	wsUUID, ok := parseUUIDOrBadRequest(w, h.resolveWorkspaceID(r), "workspace_id")
	if !ok {
		return
	}
	milestoneID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "milestone_id")
	if !ok {
		return
	}
	milestone, err := h.Queries.GetMilestone(r.Context(), milestoneID)
	if err != nil || milestone.WorkspaceID != wsUUID {
		writeError(w, http.StatusNotFound, "milestone not found")
		return
	}

	var req CreateDodAssertionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}

	position := int32(0)
	if req.Position != nil {
		position = *req.Position
	} else if existing, err := h.Queries.ListDodAssertionsByMilestone(r.Context(), milestoneID); err == nil {
		position = int32(len(existing))
	}

	assertion, err := h.Queries.CreateDodAssertion(r.Context(), db.CreateDodAssertionParams{
		WorkspaceID: wsUUID,
		FeatureID:   milestone.FeatureID,
		MilestoneID: milestoneID,
		Text:        req.Text,
		Position:    position,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create dod assertion")
		return
	}
	writeJSON(w, http.StatusCreated, dodAssertionToResponse(assertion, db.DodAssertionResult{}))
}

type dodAssertionResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	FeatureID   string `json:"feature_id"`
	MilestoneID string `json:"milestone_id"`
	Text        string `json:"text"`
	Position    int32  `json:"position"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
}

// dodAssertionToResponse derives the per-assertion status from its latest
// verdict: a zero-value (no run_id) result means the assertion is still pending.
func dodAssertionToResponse(a db.DodAssertion, latest db.DodAssertionResult) dodAssertionResponse {
	status := "pending"
	if latest.RunID.Valid {
		if latest.Passed {
			status = milestoneValidationPass
		} else {
			status = milestoneValidationFail
		}
	}
	return dodAssertionResponse{
		ID:          uuidToString(a.ID),
		WorkspaceID: uuidToString(a.WorkspaceID),
		FeatureID:   uuidToString(a.FeatureID),
		MilestoneID: uuidToString(a.MilestoneID),
		Text:        a.Text,
		Position:    a.Position,
		CreatedAt:   timestampToString(a.CreatedAt),
		Status:      status,
		Detail:      latest.Detail,
	}
}

func toDodAssertions(rows []db.DodAssertion) []dod.Assertion {
	out := make([]dod.Assertion, len(rows))
	for i, a := range rows {
		out[i] = dod.Assertion{ID: uuidToString(a.ID)}
	}
	return out
}

func toDodResults(rows []db.DodAssertionResult) []dod.Result {
	out := make([]dod.Result, len(rows))
	for i, r := range rows {
		out[i] = dod.Result{AssertionID: uuidToString(r.AssertionID), Passed: r.Passed}
	}
	return out
}
