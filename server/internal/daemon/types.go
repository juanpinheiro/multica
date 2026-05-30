package daemon

import "encoding/json"

// AgentEntry describes a single available agent CLI.
type AgentEntry struct {
	Path  string // path to CLI binary
	Model string // model override (optional)
}

// Runtime represents a registered daemon runtime.
type Runtime struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

// RepoData holds repository information from the workspace.
type RepoData struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// FeatureResourceData mirrors handler.FeatureResourceData — a single project
// resource as delivered to the daemon. resource_ref is type-specific JSON.
type FeatureResourceData struct {
	ID           string          `json:"id"`
	ResourceType string          `json:"resource_type"`
	ResourceRef  json.RawMessage `json:"resource_ref"`
	Label        string          `json:"label,omitempty"`
}

// Task represents a claimed task from the server.
// Agent data (name, skills) is populated by the claim endpoint.
type Task struct {
	ID          string `json:"id"`
	AgentID     string `json:"agent_id"`
	RuntimeID   string `json:"runtime_id"`
	IssueID     string `json:"issue_id"`
	WorkspaceID string `json:"workspace_id"`
	// WorkspaceContext mirrors workspace.context (the per-workspace system
	// prompt set in Settings → General). Server populates this on every claim
	// regardless of task kind so the daemon can inject `## Workspace Context`
	// into the brief. Empty when the owner hasn't set one.
	WorkspaceContext        string                `json:"workspace_context,omitempty"`
	Agent                   *AgentData            `json:"agent,omitempty"`
	Repos                   []RepoData            `json:"repos,omitempty"`
	FeatureID               string                `json:"feature_id,omitempty"`                // issue's feature, when present
	FeatureTitle            string                `json:"feature_title,omitempty"`             // human-readable feature title for context injection
	FeatureResources        []FeatureResourceData `json:"feature_resources,omitempty"`         // feature-scoped resources to expose to the agent
	PriorSessionID          string                `json:"prior_session_id,omitempty"`          // Claude session ID from a previous task on this issue
	PriorWorkDir            string                `json:"prior_work_dir,omitempty"`            // work_dir from a previous task on this issue
	TriggerCommentID        string                `json:"trigger_comment_id,omitempty"`        // comment that triggered this task
	TriggerCommentContent   string                `json:"trigger_comment_content,omitempty"`   // content of the triggering comment
	TriggerAuthorType       string                `json:"trigger_author_type,omitempty"`       // "agent" or "member" — author kind for the triggering comment
	TriggerAuthorName       string                `json:"trigger_author_name,omitempty"`       // display name of the triggering comment author
	ChatSessionID           string                `json:"chat_session_id,omitempty"`           // non-empty for chat tasks
	ChatMessage             string                `json:"chat_message,omitempty"`              // user message content for chat tasks
	ChatMessageAttachments  []ChatAttachmentMeta  `json:"chat_message_attachments,omitempty"`  // attachments linked to the chat message; agent uses these to `multica attachment download <id>`
	AutopilotRunID          string                `json:"autopilot_run_id,omitempty"`          // non-empty for autopilot run_only tasks
	AutopilotID             string                `json:"autopilot_id,omitempty"`              // autopilot that spawned this run
	AutopilotTitle          string                `json:"autopilot_title,omitempty"`           // autopilot title used as task context
	AutopilotDescription    string                `json:"autopilot_description,omitempty"`     // autopilot description used as task prompt
	AutopilotSource         string                `json:"autopilot_source,omitempty"`          // manual, schedule, webhook, or api
	AutopilotTriggerPayload json.RawMessage       `json:"autopilot_trigger_payload,omitempty"` // optional trigger payload for webhook/api runs
	QuickCreatePrompt       string                `json:"quick_create_prompt,omitempty"`       // user's natural-language input for quick-create tasks
	QuickCreateParentIssueID string               `json:"quick_create_parent_issue_id,omitempty"` // parent issue ID for quick-create sub-issues
	SquadID                 string                `json:"squad_id,omitempty"`                  // when the picker was a squad, the squad's UUID; Agent is still the resolved leader
	SquadName               string                `json:"squad_name,omitempty"`                // display name for the picker squad, used in prompt text
	// RequestingUserName + RequestingUserProfileDescription describe the human
	// the agent is working on behalf of. v1 sources them from the runtime
	// owner (the user who registered the daemon). Empty when the runtime has
	// no owner (cloud / system runtimes) or the user hasn't set a description.
	// Injected into the brief under `## Requesting User`; omitted entirely
	// when description is empty so the agent doesn't see a useless heading.
	RequestingUserName               string `json:"requesting_user_name,omitempty"`
	RequestingUserProfileDescription string `json:"requesting_user_profile_description,omitempty"`
	// AuthToken is the task-scoped credential the server mints at claim time.
	// The daemon injects it into the spawned agent as MULTICA_TOKEN so the
	// agent never sees the daemon's own (often workspace-owner) credential.
	// Empty when the server-side runtime has no owning user — the daemon
	// then falls back to its own token. See MUL-2600.
	AuthToken string `json:"auth_token,omitempty"`
	// TargetBranch is the resolved git branch this task should target. The
	// server computes it via feature.Resolve(issue, feature): feature's
	// target_branch wins, else the issue's metadata.target_branch override,
	// else the derived 'issue/<identifier>'. Always non-empty for issue tasks;
	// empty for chat / quick-create / autopilot tasks that have no issue.
	TargetBranch string `json:"target_branch,omitempty"`
	// IsSharedBranch is true when the resolved branch came from
	// feature.target_branch — meaning sibling issues of the same feature push
	// to the same branch. The daemon uses this to append a "## Shared branch"
	// warning to the agent brief so agents don't force-push or rewrite history
	// over each other's work.
	IsSharedBranch bool `json:"is_shared_branch,omitempty"`
	// RepoName is the human-readable name of the repo this issue targets.
	// Empty when the issue has no repo_id.
	RepoName string `json:"repo_name,omitempty"`
	// RepoRemoteURL is the remote URL of the repo this issue targets.
	// Used to direct agents to the correct repository for checkout.
	// Empty when the issue has no repo_id.
	RepoRemoteURL string `json:"repo_remote_url,omitempty"`
	// RepoLocalPath is the local filesystem path for the repo (from repo.local_path).
	// May be relative to the manifest root or absolute. Empty when not configured.
	RepoLocalPath string `json:"repo_local_path,omitempty"`
	// CrossRepoSiblings lists sibling issues in the same feature that target
	// different repos. Only populated when the feature spans multiple repos.
	// Used by the daemon to inject cross-repo context into the agent brief.
	CrossRepoSiblings []CrossRepoSiblingData `json:"cross_repo_siblings,omitempty"`
	// Mode is the workspace's execution mode: "worktree" (default) or
	// "in_place". The server projects it from workspace.mode at claim time.
	// In in_place mode the daemon runs the agent in the workspace's real
	// umbrella directory, serialized per workspace; an empty or unknown value
	// is treated as worktree.
	Mode string `json:"mode,omitempty"`
}

