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

func TestListGroupedIssuesAssigneePaginatesPerGroup(t *testing.T) {
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	var agentOneID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'workspace', 1, $4)
		RETURNING id
	`, testWorkspaceID, fmt.Sprintf("Grouped Agent One %d", suffix), testRuntimeID, testUserID).Scan(&agentOneID); err != nil {
		t.Fatalf("create agent one: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentOneID)
	})

	var agentTwoID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'workspace', 1, $4)
		RETURNING id
	`, testWorkspaceID, fmt.Sprintf("Grouped Agent Two %d", suffix), testRuntimeID, testUserID).Scan(&agentTwoID); err != nil {
		t.Fatalf("create agent two: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentTwoID)
	})

	createIssue := func(title, assigneeType, assigneeID string, position float64) string {
		t.Helper()
		var number int32
		if err := testPool.QueryRow(ctx, `
			UPDATE workspace
			SET issue_counter = GREATEST(
				issue_counter,
				(SELECT COALESCE(MAX(number), 0) FROM issue WHERE workspace_id = $1)
			) + 1
			WHERE id = $1
			RETURNING issue_counter
		`, testWorkspaceID).Scan(&number); err != nil {
			t.Fatalf("next issue number: %v", err)
		}

		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (
				workspace_id, title, description, status, priority,
				assignee_type, assignee_id, creator_type, creator_id,
				position, number
			)
			VALUES ($1, $2, NULL, 'todo', 'none', $3, $4, 'member', $5, $6, $7)
			RETURNING id
		`, testWorkspaceID, title, assigneeType, assigneeID, testUserID, position, number).Scan(&id); err != nil {
			t.Fatalf("create issue %q: %v", title, err)
		}
		t.Cleanup(func() {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, id)
		})
		return id
	}

	createIssue("Grouped agent-one one", "agent", agentOneID, 1)
	createIssue("Grouped agent-one two", "agent", agentOneID, 2)
	createIssue("Grouped agent-one three", "agent", agentOneID, 3)
	createIssue("Grouped agent-two one", "agent", agentTwoID, 1)

	path := fmt.Sprintf(
		"/api/issues/grouped?workspace_id=%s&group_by=assignee&statuses=todo&limit=2&assignee_filters=agent:%s,agent:%s",
		testWorkspaceID,
		agentOneID,
		agentTwoID,
	)
	w := httptest.NewRecorder()
	testHandler.ListGroupedIssues(w, newRequest("GET", path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListGroupedIssues: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp GroupedIssuesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode grouped response: %v", err)
	}

	agentOneGroupID := "assignee:agent:" + agentOneID
	agentTwoGroupID := "assignee:agent:" + agentTwoID
	groups := map[string]IssueAssigneeGroupResponse{}
	for _, group := range resp.Groups {
		groups[group.ID] = group
	}

	agentOneGroup, ok := groups[agentOneGroupID]
	if !ok {
		t.Fatalf("missing agent-one group %s in %#v", agentOneGroupID, resp.Groups)
	}
	if agentOneGroup.Total != 3 || len(agentOneGroup.Issues) != 2 {
		t.Fatalf("agent-one group total/page mismatch: total=%d len=%d", agentOneGroup.Total, len(agentOneGroup.Issues))
	}
	if agentOneGroup.Issues[0].Title != "Grouped agent-one one" || agentOneGroup.Issues[1].Title != "Grouped agent-one two" {
		t.Fatalf("agent-one group order mismatch: %#v", agentOneGroup.Issues)
	}

	agentTwoGroup, ok := groups[agentTwoGroupID]
	if !ok {
		t.Fatalf("missing agent-two group %s in %#v", agentTwoGroupID, resp.Groups)
	}
	if agentTwoGroup.Total != 1 || len(agentTwoGroup.Issues) != 1 {
		t.Fatalf("agent-two group total/page mismatch: total=%d len=%d", agentTwoGroup.Total, len(agentTwoGroup.Issues))
	}

	nextPath := fmt.Sprintf(
		"/api/issues/grouped?workspace_id=%s&group_by=assignee&statuses=todo&limit=2&offset=2&group_assignee_type=agent&group_assignee_id=%s",
		testWorkspaceID,
		agentOneID,
	)
	next := httptest.NewRecorder()
	testHandler.ListGroupedIssues(next, newRequest("GET", nextPath, nil))
	if next.Code != http.StatusOK {
		t.Fatalf("ListGroupedIssues next page: expected 200, got %d: %s", next.Code, next.Body.String())
	}

	var nextResp GroupedIssuesResponse
	if err := json.NewDecoder(next.Body).Decode(&nextResp); err != nil {
		t.Fatalf("decode next grouped response: %v", err)
	}
	if len(nextResp.Groups) != 1 {
		t.Fatalf("expected one next-page group, got %#v", nextResp.Groups)
	}
	if nextResp.Groups[0].ID != agentOneGroupID || nextResp.Groups[0].Total != 3 || len(nextResp.Groups[0].Issues) != 1 {
		t.Fatalf("unexpected next-page group: %#v", nextResp.Groups[0])
	}
	if nextResp.Groups[0].Issues[0].Title != "Grouped agent-one three" {
		t.Fatalf("unexpected next-page issue: %#v", nextResp.Groups[0].Issues[0])
	}
}
