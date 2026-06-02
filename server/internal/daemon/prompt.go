package daemon

import (
	"fmt"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

// BuildPrompt constructs the task prompt for an agent CLI.
// Keep this minimal — detailed instructions live in CLAUDE.md / AGENTS.md
// injected by execenv.InjectRuntimeConfig. The provider string is used by
// comment-triggered tasks: Codex's per-turn reply template needs the
// platform-aware "stdin or file" variant, every other provider gets a
// lightweight inline template (or Windows file for any provider on
// Windows).
func BuildPrompt(task Task, provider string) string {
	if task.Role == "validator" {
		return buildValidatorPrompt(task, provider)
	}
	if task.Role == "retrospective" {
		return buildRetrospectivePrompt(task)
	}
	if task.TriggerCommentID != "" {
		return buildCommentPrompt(task, provider)
	}
	if task.AutopilotRunID != "" {
		return buildAutopilotPrompt(task)
	}
	var b strings.Builder
	b.WriteString("You are running as a local coding agent for a Multica workspace.\n\n")
	fmt.Fprintf(&b, "Your assigned issue ID is: %s\n\n", task.IssueID)
	fmt.Fprintf(&b, "Start by running `multica issue get %s --output json` to understand your task, then complete it.\n", task.IssueID)
	fmt.Fprintf(&b, "For comment history, follow the rule in your runtime workflow file (assignment-triggered tasks treat the read as mandatory). `multica issue comment list %s --output json` returns all comments for the issue (server caps at 2000). On long-running issues use `--recent 20 --output json` to read the 20 most recently active threads, then page older threads via the stderr `Next thread cursor: ...` line and the matching `--before` / `--before-id` until you have enough history. `--since <RFC3339>` is still available for incremental polling and may combine with `--recent`.\n", task.IssueID)
	return b.String()
}

// buildCommentPrompt constructs a prompt for comment-triggered tasks.
// The triggering comment content is embedded directly so the agent cannot
// miss it, even when stale output files exist in a reused workdir.
// The reply instructions (including the current TriggerCommentID as --parent)
// are re-emitted on every turn so resumed sessions cannot carry forward a
// previous turn's --parent UUID.
func buildCommentPrompt(task Task, provider string) string {
	var b strings.Builder
	b.WriteString("You are running as a local coding agent for a Multica workspace.\n\n")
	fmt.Fprintf(&b, "Your assigned issue ID is: %s\n\n", task.IssueID)
	if task.TriggerCommentContent != "" {
		authorLabel := "A user"
		if task.TriggerAuthorType == "agent" {
			name := task.TriggerAuthorName
			if name == "" {
				name = "another agent"
			}
			authorLabel = fmt.Sprintf("Another agent (%s)", name)
		}
		fmt.Fprintf(&b, "[NEW COMMENT] %s just left a new comment. Focus on THIS comment — do not confuse it with previous ones:\n\n", authorLabel)
		fmt.Fprintf(&b, "> %s\n\n", task.TriggerCommentContent)
		if task.TriggerAuthorType == "agent" {
			b.WriteString("⚠️ The triggering comment was posted by another agent. Decide whether a reply is warranted. If you produced actual work this turn (investigated, fixed something, answered a real question), post the result as a normal reply — that is NOT a noise comment, and the standard rule that final results must be delivered via comment still applies. If the triggering comment was a pure acknowledgment, thanks, or sign-off AND you produced no work this turn, do NOT reply — and do NOT post a comment saying 'No reply needed' or similar. Simply exit with no output. Silence is the preferred way to end agent-to-agent threads. If you do reply, do not @mention the other agent as a sign-off (that re-triggers them and starts a loop).\n\n")
		}
	}
	fmt.Fprintf(&b, "Start by running `multica issue get %s --output json` to understand your task, then decide how to proceed.\n\n", task.IssueID)
	fmt.Fprintf(&b, "For comment history, read the triggering thread first: `multica issue comment list %s --thread %s --tail 30 --output json` returns the root + the 30 most recent replies in that thread (root is always included, even at `--tail 0`, so you keep the \"what is this about\" context without dragging hundreds of replies into your prompt). If 30 replies aren't enough, walk older replies in the same thread one page at a time by passing the stderr `Next reply cursor: --before <ts> --before-id <reply-id>` line back as `--before <ts> --before-id <reply-id>` on the next call. If you also need cross-thread background, `multica issue comment list %s --recent 20 --output json` pulls the 20 most recently active threads on the issue; under `--recent` the same `--before` / `--before-id` flags walk older *threads* (stderr label: `Next thread cursor`) instead of older replies. Avoid the unfiltered `--output json` form on long-running issues; it dumps the full flat timeline (cap 2000) and wastes context. `--since <RFC3339>` is still available for incremental polling and may combine with `--thread --tail` or `--recent`.\n\n", task.IssueID, task.TriggerCommentID, task.IssueID)
	b.WriteString(execenv.BuildCommentReplyInstructions(provider, task.IssueID, task.TriggerCommentID))
	return b.String()
}

// buildValidatorPrompt constructs a prompt for validator Runs.
// It lists the assertions to check, instructs fan-out via the Task tool for
// Claude, and shows the required output format.
func buildValidatorPrompt(task Task, provider string) string {
	var b strings.Builder
	b.WriteString("You are running as a validator agent for a Multica workspace.\n\n")
	fmt.Fprintf(&b, "Your assigned issue ID is: %s\n\n", task.IssueID)
	b.WriteString("Your role is to verify each Definition of Done assertion for this issue's Milestone.\n\n")

	b.WriteString("## Assertions to Check\n\n")
	if len(task.ValidatorAssertions) == 0 {
		b.WriteString("No assertions have been provided. Return an empty results list.\n\n")
	} else {
		for _, a := range task.ValidatorAssertions {
			fmt.Fprintf(&b, "- [%s] %s\n", a.ID, a.Text)
		}
		b.WriteString("\n")
	}

	if provider == "claude" {
		b.WriteString("## Checking Strategy\n\n")
		b.WriteString("Use the Task tool to launch one read-only sub-agent per assertion in parallel. ")
		b.WriteString("Each sub-agent must check exactly one assertion and report back a pass/fail verdict. ")
		b.WriteString("Wait for all sub-agents to complete before emitting the final output block.\n\n")
	} else {
		b.WriteString("## Checking Strategy\n\n")
		b.WriteString("Check each assertion sequentially. For each one, verify the condition and record whether it passes or fails.\n\n")
	}

	b.WriteString("## Required Output Format\n\n")
	b.WriteString("After checking all assertions, emit exactly one block in this format:\n\n")
	b.WriteString("<multica-validation-result>\n")
	b.WriteString("{\"results\":[{\"assertion_id\":\"<id>\",\"passed\":<true|false>,\"detail\":\"<optional explanation>\"}]}\n")
	b.WriteString("</multica-validation-result>\n\n")
	b.WriteString("Every assertion ID must appear in the results list. Do not skip any.\n")

	return b.String()
}

// buildAutopilotPrompt constructs a prompt for run_only autopilot tasks.
func buildAutopilotPrompt(task Task) string {
	var b strings.Builder
	b.WriteString("You are running as a local coding agent for a Multica workspace.\n\n")
	b.WriteString("This task was triggered by an Autopilot in run-only mode. There is no assigned Multica issue for this run.\n\n")
	fmt.Fprintf(&b, "Autopilot run ID: %s\n", task.AutopilotRunID)
	if task.AutopilotID != "" {
		fmt.Fprintf(&b, "Autopilot ID: %s\n", task.AutopilotID)
	}
	if task.AutopilotTitle != "" {
		fmt.Fprintf(&b, "Autopilot title: %s\n", task.AutopilotTitle)
	}
	if task.AutopilotSource != "" {
		fmt.Fprintf(&b, "Trigger source: %s\n", task.AutopilotSource)
	}
	if strings.TrimSpace(string(task.AutopilotTriggerPayload)) != "" {
		fmt.Fprintf(&b, "Trigger payload:\n%s\n", strings.TrimSpace(string(task.AutopilotTriggerPayload)))
	}
	b.WriteString("\nAutopilot instructions:\n")
	if strings.TrimSpace(task.AutopilotDescription) != "" {
		b.WriteString(task.AutopilotDescription)
		b.WriteString("\n\n")
	} else if task.AutopilotTitle != "" {
		fmt.Fprintf(&b, "%s\n\n", task.AutopilotTitle)
	} else {
		b.WriteString("No additional autopilot instructions were provided. Inspect the autopilot configuration before proceeding.\n\n")
	}
	if task.AutopilotID != "" {
		fmt.Fprintf(&b, "Start by running `multica autopilot get %s --output json` if you need the full autopilot configuration, then complete the instructions above.\n", task.AutopilotID)
	} else {
		b.WriteString("Complete the instructions above.\n")
	}
	b.WriteString("Do not run `multica issue get`; this run does not have an issue ID.\n")
	return b.String()
}
