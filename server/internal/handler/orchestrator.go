package handler

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/initiative"
	"github.com/multica-ai/multica/server/internal/orchestrator"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// RegisterOrchestrator wires the Orchestrator to the task:completed event bus.
//
// The Orchestrator is the stateless "COO" of an in-flight Initiative (ADR-0004):
// it owns no session and retains nothing between wakes. Every wake rebuilds the
// Initiative's state from the DB and reconciles it through the pure
// orchestrator.Decide core, so a process restart mid-run resumes with nothing
// lost — the durable board, Milestones, and validator verdicts are the only
// state. The intelligence stays thin and deterministic here; richer judgment is
// expected to migrate into a prompt-driven Orchestrator Run over this same seam.
//
// Wake sources, all funnelling into the same idempotent reconcile:
//   - worker Run task:completed — via this bus subscription;
//   - an Issue reaching done — via orchestrateOnIssueDone at the issue handlers;
//   - a validator Run recording its verdicts — via recordValidationOnCompletion.
//
// Validator completions are deliberately NOT reconciled from the raw bus event:
// task:completed is published synchronously inside CompleteTask, before the
// completion handler persists the validator's verdicts, so a bus-driven reconcile
// would read stale DoD results. They are reconciled after persistence instead.
func (h *Handler) RegisterOrchestrator(bus *events.Bus) {
	bus.Subscribe(protocol.EventTaskCompleted, func(e events.Event) {
		h.onTaskCompleted(context.Background(), e)
	})
}

// onTaskCompleted reconciles the Initiative of a completed worker Run. Validator
// Runs are skipped here (see RegisterOrchestrator).
func (h *Handler) onTaskCompleted(ctx context.Context, e events.Event) {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return
	}
	rawID, _ := payload["task_id"].(string)
	taskID, err := util.ParseUUID(rawID)
	if err != nil {
		return
	}
	task, err := h.Queries.GetAgentTask(ctx, taskID)
	if err != nil {
		return
	}
	if task.Role == taskRoleValidator || task.Role == taskRoleRetrospective || !task.IssueID.Valid {
		return
	}
	issue, err := h.Queries.GetIssue(ctx, task.IssueID)
	if err != nil {
		return
	}
	h.orchestrateIssue(ctx, issue)
}

// orchestrateOnIssueDone reconciles an Initiative when one of its Issues
// transitions to done. It replaces the hardcoded boundary triggers: the
// Orchestrator now decides whether to dispatch a validator, pass a Milestone, or
// advance the Initiative.
func (h *Handler) orchestrateOnIssueDone(ctx context.Context, prev, issue db.Issue) {
	if prev.Status == "done" || issue.Status != "done" {
		return
	}
	h.orchestrateIssue(ctx, issue)
}

// orchestrateIssue is the single reconcile entrypoint: it rebuilds the
// Initiative's state around the triggering Issue, asks the pure core what to do,
// and applies the Plan through the dispatch service. Idempotent — re-running it
// on unchanged state is a no-op.
func (h *Handler) orchestrateIssue(ctx context.Context, issue db.Issue) {
	if !issue.FeatureID.Valid {
		return
	}
	state, ok := h.loadOrchestratorState(ctx, issue)
	if !ok {
		return
	}

	// The AFK safety net runs before any dispatch decision: a tripped Initiative
	// is paused (moved to blocked + the human pinged) instead of advancing.
	if h.pauseOnTripwire(ctx, issue, state.InitiativeStatus) {
		return
	}

	plan := orchestrator.Decide(state)
	if plan.PassMilestone {
		h.setMilestoneValidation(ctx, issue.MilestoneID, milestoneValidationPass)
	}
	if plan.DispatchValidator {
		h.dispatchValidator(ctx, issue)
	}
	if plan.AdvanceTo != "" {
		h.advanceInitiative(ctx, issue.FeatureID, initiative.Status(state.InitiativeStatus), initiative.Status(plan.AdvanceTo))
		if plan.AdvanceTo == string(initiative.StatusInReview) {
			h.notifyInitiativeReadyForReview(ctx, issue.FeatureID)
			// The Initiative boundary is reached: run a retrospective to record
			// the Decision Log and refresh the architecture docs (issue 19).
			h.dispatchRetrospective(ctx, issue)
		}
	}
}

