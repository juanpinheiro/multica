package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TestClaimAgentTask_BranchGate verifies that ClaimAgentTask refuses to
// dispatch a queued task whose resolved branch matches a currently-dispatched
// task's resolved branch. The resolved branch mirrors feature.Resolve:
//
//   - feature.branch_slug (as 'feature/<slug>') wins,
//   - else issue.metadata->>'target_branch',
//   - else 'issue/<prefix>-<number>'.
//
// The test calls Queries.ClaimAgentTask directly to isolate the SQL claim
// behavior from per-agent capacity / runtime routing. Each scenario uses its
// own pair of issues + agents so the per-(issue, agent) serialization gate
// never confounds the branch-gate assertion.
func TestClaimAgentTask_BranchGate(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	queries := db.New(testPool)

	makeAgent := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent (
				workspace_id, name, description, runtime_mode, runtime_config,
				runtime_id, visibility, max_concurrent_tasks, owner_id
			)
			VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4)
			RETURNING id
		`, testWorkspaceID, name, handlerTestRuntimeID(t), testUserID).Scan(&id); err != nil {
			t.Fatalf("create agent: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, id)
		})
		return id
	}

	makeFeature := func(t *testing.T, branchSlug *string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO feature (workspace_id, title, branch_slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, testWorkspaceID, fmt.Sprintf("branch-gate-feature-%d", time.Now().UnixNano()), branchSlug).Scan(&id); err != nil {
			t.Fatalf("create feature: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id)
		})
		return id
	}

	makeRepo := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO repo (workspace_id, name, remote_url)
			VALUES ($1, $2, $3)
			RETURNING id
		`, testWorkspaceID, name, fmt.Sprintf("git@example.com/%s-%d.git", name, time.Now().UnixNano())).Scan(&id); err != nil {
			t.Fatalf("create repo: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM repo WHERE id = $1`, id)
		})
		return id
	}

	makeIssue := func(t *testing.T, featureID, repoID, metadata string) string {
		t.Helper()
		var id string
		var featureArg, repoArg any
		if featureID != "" {
			featureArg = featureID
		}
		if repoID != "" {
			repoArg = repoID
		}
		meta := "{}"
		if metadata != "" {
			meta = metadata
		}
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (
				workspace_id, feature_id, repo_id, title, status, priority,
				creator_id, creator_type, number, position, metadata
			)
			VALUES (
				$1, $2, $3, $4, 'todo', 'none', $5, 'member',
				(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
				0, $6::jsonb
			)
			RETURNING id
		`, testWorkspaceID, featureArg, repoArg, fmt.Sprintf("branch-gate-issue-%d", time.Now().UnixNano()), testUserID, meta).Scan(&id); err != nil {
			t.Fatalf("create issue: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
		})
		return id
	}

	enqueueQueuedTask := func(t *testing.T, agentID, issueID string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (
				agent_id, runtime_id, issue_id, status, priority
			) VALUES ($1, $2, $3, 'queued', 0)
			RETURNING id
		`, agentID, handlerTestRuntimeID(t), issueID).Scan(&id); err != nil {
			t.Fatalf("enqueue task: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id)
		})
		return id
	}

	mustClaim := func(t *testing.T, agentID, wantTaskID string) {
		t.Helper()
		row, err := queries.ClaimAgentTask(ctx, parseUUID(agentID))
		if err != nil {
			t.Fatalf("expected claim, got error: %v", err)
		}
		if got := uuidToString(row.ID); got != wantTaskID {
			t.Fatalf("claimed task id = %s, want %s", got, wantTaskID)
		}
		if row.Status != "dispatched" {
			t.Errorf("claimed task status = %q, want dispatched", row.Status)
		}
	}

	mustNotClaim := func(t *testing.T, agentID string) {
		t.Helper()
		row, err := queries.ClaimAgentTask(ctx, parseUUID(agentID))
		if err == nil {
			t.Fatalf("expected no claim, got task %s status=%s", row.ID, row.Status)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows, got %v", err)
		}
	}

	ptr := func(s string) *string { return &s }

	t.Run("two issues sharing feature branch_slug in the same repo: only one claimable at a time", func(t *testing.T) {
		featureID := makeFeature(t, ptr("shared-branch"))
		repoID := makeRepo(t, "shared-repo")
		issueA := makeIssue(t, featureID, repoID, "")
		issueB := makeIssue(t, featureID, repoID, "")
		agentA := makeAgent(t, "branch-gate-shared-a")
		agentB := makeAgent(t, "branch-gate-shared-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustNotClaim(t, agentB)

		if _, err := testPool.Exec(ctx, `
			UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id = $1
		`, taskA); err != nil {
			t.Fatalf("complete taskA: %v", err)
		}
		mustClaim(t, agentB, taskB)
	})

	t.Run("shared branch stays serialized while the first task is RUNNING (not just dispatched)", func(t *testing.T) {
		// Regression: the gate once checked only status='dispatched'. Once the
		// first task transitions dispatched→running (the long execution phase),
		// a second slice on the same (repo, feature branch) must still be blocked
		// — otherwise two agents push to the same branch concurrently.
		featureID := makeFeature(t, ptr("running-shared"))
		repoID := makeRepo(t, "running-repo")
		issueA := makeIssue(t, featureID, repoID, "")
		issueB := makeIssue(t, featureID, repoID, "")
		agentA := makeAgent(t, "branch-gate-running-a")
		agentB := makeAgent(t, "branch-gate-running-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		if _, err := testPool.Exec(ctx, `
			UPDATE agent_task_queue SET status = 'running' WHERE id = $1
		`, taskA); err != nil {
			t.Fatalf("set taskA running: %v", err)
		}
		mustNotClaim(t, agentB)

		if _, err := testPool.Exec(ctx, `
			UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id = $1
		`, taskA); err != nil {
			t.Fatalf("complete taskA: %v", err)
		}
		mustClaim(t, agentB, taskB)
	})

	t.Run("same feature, same branch name, different repos: both claimable in parallel", func(t *testing.T) {
		// The decisive cross-repo case: a single feature spans two repos. The
		// branch name (feature/<slug>) is identical in both repos, but the
		// (repo, branch) gate keys on repo too, so backend and frontend run
		// concurrently instead of being serialized against each other.
		featureID := makeFeature(t, ptr("auth-v2"))
		backend := makeRepo(t, "backend")
		frontend := makeRepo(t, "frontend")
		issueA := makeIssue(t, featureID, backend, "")
		issueB := makeIssue(t, featureID, frontend, "")
		agentA := makeAgent(t, "branch-gate-xrepo-a")
		agentB := makeAgent(t, "branch-gate-xrepo-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustClaim(t, agentB, taskB)
	})

	t.Run("two issues under different features (different branches): both claimable", func(t *testing.T) {
		f1 := makeFeature(t, ptr("branch-one"))
		f2 := makeFeature(t, ptr("branch-two"))
		repoID := makeRepo(t, "distinct-repo")
		issue1 := makeIssue(t, f1, repoID, "")
		issue2 := makeIssue(t, f2, repoID, "")
		agent1 := makeAgent(t, "branch-gate-distinct-1")
		agent2 := makeAgent(t, "branch-gate-distinct-2")
		task1 := enqueueQueuedTask(t, agent1, issue1)
		task2 := enqueueQueuedTask(t, agent2, issue2)

		mustClaim(t, agent1, task1)
		mustClaim(t, agent2, task2)
	})

	t.Run("issue with no feature: unaffected by gate (regression)", func(t *testing.T) {
		issueA := makeIssue(t, "", "", "")
		issueB := makeIssue(t, "", "", "")
		agentA := makeAgent(t, "branch-gate-no-feature-a")
		agentB := makeAgent(t, "branch-gate-no-feature-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustClaim(t, agentB, taskB)
	})

	t.Run("issues sharing feature branch but repo_id NULL: both claimable (exempt from gate)", func(t *testing.T) {
		// repo_id IS NULL means the issue targets no code and holds no branch,
		// so it is exempt from the (repo, branch) serialization gate even when
		// it shares a feature branch name with another queued issue.
		featureID := makeFeature(t, ptr("no-repo-branch"))
		issueA := makeIssue(t, featureID, "", "")
		issueB := makeIssue(t, featureID, "", "")
		agentA := makeAgent(t, "branch-gate-norepo-a")
		agentB := makeAgent(t, "branch-gate-norepo-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustClaim(t, agentB, taskB)
	})

	t.Run("issues under same feature and repo with branch_slug NULL: serialized on feature/<id> branch", func(t *testing.T) {
		// Even without an explicit branch_slug, a present feature causes all its
		// issues to converge on "feature/<featureUUID>", so two issues in the
		// same repo serialize.
		featureID := makeFeature(t, nil)
		repoID := makeRepo(t, "null-slug-repo")
		issueA := makeIssue(t, featureID, repoID, "")
		issueB := makeIssue(t, featureID, repoID, "")
		agentA := makeAgent(t, "branch-gate-null-slug-a")
		agentB := makeAgent(t, "branch-gate-null-slug-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustNotClaim(t, agentB)

		if _, err := testPool.Exec(ctx, `
			UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id = $1
		`, taskA); err != nil {
			t.Fatalf("complete taskA: %v", err)
		}
		mustClaim(t, agentB, taskB)
	})

	t.Run("cross-repo dependency: blocked issue not claimed until blocker done", func(t *testing.T) {
		// A frontend issue blocked_by a backend issue (different repos) must not
		// be claimed until the backend issue is done — Gate 1 spans repos.
		featureID := makeFeature(t, ptr("xrepo-dep"))
		backend := makeRepo(t, "dep-backend")
		frontend := makeRepo(t, "dep-frontend")
		backendIssue := makeIssue(t, featureID, backend, "")
		frontendIssue := makeIssue(t, featureID, frontend, "")
		if _, err := testPool.Exec(ctx, `
			INSERT INTO issue_dependency (issue_id, depends_on_issue_id, type)
			VALUES ($1, $2, 'blocked_by')
		`, frontendIssue, backendIssue); err != nil {
			t.Fatalf("link dependency: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue_dependency WHERE issue_id = $1`, frontendIssue)
		})

		agentBackend := makeAgent(t, "branch-gate-dep-backend")
		agentFrontend := makeAgent(t, "branch-gate-dep-frontend")
		backendTask := enqueueQueuedTask(t, agentBackend, backendIssue)
		frontendTask := enqueueQueuedTask(t, agentFrontend, frontendIssue)

		mustNotClaim(t, agentFrontend)
		mustClaim(t, agentBackend, backendTask)

		if _, err := testPool.Exec(ctx, `
			UPDATE issue SET status = 'done' WHERE id = $1
		`, backendIssue); err != nil {
			t.Fatalf("mark backend done: %v", err)
		}
		mustClaim(t, agentFrontend, frontendTask)
	})
}

