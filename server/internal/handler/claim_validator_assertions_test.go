package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestClaimTaskByRuntime_ValidatorAssertionsInjected verifies that claiming a
// validator task whose issue belongs to a Milestone with DoD assertions returns
// those assertions in the response's validator_assertions field.
func TestClaimTaskByRuntime_ValidatorAssertionsInjected(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	f := newHandoffFixture(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	agentID := f.makeAgent("val-claim-agent-" + suffix)

	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "val-claim-issue")

	a1 := f.makeDodAssertion(featureID, milestoneID, "all tests pass", 0)
	a2 := f.makeDodAssertion(featureID, milestoneID, "no lint errors", 1)

	runtimeID := handlerTestRuntimeID(t)

	// Insert a queued validator task.
	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'queued', 0, 'validator')
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("create validator task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	// Call ClaimTaskByRuntime via HTTP.
	w := httptest.NewRecorder()
	req := newDaemonTokenRequest("POST", "/api/daemon/runtimes/"+runtimeID+"/tasks/claim", nil,
		testWorkspaceID, "val-claim-test")
	req = withURLParam(req, "runtimeId", runtimeID)
	testHandler.ClaimTaskByRuntime(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ClaimTaskByRuntime returned %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Task *struct {
			ID                  string `json:"id"`
			Role                string `json:"role"`
			ValidatorAssertions []struct {
				ID       string `json:"id"`
				Text     string `json:"text"`
				Position int    `json:"position"`
			} `json:"validator_assertions"`
		} `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task == nil {
		t.Fatalf("expected a task in the response, got nil; body: %s", w.Body.String())
	}
	if resp.Task.ID != taskID {
		t.Errorf("task id = %s, want %s", resp.Task.ID, taskID)
	}
	if resp.Task.Role != taskRoleValidator {
		t.Errorf("role = %q, want %q", resp.Task.Role, taskRoleValidator)
	}
	if len(resp.Task.ValidatorAssertions) != 2 {
		t.Fatalf("validator_assertions count = %d, want 2; body: %s", len(resp.Task.ValidatorAssertions), w.Body.String())
	}

	byID := make(map[string]string, 2)
	for _, a := range resp.Task.ValidatorAssertions {
		byID[a.ID] = a.Text
	}

	if byID[a1] != "all tests pass" {
		t.Errorf("assertion %s text = %q, want \"all tests pass\"", a1, byID[a1])
	}
	if byID[a2] != "no lint errors" {
		t.Errorf("assertion %s text = %q, want \"no lint errors\"", a2, byID[a2])
	}
}

// TestClaimTaskByRuntime_WorkerNoValidatorAssertions verifies that worker (non-
// validator) claims never include validator_assertions.
func TestClaimTaskByRuntime_WorkerNoValidatorAssertions(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	f := newHandoffFixture(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	agentID := f.makeAgent("worker-claim-agent-" + suffix)

	featureID := f.makeFeature("running")
	milestoneID := f.makeMilestone(featureID, 0, "pending")
	issueID := f.makeIssueInMilestone(featureID, milestoneID, "worker-claim-issue")
	f.makeDodAssertion(featureID, milestoneID, "tests pass", 0)

	runtimeID := handlerTestRuntimeID(t)

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, role)
		VALUES ($1, $2, $3, 'queued', 0, 'worker')
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("create worker task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	w := httptest.NewRecorder()
	req := newDaemonTokenRequest("POST", "/api/daemon/runtimes/"+runtimeID+"/tasks/claim", nil,
		testWorkspaceID, "worker-claim-test")
	req = withURLParam(req, "runtimeId", runtimeID)
	testHandler.ClaimTaskByRuntime(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ClaimTaskByRuntime returned %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Task *struct {
			ValidatorAssertions []struct{} `json:"validator_assertions"`
		} `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task == nil {
		t.Fatalf("expected a task in the response, got nil; body: %s", w.Body.String())
	}
	if len(resp.Task.ValidatorAssertions) != 0 {
		t.Errorf("worker task has validator_assertions = %d, want 0", len(resp.Task.ValidatorAssertions))
	}
}
