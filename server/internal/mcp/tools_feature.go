package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func registerFeatureTools(s *Server) {
	s.mcp.AddTool(createFeatureTool(), s.handleCreateFeature)
	s.mcp.AddTool(updateFeatureTool(), s.handleUpdateFeature)
	s.mcp.AddTool(approveFeatureTool(), s.handleApproveFeature)
	s.mcp.AddTool(setFeatureStatusTool(), s.handleSetFeatureStatus)
}

// ── create_feature ─────────────────────────────────────────────────────────────

func createFeatureTool() mcp.Tool {
	return mcp.NewTool("create_feature",
		mcp.WithDescription("Create a new feature (PRD). Status is always set to 'planned' — approve separately to start dispatch."),
		mcp.WithString("title",
			mcp.Description("Title of the feature."),
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
			mcp.Description("Slug for the feature branch (e.g. 'auth-v2' → branch 'feature/auth-v2'). Omit to derive from the feature identifier."),
		),
		mcp.WithString("lead_id",
			mcp.Description("UUID of the agent or member assigned as feature lead."),
		),
	)
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
		"status":      "planned",
	}
	if v := req.GetString("priority", ""); v != "" {
		body["priority"] = v
	}
	if v := req.GetString("branch_slug", ""); v != "" {
		body["branch_slug"] = v
	}
	if v := req.GetString("lead_id", ""); v != "" {
		body["lead_id"] = v
	}

	var out any
	if err := s.client.PostJSON(ctx, "/api/features", body, &out); err != nil {
		return toolError("create_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── update_feature ─────────────────────────────────────────────────────────────

func updateFeatureTool() mcp.Tool {
	return mcp.NewTool("update_feature",
		mcp.WithDescription("Update one or more fields of a feature. Only provided fields are changed. Pass branch_slug as empty string to clear it."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the feature to update."),
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
			mcp.Description("UUID of the new lead agent or member."),
		),
	)
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
		body["branch_slug"] = req.GetString("branch_slug", "")
	}

	var out any
	if err := s.client.PatchJSON(ctx, "/api/features/"+featureID, body, &out); err != nil {
		return toolError("update_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── approve_feature ────────────────────────────────────────────────────────────

func approveFeatureTool() mcp.Tool {
	return mcp.NewTool("approve_feature",
		mcp.WithDescription("Approve a feature by setting its status to 'in_progress'. This starts the dispatch motor."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the feature to approve."),
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
	if err := s.client.PatchJSON(ctx, "/api/features/"+featureID, map[string]any{"status": "in_progress"}, &out); err != nil {
		return toolError("approve_feature failed", err), nil
	}
	return jsonResult(out)
}

// ── set_feature_status ─────────────────────────────────────────────────────────

func setFeatureStatusTool() mcp.Tool {
	return mcp.NewTool("set_feature_status",
		mcp.WithDescription("Set a feature's lifecycle status. Use 'approve_feature' as shorthand for setting status to 'in_progress'."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the feature."),
			mcp.Required(),
		),
		mcp.WithString("status",
			mcp.Description("New status: planned, in_progress, paused, completed, or cancelled."),
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
