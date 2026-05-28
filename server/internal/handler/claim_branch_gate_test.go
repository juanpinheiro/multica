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
//   - feature.target_branch wins,
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

	makeFeature := func(t *testing.T, targetBranch *string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO feature (workspace_id, title, target_branch)
			VALUES ($1, $2, $3)
			RETURNING id
		`, testWorkspaceID, fmt.Sprintf("branch-gate-feature-%d", time.Now().UnixNano()), targetBranch).Scan(&id); err != nil {
			t.Fatalf("create feature: %v", err)
		}
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, id)
		})
		return id
	}

	makeIssue := func(t *testing.T, featureID string, metadata string) string {
		t.Helper()
		var id string
		var featureArg any
		if featureID != "" {
			featureArg = featureID
		}
		meta := "{}"
		if metadata != "" {
			meta = metadata
		}
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (
				workspace_id, feature_id, title, status, priority,
				creator_id, creator_type, number, position, metadata
			)
			VALUES (
				$1, $2, $3, 'todo', 'none', $4, 'member',
				(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1),
				0, $5::jsonb
			)
			RETURNING id
		`, testWorkspaceID, featureArg, fmt.Sprintf("branch-gate-issue-%d", time.Now().UnixNano()), testUserID, meta).Scan(&id); err != nil {
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

	t.Run("two issues sharing feature target_branch: only one claimable at a time", func(t *testing.T) {
		featureID := makeFeature(t, ptr("feature/shared-branch"))
		issueA := makeIssue(t, featureID, "")
		issueB := makeIssue(t, featureID, "")
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

	t.Run("two issues under different features (different branches): both claimable", func(t *testing.T) {
		f1 := makeFeature(t, ptr("feature/branch-one"))
		f2 := makeFeature(t, ptr("feature/branch-two"))
		issue1 := makeIssue(t, f1, "")
		issue2 := makeIssue(t, f2, "")
		agent1 := makeAgent(t, "branch-gate-distinct-1")
		agent2 := makeAgent(t, "branch-gate-distinct-2")
		task1 := enqueueQueuedTask(t, agent1, issue1)
		task2 := enqueueQueuedTask(t, agent2, issue2)

		mustClaim(t, agent1, task1)
		mustClaim(t, agent2, task2)
	})

	t.Run("issue with no feature: unaffected by gate (regression)", func(t *testing.T) {
		issueA := makeIssue(t, "", "")
		issueB := makeIssue(t, "", "")
		agentA := makeAgent(t, "branch-gate-no-feature-a")
		agentB := makeAgent(t, "branch-gate-no-feature-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustClaim(t, agentB, taskB)
	})

	t.Run("issue under feature with target_branch NULL: unaffected", func(t *testing.T) {
		featureID := makeFeature(t, nil)
		issueA := makeIssue(t, featureID, "")
		issueB := makeIssue(t, featureID, "")
		agentA := makeAgent(t, "branch-gate-null-target-a")
		agentB := makeAgent(t, "branch-gate-null-target-b")
		taskA := enqueueQueuedTask(t, agentA, issueA)
		taskB := enqueueQueuedTask(t, agentB, issueB)

		mustClaim(t, agentA, taskA)
		mustClaim(t, agentB, taskB)
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

	makeFeature := func(t *testing.T, targetBranch *string) string {
		t.Helper()
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO feature (workspace_id, title, target_branch)
			VALUES ($1, $2, $3)
			RETURNING id
		`, testWorkspaceID, fmt.Sprintf("branch-payload-feature-%d", time.Now().UnixNano()), targetBranch).Scan(&id); err != nil {
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

	t.Run("feature.target_branch set: shared branch with IsSharedBranch=true", func(t *testing.T) {
		featureID := makeFeature(t, ptr("feature/auth-v2"))
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

	t.Run("feature.target_branch NULL: derived issue branch with IsSharedBranch=false", func(t *testing.T) {
		featureID := makeFeature(t, nil)
		issueID := makeIssueWithNumber(t, featureID, 100002)
		agentID, runtimeID := makeAgent(t, "branch-payload-derived")
		enqueueQueuedTask(t, agentID, runtimeID, issueID)

		branch, shared, _ := claim(t, runtimeID)
		want := "issue/HAN-100002"
		if branch != want {
			t.Errorf("target_branch = %q, want %q", branch, want)
		}
		if shared {
			t.Errorf("is_shared_branch = true, want false")
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
