package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func registerIssueTools(s *Server) {
	s.mcp.AddTool(createIssueTool(), s.handleCreateIssue)
	s.mcp.AddTool(updateIssueTool(), s.handleUpdateIssue)
	s.mcp.AddTool(setIssueStatusTool(), s.handleSetIssueStatus)
	s.mcp.AddTool(assignIssueTool(), s.handleAssignIssue)
	s.mcp.AddTool(commentOnIssueTool(), s.handleCommentOnIssue)
	s.mcp.AddTool(linkIssueDependencyTool(), s.handleLinkIssueDependency)
}

// ── create_issue ──────────────────────────────────────────────────────────────

func createIssueTool() mcp.Tool {
	return mcp.NewTool("create_issue",
		mcp.WithDescription("Create a new issue under a feature. feature_id is required — orphan issues are not supported via MCP."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the parent feature."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("Issue title."),
			mcp.Required(),
		),
		mcp.WithString("description",
			mcp.Description("Issue description in markdown."),
		),
		mcp.WithString("priority",
			mcp.Description("Priority: urgent, high, medium, low, or none."),
		),
		mcp.WithString("assignee_id",
			mcp.Description("UUID of the agent or member to assign."),
		),
		mcp.WithString("assignee_type",
			mcp.Description("Assignee type: agent or member."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository name or UUID to attach this issue to. Omit for a coordination issue with no code target."),
		),
	)
}

func (s *Server) handleCreateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}

	body := map[string]any{
		"title":      title,
		"feature_id": featureID,
	}
	if v := req.GetString("description", ""); v != "" {
		body["description"] = v
	}
	if v := req.GetString("priority", ""); v != "" {
		body["priority"] = v
	}
	if v := req.GetString("assignee_id", ""); v != "" {
		body["assignee_id"] = v
	}
	if v := req.GetString("assignee_type", ""); v != "" {
		body["assignee_type"] = v
	}
	if repo := req.GetString("repo", ""); repo != "" {
		repoID, err := s.resolveRepoID(ctx, repo)
		if err != nil {
			return toolError("create_issue: unknown repo", err), nil
		}
		body["repo_id"] = repoID
	}

	var out any
	if err := s.client.PostJSON(ctx, "/api/issues", body, &out); err != nil {
		return toolError("create_issue failed", err), nil
	}
	return jsonResult(out)
}

// ── update_issue ──────────────────────────────────────────────────────────────

func updateIssueTool() mcp.Tool {
	return mcp.NewTool("update_issue",
		mcp.WithDescription("Update one or more fields of an issue. Only provided non-empty fields are changed."),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue to update."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("New title."),
		),
		mcp.WithString("description",
			mcp.Description("New description in markdown."),
		),
		mcp.WithString("priority",
			mcp.Description("New priority: urgent, high, medium, low, or none."),
		),
	)
}

func (s *Server) handleUpdateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}

	body := map[string]any{}
	for _, key := range []string{"title", "description", "priority"} {
		if v := req.GetString(key, ""); v != "" {
			body[key] = v
		}
	}

	var out any
	if err := s.client.PatchJSON(ctx, "/api/issues/"+issueID, body, &out); err != nil {
		return toolError("update_issue failed", err), nil
	}
	return jsonResult(out)
}

// ── set_issue_status ──────────────────────────────────────────────────────────

func setIssueStatusTool() mcp.Tool {
	return mcp.NewTool("set_issue_status",
		mcp.WithDescription("Set the status of an issue."),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue."),
			mcp.Required(),
		),
		mcp.WithString("status",
			mcp.Description("New status: backlog, todo, in_progress, in_review, done, blocked, or cancelled."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleSetIssueStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}
	status, err := req.RequireString("status")
	if err != nil {
		return mcp.NewToolResultError("status is required"), nil
	}

	var out any
	if err := s.client.PatchJSON(ctx, "/api/issues/"+issueID, map[string]any{"status": status}, &out); err != nil {
		return toolError("set_issue_status failed", err), nil
	}
	return jsonResult(out)
}

// ── assign_issue ──────────────────────────────────────────────────────────────

func assignIssueTool() mcp.Tool {
	return mcp.NewTool("assign_issue",
		mcp.WithDescription("Assign an issue to an agent or member. Use list_agents to find valid agent IDs."),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue."),
			mcp.Required(),
		),
		mcp.WithString("assignee_id",
			mcp.Description("UUID of the agent or member."),
			mcp.Required(),
		),
		mcp.WithString("assignee_type",
			mcp.Description("Assignee type: agent or member."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleAssignIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}
	assigneeID, err := req.RequireString("assignee_id")
	if err != nil {
		return mcp.NewToolResultError("assignee_id is required"), nil
	}
	assigneeType, err := req.RequireString("assignee_type")
	if err != nil {
		return mcp.NewToolResultError("assignee_type is required"), nil
	}

	body := map[string]any{
		"assignee_id":   assigneeID,
		"assignee_type": assigneeType,
	}
	var out any
	if err := s.client.PatchJSON(ctx, "/api/issues/"+issueID, body, &out); err != nil {
		return toolError("assign_issue failed", err), nil
	}
	return jsonResult(out)
}

// ── comment_on_issue ──────────────────────────────────────────────────────────

func commentOnIssueTool() mcp.Tool {
	return mcp.NewTool("comment_on_issue",
		mcp.WithDescription("Post a comment on an issue."),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue."),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Comment text in markdown."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleCommentOnIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}
	body, err := req.RequireString("body")
	if err != nil {
		return mcp.NewToolResultError("body is required"), nil
	}

	var out any
	if err := s.client.PostJSON(ctx, "/api/issues/"+issueID+"/comments", map[string]any{"content": body}, &out); err != nil {
		return toolError("comment_on_issue failed", err), nil
	}
	return jsonResult(out)
}

// ── link_issue_dependency ─────────────────────────────────────────────────────

func linkIssueDependencyTool() mcp.Tool {
	return mcp.NewTool("link_issue_dependency",
		mcp.WithDescription("Link two issues with a dependency. 'blocks' means issue_id must complete before depends_on_issue_id starts. 'related' is non-gating."),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue that has the dependency."),
			mcp.Required(),
		),
		mcp.WithString("depends_on_issue_id",
			mcp.Description("UUID of the issue that must complete first (when type is 'blocks')."),
			mcp.Required(),
		),
		mcp.WithString("type",
			mcp.Description("Dependency type: blocks or related."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleLinkIssueDependency(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}
	dependsOnID, err := req.RequireString("depends_on_issue_id")
	if err != nil {
		return mcp.NewToolResultError("depends_on_issue_id is required"), nil
	}
	depType, err := req.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError("type is required"), nil
	}

	body := map[string]any{
		"depends_on_issue_id": dependsOnID,
		"type":                depType,
	}
	var out any
	if err := s.client.PostJSON(ctx, "/api/issues/"+issueID+"/dependencies", body, &out); err != nil {
		return toolError("link_issue_dependency failed", err), nil
	}
	return jsonResult(out)
}

// resolveRepoID looks up a repo by name or UUID within the active workspace.
// It calls GET /api/repos and matches by id or name. Returns the UUID string.
func (s *Server) resolveRepoID(ctx context.Context, nameOrID string) (string, error) {
	var repos []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := s.client.GetJSON(ctx, "/api/repos", &repos); err != nil {
		return "", err
	}
	for _, r := range repos {
		if r.ID == nameOrID || r.Name == nameOrID {
			return r.ID, nil
		}
	}
	return "", fmt.Errorf("repo %q not found in workspace", nameOrID)
}
