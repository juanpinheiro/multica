package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// makeMilestone inserts a milestone under a feature at the given position and
// validation status, returning its id.
func (f initiativeGateFixture) makeMilestone(featureID string, position int, validationStatus string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO milestone (workspace_id, feature_id, title, position, validation_status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, testWorkspaceID, featureID, fmt.Sprintf("ms-%d", time.Now().UnixNano()), position, validationStatus).Scan(&id); err != nil {
		f.t.Fatalf("create milestone: %v", err)
	}
	f.t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM milestone WHERE id = $1`, id) })
	return id
}

// makeIssueInMilestone inserts an issue under a feature and milestone.
func (f initiativeGateFixture) makeIssueInMilestone(featureID, milestoneID, label string) string {
	f.t.Helper()
	issueID := f.makeIssue(featureID, label)
	if _, err := testPool.Exec(f.ctx, `UPDATE issue SET milestone_id = $1 WHERE id = $2`, milestoneID, issueID); err != nil {
		f.t.Fatalf("assign issue to milestone: %v", err)
	}
	return issueID
}

func (f initiativeGateFixture) passMilestone(milestoneID string) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `UPDATE milestone SET validation_status = 'passed' WHERE id = $1`, milestoneID); err != nil {
		f.t.Fatalf("pass milestone: %v", err)
	}
}

// TestClaimAgentTask_MilestoneGate verifies milestone-gating: a Milestone's
// Issues are not claimable until every earlier Milestone in the Initiative has
// passed validation. The Initiative is 'running' so the status gate is open and
// only milestone-gating is under test.
func TestClaimAgentTask_MilestoneGate(t *testing.T) {
	f := newInitiativeGateFixture(t)

	mustNotClaim := func(agentID string) {
		t.Helper()
		if row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(agentID)); err == nil {
			t.Fatalf("expected no claim, got task %s", row.ID)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows, got %v", err)
		}
	}
	mustClaim := func(agentID, wantTaskID string) {
		t.Helper()
		row, err := f.queries.ClaimAgentTask(f.ctx, parseUUID(agentID))
		if err != nil {
			t.Fatalf("expected claim, got error: %v", err)
		}
		if got := uuidToString(row.ID); got != wantTaskID {
			t.Fatalf("claimed task id = %s, want %s", got, wantTaskID)
		}
	}

	t.Run("first milestone's issue is claimable (no earlier milestone)", func(t *testing.T) {
		feature := f.makeFeature("running")
		m1 := f.makeMilestone(feature, 0, "pending")
		f.makeMilestone(feature, 1, "pending")
		agentID := f.makeAgent("ms-gate-first")
		issueID := f.makeIssueInMilestone(feature, m1, "m1-issue")
		taskID := f.enqueueQueuedTask(agentID, issueID)
		mustClaim(agentID, taskID)
	})

	t.Run("second milestone gated while first is pending", func(t *testing.T) {
		feature := f.makeFeature("running")
		f.makeMilestone(feature, 0, "pending")
		m2 := f.makeMilestone(feature, 1, "pending")
		agentID := f.makeAgent("ms-gate-second-pending")
		issueID := f.makeIssueInMilestone(feature, m2, "m2-issue")
		f.enqueueQueuedTask(agentID, issueID)
		mustNotClaim(agentID)
	})

	t.Run("second milestone opens once first is validated", func(t *testing.T) {
		feature := f.makeFeature("running")
		m1 := f.makeMilestone(feature, 0, "pending")
		m2 := f.makeMilestone(feature, 1, "pending")
		agentID := f.makeAgent("ms-gate-second-passed")
		issueID := f.makeIssueInMilestone(feature, m2, "m2-issue")
		taskID := f.enqueueQueuedTask(agentID, issueID)

		mustNotClaim(agentID)
		f.passMilestone(m1)
		mustClaim(agentID, taskID)
	})
}