// CrossRepoSiblingData describes a sibling issue in a different repo within
// the same feature. Used for cross-repo context injection in the agent brief.
type CrossRepoSiblingData struct {
	IssueIdentifier string `json:"issue_identifier"`
	IssueTitle      string `json:"issue_title"`
	RepoName        string `json:"repo_name"`
}

// ChatAttachmentMeta is the structured attachment metadata the daemon
// hands to the agent for chat tasks. We pass id + filename + content_type
// so the chat prompt can list them explicitly and instruct the agent to
// run `multica attachment download <id>` instead of guessing from a
// signed CDN URL (which expires).
type ChatAttachmentMeta struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}

// AgentData holds agent details returned by the claim endpoint.
type AgentData struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Instructions  string            `json:"instructions"`
	Skills        []SkillData       `json:"skills"`
	CustomEnv     map[string]string `json:"custom_env,omitempty"`
	CustomArgs    []string          `json:"custom_args,omitempty"`
	McpConfig     json.RawMessage   `json:"mcp_config,omitempty"`
	Model         string            `json:"model,omitempty"`
	ThinkingLevel string            `json:"thinking_level,omitempty"`
}

// SkillData represents a structured skill for task execution.
type SkillData struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Content     string          `json:"content"`
	Files       []SkillFileData `json:"files,omitempty"`
}

// SkillFileData represents a supporting file within a skill.
type SkillFileData struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// TaskUsageEntry represents token usage for a single model during a task execution.
type TaskUsageEntry struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	InputTokens      int64  `json:"input_tokens"`
	OutputTokens     int64  `json:"output_tokens"`
	CacheReadTokens  int64  `json:"cache_read_tokens"`
	CacheWriteTokens int64  `json:"cache_write_tokens"`
}

// TaskResult is the outcome of executing a task.
type TaskResult struct {
	Status        string           `json:"status"`
	Comment       string           `json:"comment"`
	BranchName    string           `json:"branch_name,omitempty"`
	EnvType       string           `json:"env_type,omitempty"`
	SessionID     string           `json:"session_id,omitempty"` // Claude session ID for future resumption
	WorkDir       string           `json:"work_dir,omitempty"`   // working directory used during execution
	EnvRoot       string           `json:"-"`                    // env root dir for writing GC metadata (not sent to server)
	FailureReason string           `json:"-"`                    // classifier forwarded to FailTask on the blocked path; empty falls back to 'agent_error'
	Usage         []TaskUsageEntry `json:"usage,omitempty"`      // per-model token usage
}
