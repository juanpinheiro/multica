package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/multica-ai/multica/server/internal/cli"
)

func registerReadTools(s *Server) {
	s.mcp.AddTool(listFeaturesTool(), s.handleListFeatures)
	s.mcp.AddTool(getFeatureTool(), s.handleGetFeature)
	s.mcp.AddTool(listIssuesTool(), s.handleListIssues)
	s.mcp.AddTool(getIssueTool(), s.handleGetIssue)
	s.mcp.AddTool(listAgentsTool(), s.handleListAgents)
}

// toolError wraps an API error into a tool error result, surfacing the HTTP
// status code and response body verbatim so the model can see the cause.
func toolError(msg string, err error) *mcp.CallToolResult {
	var httpErr *cli.HTTPError
	if errors.As(err, &httpErr) {
		return mcp.NewToolResultError(fmt.Sprintf("%s: HTTP %d: %s", msg, httpErr.StatusCode, httpErr.Body))
	}
	return mcp.NewToolResultError(fmt.Sprintf("%s: %v", msg, err))
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// ── list_features ─────────────────────────────────────────────────────────────

func listFeaturesTool() mcp.Tool {
	return mcp.NewTool("list_features",
		mcp.WithDescription("List features (PRDs) in the workspace. Optionally filter by status."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("status",
			mcp.Description("Filter by status: planned, in_progress, paused, completed, or cancelled."),
		),
	)
}

func (s *Server) handleListFeatures(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := url.Values{}
	if status := req.GetString("status", ""); status != "" {
		params.Set("status", status)
	}
	path := "/api/features"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var out any
	if err := s.client.GetJSON(ctx, path, &out); err != nil {
		return toolError("list_features failed", err), nil
	}
	return jsonResult(out)
}

// ── get_feature ───────────────────────────────────────────────────────────────

func getFeatureTool() mcp.Tool {
	return mcp.NewTool("get_feature",
		mcp.WithDescription("Get a feature (PRD) by ID, including its child issues grouped by dependency readiness (ready_now vs blocked)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the feature."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleGetFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}

	var feature any
	if err := s.client.GetJSON(ctx, "/api/features/"+featureID, &feature); err != nil {
		return toolError("get_feature failed", err), nil
	}

	var issues any
	if err := s.client.GetJSON(ctx, "/api/features/"+featureID+"/issues", &issues); err != nil {
		return toolError("get_feature issues failed", err), nil
	}

	return jsonResult(map[string]any{"feature": feature, "issues": issues})
}

// ── list_issues ───────────────────────────────────────────────────────────────

func listIssuesTool() mcp.Tool {
	return mcp.NewTool("list_issues",
		mcp.WithDescription("List issues in the workspace. Filter by feature, status, or assignee."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("feature_id",
			mcp.Description("Filter by feature UUID (only return issues under this feature)."),
		),
		mcp.WithString("status",
			mcp.Description("Filter by status: todo, in_progress, done, cancelled."),
		),
		mcp.WithString("assignee_id",
			mcp.Description("Filter by assignee UUID (agent or member)."),
		),
	)
}

func (s *Server) handleListIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := url.Values{}
	if v := req.GetString("feature_id", ""); v != "" {
		params.Set("feature_id", v)
	}
	if v := req.GetString("status", ""); v != "" {
		params.Set("status", v)
	}
	if v := req.GetString("assignee_id", ""); v != "" {
		params.Set("assignee_id", v)
	}
	path := "/api/issues"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var out any
	if err := s.client.GetJSON(ctx, path, &out); err != nil {
		return toolError("list_issues failed", err), nil
	}
	return jsonResult(out)
}

// ── get_issue ─────────────────────────────────────────────────────────────────

func getIssueTool() mcp.Tool {
	return mcp.NewTool("get_issue",
		mcp.WithDescription("Get a single issue by ID, including full description and metadata."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("issue_id",
			mcp.Description("UUID of the issue."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleGetIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultError("issue_id is required"), nil
	}
	var out any
	if err := s.client.GetJSON(ctx, "/api/issues/"+issueID, &out); err != nil {
		return toolError("get_issue failed", err), nil
	}
	return jsonResult(out)
}

// ── list_agents ───────────────────────────────────────────────────────────────

func listAgentsTool() mcp.Tool {
	return mcp.NewTool("list_agents",
		mcp.WithDescription("List agents available in the workspace. Use this to find valid assignee IDs for create_issue or assign_issue."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (s *Server) handleListAgents(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var out any
	if err := s.client.GetJSON(ctx, "/api/agents", &out); err != nil {
		return toolError("list_agents failed", err), nil
	}
	return jsonResult(out)
}
