package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// notifyParentOfChildDone posts a top-level system comment on the parent
// issue when a child issue transitions from non-done into done. This replaces
// the agent-prompt rule that previously made child agents post the
// notification themselves (PR #2918 user feedback — the agent rule caused
// self-mention loops, planner ping-pong, and accidental `MUL-` prefix
// hardcoding because the agent did not always know the workspace prefix).
//
// Guards on whether the comment fires at all:
//   - prev.Status must not already be "done" (idempotent — repeat saves of
//     done do not re-fire; only the transition fires)
//   - issue.Status must be "done"
//   - issue.ParentIssueID must be set
//   - parent must not be "done" or "cancelled" — the parent is already
//     closed and a notification has no follow-up to drive
//   - parent assignee must not be a member (human). Humans read their
//     issues manually; an automated system comment is pure noise for them
//     and there is nothing to "trigger" on a human assignee. Skipping the
//     comment entirely (Bohan's call on MUL-2538) also sidesteps the
//     mention question — no comment, no mention, no inbox row.
//
// The comment is inserted directly via db.Queries (not through the
// CreateComment HTTP handler) so it bypasses the generic on_comment trigger
// path. When the parent has an agent assignee, the comment body embeds a
// `mention://agent/<id>` link that targets the parent assignee. To keep the
// platform in control of side effects, the cmd/server notification + subscriber
// listeners still skip system comments wholesale, so smuggled mentions from
// the child title cannot light up unrelated members. The parent assignee's
// own trigger is fired explicitly by dispatchParentAssigneeTrigger below,
// with the loop and idempotency guards documented there.
//
// Errors are logged at warn level and swallowed: this is a best-effort
// notification on the side of a successful status update; failing it must
// not roll back the user's status change.
func (h *Handler) notifyParentOfChildDone(ctx context.Context, prev, issue db.Issue) {
	if !issue.ParentIssueID.Valid {
		return
	}
	if prev.Status == "done" || issue.Status != "done" {
		return
	}
	parent, err := h.Queries.GetIssue(ctx, issue.ParentIssueID)
	if err != nil {
		slog.Warn("child done: failed to load parent",
			"error", err,
			"child_id", uuidToString(issue.ID),
			"parent_id", uuidToString(issue.ParentIssueID))
		return
	}
	if parent.Status == "done" || parent.Status == "cancelled" {
		return
	}
	// Human-assigned parents read their own timeline; an automated system
	// comment is just noise and there is no agent task to trigger. Skip the
	// whole notification (comment + mention + inbox row) — MUL-2538.
	if parent.AssigneeType.Valid && parent.AssigneeType.String == "member" {
		return
	}

	prefix := h.getIssuePrefix(ctx, issue.WorkspaceID)
	identifier := prefix + "-" + strconv.Itoa(int(issue.Number))
	childID := uuidToString(issue.ID)
	title := sanitizeChildTitleForSystemComment(issue.Title)

	// Build the parent-assignee mention prefix. Empty when the parent has no
	// assignee or the assignee row is missing (deleted member, archived
	// agent the workspace lost track of, etc.).
	mentionPrefix := h.buildParentAssigneeMention(ctx, parent)

	content := fmt.Sprintf(
		"%sSub-issue [%s](mention://issue/%s) — \"%s\" — is done. Before promoting any waiting `backlog` sub-issue, read each sibling's description and only promote items whose stated dependencies are already satisfied — do not rely on this parent's higher-level breakdown alone. If a sibling's description conflicts with that breakdown (e.g. it lists a prerequisite the parent treats as parallel), do NOT change its status — leave it `backlog` and post a comment to confirm first.",
		mentionPrefix, identifier, childID, title,
	)

	// author_type='system', author_id=zero UUID. The zero UUID is a valid 16
	// byte value and the column is NOT NULL; frontend code should branch on
	// author_type === 'system' rather than on the UUID value.
	comment, err := h.Queries.CreateComment(ctx, db.CreateCommentParams{
		IssueID:     parent.ID,
		WorkspaceID: parent.WorkspaceID,
		AuthorType:  "system",
		AuthorID:    pgtype.UUID{Valid: true},
		Content:     content,
		Type:        "system",
		ParentID:    pgtype.UUID{Valid: false},
	})
	if err != nil {
		slog.Warn("child done: create system comment failed",
			"error", err,
			"child_id", childID,
			"parent_id", uuidToString(parent.ID))
		return
	}

	h.publish(protocol.EventCommentCreated, uuidToString(parent.WorkspaceID), "system", "", map[string]any{
		"comment":             commentToResponse(comment, nil),
		"issue_title":         parent.Title,
		"issue_assignee_type": textToPtr(parent.AssigneeType),
		"issue_assignee_id":   uuidToPtr(parent.AssigneeID),
		"issue_status":        parent.Status,
	})

	// Dispatch the explicit trigger / inbox row for the parent assignee.
	// Listener-level mention parsing is intentionally NOT involved (the
	// notification + subscriber listeners both short-circuit on
	// author_type='system'); this keeps smuggled mentions from the child
	// title inert and gives the platform a single place to apply the loop
	// and idempotency guards.
	h.dispatchParentAssigneeTrigger(ctx, parent, issue, comment)
}

