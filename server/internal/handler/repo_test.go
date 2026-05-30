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

// createTestRepo posts a repo to the active test workspace and returns the
// decoded response, failing the test on a non-201.
func createTestRepo(t *testing.T, name, remoteURL string) RepoResponse {
	t.Helper()
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/repos", map[string]any{
		"name":       name,
		"remote_url": remoteURL,
	})
	testHandler.CreateRepo(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateRepo: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var repo RepoResponse
	if err := json.NewDecoder(w.Body).Decode(&repo); err != nil {
		t.Fatalf("decode repo: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM repo WHERE id = $1`, repo.ID)
	})
	return repo
}

func TestRepoCRUDLifecycle(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	suffix := time.Now().UnixNano()
	name := fmt.Sprintf("backend-%d", suffix)
	remote := fmt.Sprintf("github.com/team/backend-%d", suffix)

	repo := createTestRepo(t, name, remote)
	if repo.Name != name || repo.RemoteURL != remote {
		t.Fatalf("unexpected repo: %+v", repo)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("default_branch = %q, want main", repo.DefaultBranch)
	}

	// List includes the new repo.
	w := httptest.NewRecorder()
	testHandler.ListRepos(w, newRequest("GET", "/api/repos", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRepos: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []RepoResponse
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !containsRepoID(list, repo.ID) {
		t.Fatalf("list did not contain created repo %s", repo.ID)
	}

	// Get by id.
	w = httptest.NewRecorder()
	req := withURLParam(newRequest("GET", "/api/repos/"+repo.ID, nil), "id", repo.ID)
	testHandler.GetRepo(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetRepo: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete.
	w = httptest.NewRecorder()
	req = withURLParam(newRequest("DELETE", "/api/repos/"+repo.ID, nil), "id", repo.ID)
	testHandler.DeleteRepo(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteRepo: expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRepoUnknownIDReturns404(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	// A syntactically valid UUID absent from this workspace must 404, not
	// silently 204 (the #1661 zero-rows trap the DeleteRepo :execrows guard fixes).
	const unknownID = "99999999-9999-9999-9999-999999999999"
	w := httptest.NewRecorder()
	req := withURLParam(newRequest("DELETE", "/api/repos/"+unknownID, nil), "id", unknownID)
	testHandler.DeleteRepo(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("DeleteRepo unknown id: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRepoRejectsDuplicates(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	suffix := time.Now().UnixNano()
	name := fmt.Sprintf("dup-name-%d", suffix)
	remote := fmt.Sprintf("github.com/team/dup-%d", suffix)
	createTestRepo(t, name, remote)

	// Same name, different remote → 409.
	w := httptest.NewRecorder()
	testHandler.CreateRepo(w, newRequest("POST", "/api/repos", map[string]any{
		"name":       name,
		"remote_url": remote + "-other",
	}))
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate name: expected 409, got %d: %s", w.Code, w.Body.String())
	}

	// Same remote, different name → 409.
	w = httptest.NewRecorder()
	testHandler.CreateRepo(w, newRequest("POST", "/api/repos", map[string]any{
		"name":       name + "-other",
		"remote_url": remote,
	}))
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate remote_url: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReposIsWorkspaceScoped(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	// A repo in a different workspace must not leak into this workspace's list.
	var otherWsID string
	suffix := time.Now().UnixNano()
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, issue_prefix)
		VALUES ($1, $2, 'OTH') RETURNING id
	`, "Other WS", fmt.Sprintf("other-ws-%d", suffix)).Scan(&otherWsID); err != nil {
		t.Fatalf("create other workspace: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, otherWsID) })

	var otherRepoID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO repo (workspace_id, name, remote_url)
		VALUES ($1, 'foreign', $2) RETURNING id
	`, otherWsID, fmt.Sprintf("github.com/team/foreign-%d", suffix)).Scan(&otherRepoID); err != nil {
		t.Fatalf("create foreign repo: %v", err)
	}

	w := httptest.NewRecorder()
	testHandler.ListRepos(w, newRequest("GET", "/api/repos", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRepos: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []RepoResponse
	json.NewDecoder(w.Body).Decode(&list)
	if containsRepoID(list, otherRepoID) {
		t.Fatalf("list leaked a repo from another workspace")
	}
}

func TestIssueRepoAttachment(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}

	suffix := time.Now().UnixNano()
	repo := createTestRepo(t, fmt.Sprintf("attach-%d", suffix), fmt.Sprintf("github.com/team/attach-%d", suffix))

	// Create an issue tagged with the repo; it should persist and round-trip.
	w := httptest.NewRecorder()
	testHandler.CreateIssue(w, newRequest("POST", "/api/issues", map[string]any{
		"title":   "repo-tagged issue",
		"status":  "todo",
		"repo_id": repo.ID,
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateIssue: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var issue IssueResponse
	json.NewDecoder(w.Body).Decode(&issue)
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issue.ID) })
	if issue.RepoID == nil || *issue.RepoID != repo.ID {
		t.Fatalf("issue.repo_id = %v, want %s", issue.RepoID, repo.ID)
	}

	// Read back returns the repo_id.
	w = httptest.NewRecorder()
	req := withURLParam(newRequest("GET", "/api/issues/"+issue.ID, nil), "id", issue.ID)
	testHandler.GetIssue(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetIssue: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got IssueResponse
	json.NewDecoder(w.Body).Decode(&got)
	if got.RepoID == nil || *got.RepoID != repo.ID {
		t.Fatalf("read-back issue.repo_id = %v, want %s", got.RepoID, repo.ID)
	}
}

func TestCreateIssueRejectsForeignRepo(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var otherWsID, foreignRepoID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, issue_prefix)
		VALUES ($1, $2, 'OTH') RETURNING id
	`, "Other WS2", fmt.Sprintf("other-ws2-%d", suffix)).Scan(&otherWsID); err != nil {
		t.Fatalf("create other workspace: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, otherWsID) })
	if err := testPool.QueryRow(ctx, `
		INSERT INTO repo (workspace_id, name, remote_url)
		VALUES ($1, 'foreign2', $2) RETURNING id
	`, otherWsID, fmt.Sprintf("github.com/team/foreign2-%d", suffix)).Scan(&foreignRepoID); err != nil {
		t.Fatalf("create foreign repo: %v", err)
	}

	w := httptest.NewRecorder()
	testHandler.CreateIssue(w, newRequest("POST", "/api/issues", map[string]any{
		"title":   "cross-workspace repo issue",
		"status":  "todo",
		"repo_id": foreignRepoID,
	}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("cross-workspace repo_id: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func containsRepoID(repos []RepoResponse, id string) bool {
	for _, r := range repos {
		if r.ID == id {
			return true
		}
	}
	return false
}
