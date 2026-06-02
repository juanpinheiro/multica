package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func registerMilestoneTools(s *Server) {
	s.mcp.AddTool(createMilestoneTool(), s.handleCreateMilestone)
	s.mcp.AddTool(updateMilestoneTool(), s.handleUpdateMilestone)
	s.mcp.AddTool(createDodAssertionTool(), s.handleCreateDodAssertion)
}

// ── create_milestone ────────────────────────────────────────────────────────

func createMilestoneTool() mcp.Tool {
	return mcp.NewTool("create_milestone",
		mcp.WithDescription("Create a Milestone (an ordered validation checkpoint) within an Initiative. Omit position to append to the end."),
		mcp.WithString("feature_id",
			mcp.Description("UUID of the parent Initiative."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("Milestone title."),
			mcp.Required(),
		),
		mcp.WithNumber("position",
			mcp.Description("Ordering position within the Initiative. Omit to append after the last Milestone."),
		),
	)
}

func (s *Server) handleCreateMilestone(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcp.NewToolResultError("feature_id is required"), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}

	body := map[string]any{
		"feature_id": featureID,
		"title":      title,
	}
	if _, ok := req.GetArguments()["position"]; ok {
		body["position"] = req.GetInt("position", 0)
	}

	var out any
	if err := s.client.PostJSON(ctx, "/api/milestones", body, &out); err != nil {
		return toolError("create_milestone failed", err), nil
	}
	return jsonResult(out)
}

// ── update_milestone ────────────────────────────────────────────────────────

func updateMilestoneTool() mcp.Tool {
	return mcp.NewTool("update_milestone",
		mcp.WithDescription("Update a Milestone's title and/or ordering position. Only provided fields change."),
		mcp.WithString("milestone_id",
			mcp.Description("UUID of the Milestone to update."),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("New title."),
		),
		mcp.WithNumber("position",
			mcp.Description("New ordering position."),
		),
	)
}

func (s *Server) handleUpdateMilestone(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	milestoneID, err := req.RequireString("milestone_id")
	if err != nil {
		return mcp.NewToolResultError("milestone_id is required"), nil
	}

	body := map[string]any{}
	if v := req.GetString("title", ""); v != "" {
		body["title"] = v
	}
	if _, ok := req.GetArguments()["position"]; ok {
		body["position"] = req.GetInt("position", 0)
	}

	var out any
	if err := s.client.PatchJSON(ctx, "/api/milestones/"+milestoneID, body, &out); err != nil {
		return toolError("update_milestone failed", err), nil
	}
	return jsonResult(out)
}

// ── create_dod_assertion ────────────────────────────────────────────────────

func createDodAssertionTool() mcp.Tool {
	return mcp.NewTool("create_dod_assertion",
		mcp.WithDescription("Write a Definition-of-Done assertion tagged to a Milestone. Validators check the Milestone's accumulated work against these assertions. Omit position to append."),
		mcp.WithString("milestone_id",
			mcp.Description("UUID of the Milestone the assertion belongs to."),
			mcp.Required(),
		),
		mcp.WithString("text",
			mcp.Description("The assertion — a checkable statement about the goal, independent of implementation."),
			mcp.Required(),
		),
		mcp.WithNumber("position",
			mcp.Description("Ordering position within the Milestone's assertions. Omit to append."),
		),
	)
}

func (s *Server) handleCreateDodAssertion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	milestoneID, err := req.RequireString("milestone_id")
	if err != nil {
		return mcp.NewToolResultError("milestone_id is required"), nil
	}
	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text is required"), nil
	}

	body := map[string]any{"text": text}
	if _, ok := req.GetArguments()["position"]; ok {
		body["position"] = req.GetInt("position", 0)
	}

	var out any
	if err := s.client.PostJSON(ctx, "/api/milestones/"+milestoneID+"/dod", body, &out); err != nil {
		return toolError("create_dod_assertion failed", err), nil
	}
	return jsonResult(out)
}