// sanitizeChildTitleForSystemComment removes mention-style markdown from a
// child issue's title before it is embedded into the parent's system
// comment. Smuggled mentions are already harmless on the listener path
// (notification + subscriber listeners both skip system comments), but the
// timeline still renders the title verbatim — stripping the markdown keeps
// the rendered comment readable and stops a maliciously titled child issue
// from looking like a directive ("@all please look").
func sanitizeChildTitleForSystemComment(title string) string {
	// Replace any markdown link target so the regex no longer matches it,
	// while preserving the human-readable label text. `]` and `(` are the
	// minimum delimiters of the mention regex; replacing the `(` is enough
	// to break the match without mangling the label.
	cleaned := strings.ReplaceAll(title, "](mention://", "] (mention-stripped://")
	return cleaned
}

// buildParentAssigneeMention returns the markdown prefix that the system
// comment should lead with, including a trailing space, so the body reads
// like a normal mention-led comment. Returns the empty string when the
// parent has no assignee or the assignee row could not be loaded.
func (h *Handler) buildParentAssigneeMention(ctx context.Context, parent db.Issue) string {
	if !parent.AssigneeType.Valid || !parent.AssigneeID.Valid {
		return ""
	}
	label, ok := h.resolveAssigneeMentionLabel(ctx, parent.WorkspaceID, parent.AssigneeType.String, parent.AssigneeID)
	if !ok {
		return ""
	}
	return fmt.Sprintf("[@%s](mention://%s/%s) ", label, parent.AssigneeType.String, uuidToString(parent.AssigneeID))
}

// resolveAssigneeMentionLabel returns the label text to render inside the
// mention link. The label is for human display only — the mention regex
// keys off the URL path, not the label — but a sensible fallback keeps the
// rendered comment legible if the frontend has not pre-loaded the assignee.
// Returns ok=false when the assignee row cannot be loaded; the caller
// should then omit the mention entirely rather than emit a broken link.
func (h *Handler) resolveAssigneeMentionLabel(ctx context.Context, workspaceID pgtype.UUID, assigneeType string, assigneeID pgtype.UUID) (string, bool) {
	if assigneeType != "agent" {
		return "", false
	}
	agent, err := h.Queries.GetAgentInWorkspace(ctx, db.GetAgentInWorkspaceParams{
		ID:          assigneeID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return "", false
	}
	return sanitizeMentionLabel(agent.Name), true
}

// sanitizeMentionLabel strips characters that would break the mention
// markdown if a name contained them. The mention regex is non-greedy on the
// label, so a stray `]` would short-circuit it. Names with `]` are
// vanishingly rare but cheap to defend against.
func sanitizeMentionLabel(name string) string {
	cleaned := strings.ReplaceAll(name, "]", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "assignee"
	}
	return cleaned
}

// dispatchParentAssigneeTrigger fires the explicit side effect that pairs
// with the @mention link in the system comment body — an agent task for
// agent assignees. Member assignees never reach this code path;
// notifyParentOfChildDone skips them outright. The generic comment listener
// is intentionally bypassed (it short-circuits on author_type='system'), so
// this is the single place where the platform applies loop and idempotency
// guards for the child-done notification.
func (h *Handler) dispatchParentAssigneeTrigger(ctx context.Context, parent, child db.Issue, systemComment db.Comment) {
	if !parent.AssigneeType.Valid || !parent.AssigneeID.Valid || parent.AssigneeType.String != "agent" {
		return
	}
	h.triggerChildDoneAgent(ctx, parent, child, systemComment.ID)
}

// triggerChildDoneAgent enqueues a mention-style task for the parent's agent
// assignee. Skips if the child's agent assignee is the same as the parent's
// (self-loop guard).
func (h *Handler) triggerChildDoneAgent(ctx context.Context, parent, child db.Issue, triggerCommentID pgtype.UUID) {
	if child.AssigneeType.Valid && child.AssigneeType.String == "agent" && child.AssigneeID.Valid &&
		uuidToString(child.AssigneeID) == uuidToString(parent.AssigneeID) {
		return
	}

	agent, err := h.Queries.GetAgentInWorkspace(ctx, db.GetAgentInWorkspaceParams{
		ID:          parent.AssigneeID,
		WorkspaceID: parent.WorkspaceID,
	})
	if err != nil || !agent.RuntimeID.Valid || agent.ArchivedAt.Valid {
		return
	}

	hasPending, err := h.Queries.HasPendingTaskForIssueAndAgent(ctx, db.HasPendingTaskForIssueAndAgentParams{
		IssueID: parent.ID,
		AgentID: parent.AssigneeID,
	})
	if err != nil || hasPending {
		return
	}

	if _, err := h.TaskService.EnqueueTaskForMention(ctx, parent, parent.AssigneeID, triggerCommentID); err != nil {
		slog.Warn("child done: enqueue parent agent task failed",
			"error", err,
			"parent_id", uuidToString(parent.ID),
			"agent_id", uuidToString(parent.AssigneeID))
	}
}
