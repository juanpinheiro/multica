package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/multica-ai/multica/server/internal/feature"
)

func registerFeatureTools(s *Server) {
	s.mcp.AddTool(createFeatureTool(), s.handleCreateFeature)
	s.mcp.AddTool(updateFeatureTool(), s.handleUpdateFeature)
	s.mcp.AddTool(approveFeatureTool(), s.handleApproveFeature)
	s.mcp.AddTool(setFeatureStatusTool(), s.handleSetFeatureStatus)
}

// addInitiativeBudgetParams attaches the optional Mode and budget/tolerance
// fields shared by create_feature and update_feature.
func addInitiativeBudgetParams() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("mode",
			mcp.Description("Autonomy mode: 'hitl' (several reviewed PRs) or 'afk' (one autonomous PR). Defaults to hitl."),
		),
		mcp.WithNumber("budget_tokens",
			mcp.Description("Token budget cap for the Initiative. 0 (default) means no cap."),
		),
		mcp.WithNumber("budget_runs",
			mcp.Description("Run-count budget cap. 0 (default) means no cap."),
		),
		mcp.WithNumber("budget_seconds",
			mcp.Description("Wall-clock budget cap in seconds. 0 (default) means no cap."),
		),
		mcp.WithNumber("failure_tolerance",
			mcp.Description("Max repeated same-Milestone validation failures before the tripwire pauses. Defaults to 3."),
		),
	}
}

// applyInitiativeBudgetFields copies the optional Mode and budget/tolerance
// fields from the request into the API request body when they were supplied.
func applyInitiativeBudgetFields(req mcp.CallToolRequest, body map[string]any) {
	args := req.GetArguments()
	if v := req.GetString("mode", ""); v != "" {
		body["mode"] = v
	}
	for _, key := range []string{"budget_tokens", "budget_runs", "budget_seconds", "failure_tolerance"} {
		if _, ok := args[key]; ok {
			body[key] = req.GetInt(key, 0)
		}
	}
}

// ── create_feature ─────────────────────────────────────────────────────────────

func createFeatureTool() mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription("Create an Initiative (PRD). Status is always 'draft' — flip it to 'ready' with approve_feature to start the execution plane."),
		mcp.WithString("title",
			mcp.Description("Title of the Initiative."),
			mcp.Required(),
		),
		mcp.WithString("description",
			mcp.Description("Full PRD body in markdown."),
			mcp.Required(),
		),
		mcp.WithString("priority",
			mcp.Description("Priority: urgent, high, medium, low, or none."),
		),
		mcp.WithString("branch_slug",
			mcp.Description("Slug for the shared Initiative branch (e.g. 'auth-v2' → branch 'feature/auth-v2'). Omit to derive from the identifier."),
		),
		mcp.WithString("lead_id",
			mcp.Description("UUID of the agent assigned as Initiative lead."),
		),
	}
	opts = append(opts, addInitiativeBudgetParams()...)
	return mcp.NewTool("create_feature", opts...)
}

func (s *Server) handleCreateFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}
	description, err := req.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError("description is required"), nil
	}

	body := map[string]any{
		"title":       title,
		"description": description,
		"status":      "draft",
	}
	if v := req.GetString("priority", ""); v != "" {
		body["priority"] = v
	}
	if v := req.GetString("branch_slug", ""); v != "" {
		if err := feature.ValidateBranchSlug(v); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body["branch_slug"] = v
	}
	if v := req.GetString("lead_id", ""); v != "" {
		body["lead_id"] = v
	}
	applyInitiativeBudgetFields(req, body)

	var out any
	if err := s.client.PostJSON(ctx, "/api/features", body, &out); err != nil {
		return toolError("create_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── update_feature ─────────────────────────────────────────────────────────────

func updateFeatureTool() mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription("Update one or more fields of an Initiative. Only provided fields are changed. Pass branch_slug as empty string to clear it. Use set_feature_status to change status."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the Initiative to update."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("New title."),
		),
		mcp.WithString("description",
			mcp.Description("New PRD body in markdown."),
		),
		mcp.WithString("priority",
			mcp.Description("New priority: urgent, high, medium, low, or none."),
		),
		mcp.WithString("branch_slug",
			mcp.Description("New branch slug. Pass empty string to clear."),
		),
		mcp.WithString("lead_id",
			mcp.Description("UUID of the new lead agent."),
		),
	}
	opts = append(opts, addInitiativeBudgetParams()...)
	return mcp.NewTool("update_feature", opts...)
}

func (s *Server) handleUpdateFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}

	body := map[string]any{}
	args := req.GetArguments()
	for _, key := range []string{"title", "description", "priority", "lead_id"} {
		if v, ok := args[key]; ok {
			if str, _ := v.(string); str != "" {
				body[key] = str
			}
		}
	}
	// branch_slug is included when explicitly provided, even as empty string (clears the field).
	if _, ok := args["branch_slug"]; ok {
		slug := req.GetString("branch_slug", "")
		if err := feature.ValidateBranchSlug(slug); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body["branch_slug"] = slug
	}
	applyInitiativeBudgetFields(req, body)

	var out any
	if err := s.client.PatchJSON(ctx, "/api/features/"+featureID, body, &out); err != nil {
		return toolError("update_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── approve_feature ────────────────────────────────────────────────────────────

func approveFeatureTool() mcp.Tool {
	return mcp.NewTool("approve_feature",
		mcp.WithDescription("Approve an Initiative by flipping its status from 'draft' to 'ready'. This is the trigger: the execution plane claims any 'ready' Initiative."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the Initiative to approve."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleApproveFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}
	var out any
	if err := s.client.PatchJSON(ctx, "/api/features/"+featureID, map[string]any{"status": "ready"}, &out); err != nil {
		return toolError("approve_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── set_feature_status ─────────────────────────────────────────────────────────

func setFeatureStatusTool() mcp.Tool {
	return mcp.NewTool("set_feature_status",
		mcp.WithDescription("Set an Initiative's lifecycle status. Illegal transitions are rejected by the server. Use 'approve_feature' as shorthand for the draft→ready flip."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the Initiative."),
			mcp.Required(),
		),
		mcp.WithString("status",
			mcp.Description("New status: draft, ready, running, in_review, done, blocked, or cancelled."),
			mcp.Required(),
		),
	)
}

func (s *Server) handleSetFeatureStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}
	status, err := req.RequireString("status")
	if err != nil {
		return mcp.NewToolResultError("status is required"), nil
	}
	var out any
	if err := s.client.PatchJSON(ctx, "/api/features/"+featureID, map[string]any{"status": status}, &out); err != nil {
		return toolError("set_feature_status failed", err), nil
	}
	return jsonResult(out)
}