// loadOrchestratorState reads the Initiative's durable state fresh from the DB.
// ok is false when the Initiative cannot be loaded (the wake is dropped).
func (h *Handler) loadOrchestratorState(ctx context.Context, issue db.Issue) (orchestrator.State, bool) {
	feature, err := h.Queries.GetFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("orchestrator: load feature failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		return orchestrator.State{}, false
	}

	state := orchestrator.State{
		InitiativeStatus: feature.Status,
		TriggerIssueDone: issue.Status == "done",
	}

	featureOpen, err := h.Queries.CountNonDoneFeatureSiblings(ctx, db.CountNonDoneFeatureSiblingsParams{
		FeatureID: issue.FeatureID,
		ID:        issue.ID,
	})
	if err != nil {
		slog.Warn("orchestrator: count feature siblings failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		return orchestrator.State{}, false
	}
	state.FeatureOpenSiblings = int(featureOpen)

	milestones, err := h.Queries.ListMilestonesByFeature(ctx, issue.FeatureID)
	if err != nil {
		slog.Warn("orchestrator: list milestones failed", "feature_id", uuidToString(issue.FeatureID), "error", err)
		return orchestrator.State{}, false
	}
	state.AllMilestones = toOrchestratorMilestones(milestones)

	if issue.MilestoneID.Valid {
		if !h.loadTriggerMilestone(ctx, issue, &state) {
			return orchestrator.State{}, false
		}
	}
	return state, true
}

// loadTriggerMilestone fills in the facts about the triggering Issue's Milestone.
func (h *Handler) loadTriggerMilestone(ctx context.Context, issue db.Issue, state *orchestrator.State) bool {
	milestone, err := h.Queries.GetMilestone(ctx, issue.MilestoneID)
	if err != nil {
		slog.Warn("orchestrator: load milestone failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return false
	}
	openSiblings, err := h.Queries.CountNonDoneMilestoneSiblings(ctx, db.CountNonDoneMilestoneSiblingsParams{
		MilestoneID: issue.MilestoneID,
		ID:          issue.ID,
	})
	if err != nil {
		slog.Warn("orchestrator: count milestone siblings failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return false
	}
	assertions, err := h.Queries.ListDodAssertionsByMilestone(ctx, issue.MilestoneID)
	if err != nil {
		slog.Warn("orchestrator: list assertions failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return false
	}
	activeValidators, err := h.Queries.CountActiveValidatorRunsByMilestone(ctx, issue.MilestoneID)
	if err != nil {
		slog.Warn("orchestrator: count active validators failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
		return false
	}

	state.HasMilestone = true
	state.TriggerMilestoneID = uuidToString(issue.MilestoneID)
	state.TriggerMilestoneValidation = milestone.ValidationStatus
	state.TriggerMilestoneHasAssertions = len(assertions) > 0
	state.TriggerMilestoneOpenSiblings = int(openSiblings)
	state.TriggerMilestoneActiveValidators = int(activeValidators)
	return true
}

// dispatchValidator enqueues a validator Run for the Milestone's accumulated
// work, resolving an agent distinct from the worker where one exists.
func (h *Handler) dispatchValidator(ctx context.Context, issue db.Issue) {
	validatorAgent := h.resolveValidatorAgent(ctx, issue)
	if !validatorAgent.Valid {
		return
	}
	if _, err := h.TaskService.DispatchValidatorRun(ctx, issue, validatorAgent); err != nil {
		slog.Warn("orchestrator: dispatch validator failed", "milestone_id", uuidToString(issue.MilestoneID), "error", err)
	}
}

// advanceInitiative moves the Initiative toward to, stepping through running
// first when it is still ready (so each hop stays a legal transition). Best-
// effort: illegal moves are skipped, errors logged by setFeatureStatus.
func (h *Handler) advanceInitiative(ctx context.Context, featureID pgtype.UUID, from, to initiative.Status) {
	cur := from
	if cur == initiative.StatusReady && to != initiative.StatusReady {
		if initiative.Transition(cur, initiative.StatusRunning) == nil {
			h.setFeatureStatus(ctx, featureID, initiative.StatusRunning)
			cur = initiative.StatusRunning
		}
	}
	if initiative.Transition(cur, to) != nil {
		return
	}
	h.setFeatureStatus(ctx, featureID, to)
}

func toOrchestratorMilestones(rows []db.Milestone) []orchestrator.Milestone {
	out := make([]orchestrator.Milestone, len(rows))
	for i, m := range rows {
		out[i] = orchestrator.Milestone{ID: uuidToString(m.ID), ValidationStatus: m.ValidationStatus}
	}
	return out
}
