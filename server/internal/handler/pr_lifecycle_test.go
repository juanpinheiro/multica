package handler

import (
	"testing"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// makeFeatureWithBranch inserts a feature with a branch_slug, returning its id.
func (f initiativeGateFixture) makeFeatureWithBranch(status, branchSlug string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO feature (workspace_id, title, status, branch_slug)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, testWorkspaceID, "pr-lifecycle-feat-"+status, status, branchSlug).Scan(&id); err != nil {
		f.t.Fatalf("create feature(status=%s, branch=%s): %v", status, branchSlug, err)
	}
	f.t.Cleanup(func() {
		testPool.Exec(f.ctx, `DELETE FROM feature WHERE id = $1`, id)
	})
	return id
}

// countPRDraftInboxItems returns the number of feature_pr_draft inbox items for
// the given feature in the test workspace.
func (f initiativeGateFixture) countPRDraftInboxItems(featureID string) int {
	f.t.Helper()
	var n int
	if err := testPool.QueryRow(f.ctx, `
		SELECT count(*) FROM inbox_item
		WHERE workspace_id = $1
		  AND type = 'feature_pr_draft'
		  AND details->>'feature_id' = $2
	`, testWorkspaceID, featureID).Scan(&n); err != nil {
		f.t.Fatalf("count pr draft inbox items: %v", err)
	}
	return n
}

// TestInitiative_PRMerge_AdvancesInReviewToDone is the tracer bullet for the PR
// lifecycle gate: a merged PR advances an Initiative in in_review to done.
func TestInitiative_PRMerge_AdvancesInReviewToDone(t *testing.T) {
	f := newInitiativeGateFixture(t)

	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "pr-lifecycle")
	issue := f.loadIssue(issueID)

	testHandler.advanceFeaturesOnPRMerge(f.ctx, []db.Issue{issue})

	if got := f.featureStatus(featureID); got != "done" {
		t.Errorf("feature status = %q, want done after PR merge on in_review initiative", got)
	}
}

// TestInitiative_PRMerge_NoAdvanceUnlessInReview guards that the PR-merge gate
// only fires for in_review Initiatives — running/blocked/cancelled are left alone.
func TestInitiative_PRMerge_NoAdvanceUnlessInReview(t *testing.T) {
	f := newInitiativeGateFixture(t)

	for _, status := range []string{"running", "blocked", "cancelled"} {
		status := status
		t.Run(status, func(t *testing.T) {
			featureID := f.makeFeature(status)
			issueID := f.makeIssue(featureID, "no-advance-"+status)
			issue := f.loadIssue(issueID)

			testHandler.advanceFeaturesOnPRMerge(f.ctx, []db.Issue{issue})

			if got := f.featureStatus(featureID); got != status {
				t.Errorf("feature = %q, want %q (PR merge must not advance %s)", got, status, status)
			}
		})
	}
}

// TestInitiative_PRMerge_DeduplicatesFeatures verifies that two issues in the
// same feature only advance the feature once.
func TestInitiative_PRMerge_DeduplicatesFeatures(t *testing.T) {
	f := newInitiativeGateFixture(t)

	featureID := f.makeFeature("in_review")
	issueA := f.makeIssue(featureID, "dedup-a")
	issueB := f.makeIssue(featureID, "dedup-b")
	issues := []db.Issue{f.loadIssue(issueA), f.loadIssue(issueB)}

	testHandler.advanceFeaturesOnPRMerge(f.ctx, issues)

	if got := f.featureStatus(featureID); got != "done" {
		t.Errorf("feature = %q, want done", got)
	}
}

// TestInitiative_FirstClaim_NotifiesPRDraft verifies that the first task claim
// for a ready Initiative fires a feature_pr_draft inbox notification.
func TestInitiative_FirstClaim_NotifiesPRDraft(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agentID := f.makeAgent("pr-draft-agent")
	featureID := f.makeFeatureWithBranch("ready", "feat/my-pr")
	issueID := f.makeIssue(featureID, "pr-draft-issue")
	f.enqueueQueuedTask(agentID, issueID)

	task, err := testHandler.TaskService.ClaimTask(f.ctx, parseUUID(agentID))
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if task == nil {
		t.Fatal("ClaimTask returned no task")
	}

	if n := f.countPRDraftInboxItems(featureID); n != 1 {
		t.Errorf("feature_pr_draft inbox items = %d, want 1 after first claim", n)
	}
}

// TestInitiative_FirstClaim_NoPRDraftWithoutBranch verifies that initiatives
// without a branch_slug do not get a pr_draft notification.
func TestInitiative_FirstClaim_NoPRDraftWithoutBranch(t *testing.T) {
	f := newInitiativeGateFixture(t)

	agentID := f.makeAgent("pr-draft-nobranch-agent")
	featureID := f.makeFeature("ready") // no branch_slug
	issueID := f.makeIssue(featureID, "pr-draft-nobranch")
	f.enqueueQueuedTask(agentID, issueID)

	task, err := testHandler.TaskService.ClaimTask(f.ctx, parseUUID(agentID))
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if task == nil {
		t.Fatal("ClaimTask returned no task")
	}

	if n := f.countPRDraftInboxItems(featureID); n != 0 {
		t.Errorf("feature_pr_draft inbox items = %d, want 0 (no branch_slug)", n)
	}
}
