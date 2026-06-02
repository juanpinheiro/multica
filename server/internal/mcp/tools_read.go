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
	s.mcp.AddTool(listReposTool(), s.handleListRepos)
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
		mcp.WithDescription("List Initiatives (PRDs) in the workspace. Optionally filter by status."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("status",
			mcp.Description("Filter by status: draft, ready, running, in_review, done, blocked, or cancelled."),
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

// issueAPIEntry mirrors the server's IssueSummary + BlockedIssueSummary wire format.
type issueAPIEntry struct {
	ID         string   `json:"id"`
	Identifier string   `json:"identifier"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Priority   string   `json:"priority"`
	RepoID     *string  `json:"repo_id"`
	RepoName   *string  `json:"repo_name"`
	BlockedBy  []string `json:"blocked_by"`
}

type prAPIEntry struct {
	Number  int32   `json:"number"`
	HtmlURL string  `json:"html_url"`
	State   string  `json:"state"`
	Title   string  `json:"title"`
	RepoID  *string `json:"repo_id"`
}

type featureIssuesAPIResponse struct {
	ReadyNow     []issueAPIEntry `json:"ready_now"`
	Blocked      []issueAPIEntry `json:"blocked"`
	PullRequests []prAPIEntry    `json:"pull_requests"`
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

	var issuesResp featureIssuesAPIResponse
	if err := s.client.GetJSON(ctx, "/api/features/"+featureID+"/issues", &issuesResp); err != nil {
		return toolError("get_feature issues failed", err), nil
	}

	return jsonResult(map[string]any{
		"feature":               feature,
		"issues_by_repo":        groupIssuesByRepo(issuesResp),
		"pull_requests_by_repo": groupPRsByRepo(issuesResp.PullRequests),
	})
}

func repoKey(name *string) string {
	if name == nil || *name == "" {
		return "unassigned"
	}
	return *name
}

func groupIssuesByRepo(resp featureIssuesAPIResponse) map[string]any {
	type repoGroup struct {
		ReadyNow []issueAPIEntry `json:"ready_now"`
		Blocked  []issueAPIEntry `json:"blocked"`
	}
	groups := map[string]*repoGroup{}
	ensure := func(key string) *repoGroup {
		if groups[key] == nil {
			groups[key] = &repoGroup{ReadyNow: []issueAPIEntry{}, Blocked: []issueAPIEntry{}}
		}
		return groups[key]
	}
	for _, iss := range resp.ReadyNow {
		g := ensure(repoKey(iss.RepoName))
		g.ReadyNow = append(g.ReadyNow, iss)
	}
	for _, iss := range resp.Blocked {
		g := ensure(repoKey(iss.RepoName))
		g.Blocked = append(g.Blocked, iss)
	}
	out := make(map[string]any, len(groups))
	for k, g := range groups {
		out[k] = g
	}
	return out
}

func groupPRsByRepo(prs []prAPIEntry) map[string][]prAPIEntry {
	out := map[string][]prAPIEntry{}
	for _, pr := range prs {
		key := "unassigned"
		if pr.RepoID != nil && *pr.RepoID != "" {
			key = *pr.RepoID
		}
		out[key] = append(out[key], pr)
	}
	return out
}

// ── list_repos ────────────────────────────────────────────────────────────────

func listReposTool() mcp.Tool {
	return mcp.NewTool("list_repos",
		mcp.WithDescription("List repositories in the workspace. Use this to find valid repo names and IDs for create_issue."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (s *Server) handleListRepos(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var out any
	if err := s.client.GetJSON(ctx, "/api/repos", &out); err != nil {
		return toolError("list_repos failed", err), nil
	}
	return jsonResult(out)
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
