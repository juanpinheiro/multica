package handler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ---- pure function: ClassifyPRMergeState ----

func TestClassifyPRMergeState(t *testing.T) {
	cases := []struct {
		state string
		want  PRMergeClassification
	}{
		{"merged", PRStateMerged},
		{"open", PRStateOpen},
		{"draft", PRStateDraft},
		{"closed", PRStateClosed},
		{"", PRStateUnknown},
		{"MERGED", PRStateUnknown},  // case-sensitive; must never falsely report merged
		{"unknown", PRStateUnknown}, // future enum values degrade safely
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("state=%q", tc.state), func(t *testing.T) {
			if got := ClassifyPRMergeState(tc.state); got != tc.want {
				t.Errorf("ClassifyPRMergeState(%q) = %v, want %v", tc.state, got, tc.want)
			}
		})
	}
}

// ---- fixture helpers ----

// makeMergedPR inserts a github_pull_request row with state='merged'.
func (f initiativeGateFixture) makeMergedPR(label string) string {
	f.t.Helper()
	var id string
	if err := testPool.QueryRow(f.ctx, `
		INSERT INTO github_pull_request (
			workspace_id, installation_id, repo_owner, repo_name,
			pr_number, title, state, html_url,
			pr_created_at, pr_updated_at
		) VALUES (
			$1, 0, 'owner', 'repo',
			(SELECT COALESCE(MAX(pr_number), 0) + 1 FROM github_pull_request WHERE workspace_id = $1),
			$2, 'merged', 'https://github.com/owner/repo/pull/1',
			now(), now()
		)
		RETURNING id
	`, testWorkspaceID, "PR: "+label).Scan(&id); err != nil {
		f.t.Fatalf("create merged PR (%s): %v", label, err)
	}
	f.t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM github_pull_request WHERE id = $1`, id)
	})
	return id
}

// linkIssueToPR links an issue to a PR via issue_pull_request.
func (f initiativeGateFixture) linkIssueToPR(issueID, prID string) {
	f.t.Helper()
	if _, err := testPool.Exec(f.ctx, `
		INSERT INTO issue_pull_request (issue_id, pull_request_id, close_intent)
		VALUES ($1, $2, true)
		ON CONFLICT DO NOTHING
	`, issueID, prID); err != nil {
		f.t.Fatalf("link issue %s to PR %s: %v", issueID, prID, err)
	}
	f.t.Cleanup(func() {
		testPool.Exec(context.Background(),
			`DELETE FROM issue_pull_request WHERE issue_id = $1 AND pull_request_id = $2`,
			issueID, prID)
	})
}

// ---- integration tests ----

// TestPRMergePoller_AdvancesInReviewOnMergedPR is the tracer bullet: one tick
// detects a merged PR linked to an in_review Initiative and advances it to done.
func TestPRMergePoller_AdvancesInReviewOnMergedPR(t *testing.T) {
	f := newInitiativeGateFixture(t)

	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "poll-advance")
	prID := f.makeMergedPR("poll-advance")
	f.linkIssueToPR(issueID, prID)

	poller := &PRMergePoller{queries: testHandler.Queries, handler: testHandler}
	n := poller.tick(f.ctx)

	if n == 0 {
		t.Error("tick returned 0, expected at least one candidate issue")
	}
	if got := f.featureStatus(featureID); got != "done" {
		t.Errorf("feature status = %q, want done after poll detects merged PR", got)
	}
}

// TestPRMergePoller_DeduplicatesOnDoubleObservation verifies that a second tick
// (simulating poll + webhook both observing the same merge) advances only once
// and does not error.
func TestPRMergePoller_DeduplicatesOnDoubleObservation(t *testing.T) {
	f := newInitiativeGateFixture(t)

	featureID := f.makeFeature("in_review")
	issueID := f.makeIssue(featureID, "poll-dedup")
	prID := f.makeMergedPR("poll-dedup")
	f.linkIssueToPR(issueID, prID)

	poller := &PRMergePoller{queries: testHandler.Queries, handler: testHandler}
	poller.tick(f.ctx) // first observation: advances to done
	poller.tick(f.ctx) // second observation: feature is no longer in_review; no-op

	if got := f.featureStatus(featureID); got != "done" {
		t.Errorf("feature status = %q, want done after idempotent double observation", got)
	}
}

// TestPRMergePoller_NoAdvanceForNonInReview guards that only in_review
// Initiatives are advanced — running, blocked, and ready are left alone.
func TestPRMergePoller_NoAdvanceForNonInReview(t *testing.T) {
	f := newInitiativeGateFixture(t)

	for _, status := range []string{"running", "blocked", "ready"} {
		status := status
		t.Run(status, func(t *testing.T) {
			featureID := f.makeFeature(status)
			label := fmt.Sprintf("poll-skip-%s-%d", status, time.Now().UnixNano())
			issueID := f.makeIssue(featureID, label)
			prID := f.makeMergedPR(label)
			f.linkIssueToPR(issueID, prID)

			poller := &PRMergePoller{queries: testHandler.Queries, handler: testHandler}
			poller.tick(f.ctx)

			if got := f.featureStatus(featureID); got != status {
				t.Errorf("status %s: feature changed to %q, should not advance non-in_review feature", status, got)
			}
		})
	}
}