// TestClaimTaskByRuntime_BranchPayload verifies that ClaimTaskByRuntime
// populates TargetBranch and IsSharedBranch on the response by running
// feature.Resolve(issue, feature) on the claim path.
func TestClaimTaskByRuntime_BranchPayload(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	makeFeature := func(t *testing.T, branchSlug *string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO feature (workspace_id, title, branch_slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, testWorkspaceID, fmt.Sprintf("branch-payload-feature-%d", time.Now().UnixNano()), branchSlug).Scan(&id); err != nil {
			t.Fatalf("create feature: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id)
		})
		return id
	}

	makeIssueWithNumber := func(t *testing.T, featureID string, number int32) string {
		t.Helper()
		var id string
		var featureArg any
		if featureID != "" {
			featureArg = featureID
		}
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (
				workspace_id, feature_id, title, status, priority,
				creator_id, creator_type, number, position
			)
			VALUES (
				$1, $2, $3, 'todo', 'none', $4, 'member', $5, 0
			)
			RETURNING id
		`, testWorkspaceID, featureArg, fmt.Sprintf("branch-payload-issue-%d", time.Now().UnixNano()), testUserID, number).Scan(&id); err != nil {
			t.Fatalf("create issue: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
		})
		return id
	}

	makeAgent := func(t *testing.T, name string) (string, string) {
		t.Helper()
		runtimeID := handlerTestRuntimeID(t)
		var agentID string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent (
				workspace_id, name, description, runtime_mode, runtime_config,
				runtime_id, visibility, max_concurrent_tasks, owner_id
			)
			VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4)
			RETURNING id
		`, testWorkspaceID, name, runtimeID, testUserID).Scan(&agentID); err != nil {
			t.Fatalf("create agent: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentID)
		})
		return agentID, runtimeID
	}

	enqueueQueuedTask := func(t *testing.T, agentID, runtimeID, issueID string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (
				agent_id, runtime_id, issue_id, status, priority
			) VALUES ($1, $2, $3, 'queued', 0)
			RETURNING id
		`, agentID, runtimeID, issueID).Scan(&id); err != nil {
			t.Fatalf("enqueue task: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, id)
		})
		return id
	}

	claim := func(t *testing.T, runtimeID string) (string, bool, string) {
		t.Helper()
		w := httptest.NewRecorder()
		req := newDaemonTokenRequest("POST", "/api/daemon/runtimes/"+runtimeID+"/claim", nil, testWorkspaceID, "branch-payload-claim")
		req = withURLParam(req, "runtimeId", runtimeID)
		testHandler.ClaimTaskByRuntime(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("ClaimTaskByRuntime: %d %s", w.Code, w.Body.String())
		}
		var resp struct {
			Task *struct {
				ID             string `json:"id"`
				TargetBranch   string `json:"target_branch"`
				IsSharedBranch bool   `json:"is_shared_branch"`
			} `json:"task"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Task == nil {
			t.Fatal("expected task in response")
		}
		return resp.Task.TargetBranch, resp.Task.IsSharedBranch, resp.Task.ID
	}

	ptr := func(s string) *string { return &s }

	t.Run("feature.branch_slug set: shared branch with IsSharedBranch=true", func(t *testing.T) {
		featureID := makeFeature(t, ptr("auth-v2"))
		issueID := makeIssueWithNumber(t, featureID, 100001)
		agentID, runtimeID := makeAgent(t, "branch-payload-shared")
		enqueueQueuedTask(t, agentID, runtimeID, issueID)

		branch, shared, _ := claim(t, runtimeID)
		if branch != "feature/auth-v2" {
			t.Errorf("target_branch = %q, want feature/auth-v2", branch)
		}
		if !shared {
			t.Errorf("is_shared_branch = false, want true")
		}
	})

	t.Run("feature.branch_slug NULL: shared feature/<UUID> branch with IsSharedBranch=true", func(t *testing.T) {
		// Feature present but no branch_slug → feature.<featureUUID>, shared=true.
		featureID := makeFeature(t, nil)
		issueID := makeIssueWithNumber(t, featureID, 100002)
		agentID, runtimeID := makeAgent(t, "branch-payload-derived")
		enqueueQueuedTask(t, agentID, runtimeID, issueID)

		branch, shared, _ := claim(t, runtimeID)
		want := "feature/" + featureID
		if branch != want {
			t.Errorf("target_branch = %q, want %q", branch, want)
		}
		if !shared {
			t.Errorf("is_shared_branch = false, want true")
		}
	})

	t.Run("no feature, issue.metadata target_branch set: per-issue branch with IsSharedBranch=false", func(t *testing.T) {
		var issueID string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (
				workspace_id, title, status, priority, creator_id, creator_type,
				number, position, metadata
			)
			VALUES (
				$1, $2, 'todo', 'none', $3, 'member', 100003, 0,
				'{"target_branch":"issue/per-issue-override"}'::jsonb
			)
			RETURNING id
		`, testWorkspaceID, "branch-payload-per-issue", testUserID).Scan(&issueID); err != nil {
			t.Fatalf("create issue: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
		})

		agentID, runtimeID := makeAgent(t, "branch-payload-per-issue")
		enqueueQueuedTask(t, agentID, runtimeID, issueID)

		branch, shared, _ := claim(t, runtimeID)
		if branch != "issue/per-issue-override" {
			t.Errorf("target_branch = %q, want issue/per-issue-override", branch)
		}
		if shared {
			t.Errorf("is_shared_branch = true, want false")
		}
	})
}
