package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/workspace/inplace"
)

// postWaitLocalDirectory drives the daemon endpoint that parks a task in
// waiting_local_directory and returns the decoded task response.
func postWaitLocalDirectory(t *testing.T, taskID, reason string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	req := newDaemonTokenRequest("POST", "/api/daemon/tasks/"+taskID+"/wait-local-directory",
		map[string]any{"reason": reason}, testWorkspaceID, "legit-daemon")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	testHandler.WaitTaskForLocalDirectory(w, req)

	var body map[string]any
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode wait response: %v (body=%s)", err, w.Body.String())
		}
	}
	return w, body
}

func taskStatusAndWaitReason(t *testing.T, ctx context.Context, taskID string) (string, *string) {
	t.Helper()
	var status string
	var reason *string
	if err := testPool.QueryRow(ctx,
		`SELECT status, wait_reason FROM agent_task_queue WHERE id = $1`, taskID,
	).Scan(&status, &reason); err != nil {
		t.Fatalf("read task status: %v", err)
	}
	return status, reason
}

// A dispatched task parked via the wait endpoint moves to
// waiting_local_directory and carries the reason; starting it clears both back
// to running with a null wait_reason.
func TestWaitTaskForLocalDirectory_SetsThenClearsReason(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	runtimeID := createClaimReclaimRuntime(t, ctx, "inplace-wait-runtime")
	agentID, issueID := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "inplace-wait")
	taskID := createDispatchedClaimFixtureTask(t, ctx, agentID, runtimeID, issueID, "1 second", false)

	reason := inplace.WaitReason("/code/meu-produto", "holder-task")
	w, body := postWaitLocalDirectory(t, taskID, reason)
	if w.Code != http.StatusOK {
		t.Fatalf("wait endpoint: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if body["status"] != "waiting_local_directory" {
		t.Fatalf("response status: got %v, want waiting_local_directory", body["status"])
	}
	if body["wait_reason"] != reason {
		t.Fatalf("response wait_reason: got %v, want %q", body["wait_reason"], reason)
	}

	status, dbReason := taskStatusAndWaitReason(t, ctx, taskID)
	if status != "waiting_local_directory" {
		t.Fatalf("db status after wait: got %q, want waiting_local_directory", status)
	}
	if dbReason == nil || *dbReason != reason {
		t.Fatalf("db wait_reason after wait: got %v, want %q", dbReason, reason)
	}

	startTaskForTest(t, taskID)

	status, dbReason = taskStatusAndWaitReason(t, ctx, taskID)
	if status != "running" {
		t.Fatalf("db status after start: got %q, want running", status)
	}
	if dbReason != nil {
		t.Fatalf("db wait_reason after start: got %q, want null", *dbReason)
	}
}

func startTaskForTest(t *testing.T, taskID string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := newDaemonTokenRequest("POST", "/api/daemon/tasks/"+taskID+"/start", nil,
		testWorkspaceID, "legit-daemon")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	testHandler.StartTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("start endpoint: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Two in-place tasks contend for one umbrella directory. The locker serializes
// them: the second parks in waiting_local_directory with a reason naming the
// holder, then runs once the first releases. This is the protocol that keeps
// in-place execution serial per workspace while worktree repos stay parallel.
func TestWaitLocalDirectory_SecondTaskWaitsThenRunsAfterRelease(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	runtimeID := createClaimReclaimRuntime(t, ctx, "inplace-serial-runtime")
	// Two distinct in-place tasks contend for one umbrella. They are different
	// issues (and agents): the partial unique index idx_one_pending_task_per_issue_agent
	// forbids two pending tasks for the same (issue, agent), and in production two
	// in-place tasks sharing a workspace are always different issues. The locker
	// keys on the umbrella path, not the issue, so the serialization behavior is
	// independent of which issue each task carries.
	agentA, issueA := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "inplace-serial-a")
	agentB, issueB := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "inplace-serial-b")
	taskA := createDispatchedClaimFixtureTask(t, ctx, agentA, runtimeID, issueA, "1 second", false)
	taskB := createDispatchedClaimFixtureTask(t, ctx, agentB, runtimeID, issueB, "1 second", false)

	svc := testHandler.TaskService
	locker := inplace.NewLocker()
	umbrella := t.TempDir()

	releaseA, err := locker.Acquire(ctx, umbrella, taskA, nil)
	if err != nil {
		t.Fatalf("task A acquire: %v", err)
	}
	if _, err := svc.StartTask(ctx, parseUUID(taskA)); err != nil {
		t.Fatalf("task A start: %v", err)
	}

	parked := make(chan error, 1)
	bRan := make(chan error, 1)
	go func() {
		releaseB, err := locker.Acquire(ctx, umbrella, taskB, func(holder string) {
			_, e := svc.WaitTaskForLocalDirectory(ctx, parseUUID(taskB), inplace.WaitReason(umbrella, holder))
			parked <- e
		})
		if err != nil {
			bRan <- err
			return
		}
		defer releaseB()
		_, e := svc.StartTask(ctx, parseUUID(taskB))
		bRan <- e
	}()

	select {
	case e := <-parked:
		if e != nil {
			t.Fatalf("park task B: %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for task B to park")
	}

	status, reason := taskStatusAndWaitReason(t, ctx, taskB)
	if status != "waiting_local_directory" {
		t.Fatalf("task B status while A holds: got %q, want waiting_local_directory", status)
	}
	if reason == nil || !strings.Contains(*reason, taskA) {
		t.Fatalf("task B wait_reason: got %v, want one naming holder %s", reason, taskA)
	}

	releaseA()

	select {
	case e := <-bRan:
		if e != nil {
			t.Fatalf("task B run after release: %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for task B to run")
	}

	status, reason = taskStatusAndWaitReason(t, ctx, taskB)
	if status != "running" {
		t.Fatalf("task B status after release: got %q, want running", status)
	}
	if reason != nil {
		t.Fatalf("task B wait_reason after running: got %q, want null", *reason)
	}
}
