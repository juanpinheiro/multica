package daemon

import (
	"strings"
	"testing"
)

// TestBuildQuickCreatePromptRules locks in the rules that govern how the
// quick-create agent is allowed to translate raw user input into the issue
// description body. Each substring corresponds to a concrete failure mode
// observed in production output:
//   - meta-instructions ("create an issue", "cc @X") leaking into the body
//   - the Context section being misused as an apology log when no external
//     references were actually fetched
//   - hard-line rules being silently dropped on prompt rewrites
func TestBuildPromptCommentTriggerPromotesThreadReads(t *testing.T) {
	const (
		issueID   = "issue-thread-1"
		triggerID = "trigger-comment-1"
	)
	task := Task{
		IssueID:               issueID,
		TriggerCommentID:      triggerID,
		TriggerCommentContent: "anything",
		TriggerAuthorType:     "member",
		TriggerAuthorName:     "Bohan",
	}
	out := BuildPrompt(task, "claude")

	mustContain := []string{
		// Thread-first read pinned by trigger comment id, capped via --tail 30.
		"--thread " + triggerID,
		"--tail 30",
		"`multica issue comment list " + issueID + " --thread " + triggerID + " --tail 30 --output json`",
		// Reply cursor walks older replies inside the same thread.
		"Next reply cursor:",
		"--before-id <reply-id>",
		// --recent stays as the cross-thread background fallback.
		"--recent 20 --output json",
		// Cursor walks via the stderr line the CLI emits, not invented flags.
		"Next thread cursor",
		"--before",
		"--before-id",
		// --since is preserved as an additional, combinable knob (now scoped
		// to the post-MUL-2421 mode names).
		"--since",
		"may combine with `--thread --tail` or `--recent`",
		// Discourage the unfiltered full dump on long-running issues.
		"Avoid the unfiltered",
		"wastes context",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("buildCommentPrompt missing thread-first guidance %q\n--- output ---\n%s", s, out)
		}
	}

	// The old "dump everything via --output json alone" prose is exactly the
	// pattern this PR is replacing — guard against the legacy phrasing
	// sneaking back in.
	if strings.Contains(out, "returns all comments for the issue (server caps at 2000)") {
		t.Errorf("buildCommentPrompt still carries the legacy full-dump phrasing")
	}
	// The pre-MUL-2421 unbounded `--thread` recipe (no --tail) is also a
	// regression target: it dumps the entire thread on long threads, which
	// is exactly what --tail 30 is meant to bound.
	if strings.Contains(out, "--thread "+triggerID+" --output json") {
		t.Errorf("buildCommentPrompt regressed to unbounded --thread recipe (no --tail) — long threads will overflow context\n--- output ---\n%s", out)
	}
}

// TestBuildPromptDefaultMentionsRecent pins that the catch-all fallback
// prompt (no trigger comment, no chat, no autopilot, no quick-create) also
// teaches the agent about --recent as the long-issue-friendly alternative
// to the flat dump, even though it cannot anchor a --thread without a
// trigger comment id.
func TestBuildPromptDefaultMentionsRecent(t *testing.T) {
	out := BuildPrompt(Task{IssueID: "issue-default-1"}, "claude")
	for _, s := range []string{
		"--recent 20 --output json",
		"Next thread cursor:",
		"--since",
	} {
		if !strings.Contains(out, s) {
			t.Errorf("default BuildPrompt missing %q\n--- output ---\n%s", s, out)
		}
	}
	// And the default path must NOT inject a --thread example, because there
	// is no trigger comment id to anchor on.
	if strings.Contains(out, "--thread") {
		t.Errorf("default BuildPrompt should NOT mention --thread (no trigger comment to anchor on)\n--- output ---\n%s", out)
	}
	// The legacy "If you need comment history" soft phrasing conflicts with
	// the assignment-trigger runtime workflow, which treats reading comments
	// as mandatory. Guard against it sneaking back in.
	if strings.Contains(out, "If you need comment history") {
		t.Errorf("default BuildPrompt still carries the legacy 'If you need' soft phrasing that conflicts with the mandatory workflow\n--- output ---\n%s", out)
	}
}

// TestBuildPromptNonSquadLeaderNoRule verifies that non-squad-leader agents
// do NOT get the squad leader no_action rule injected.

// TestBuildValidatorPrompt_Claude verifies the Claude-specific fan-out instructions
// are included when provider is "claude".
func TestBuildValidatorPrompt_Claude(t *testing.T) {
	t.Parallel()

	assertions := []ValidatorAssertionData{
		{ID: "a1", Text: "all tests pass", Position: 0},
		{ID: "a2", Text: "no lint errors", Position: 1},
	}
	task := Task{
		IssueID:            "issue-val-1",
		Role:               "validator",
		ValidatorAssertions: assertions,
	}
	out := BuildPrompt(task, "claude")

	for _, want := range []string{
		"a1",
		"all tests pass",
		"a2",
		"no lint errors",
		"<multica-validation-result>",
		"</multica-validation-result>",
		"assertion_id",
		"passed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("validator prompt missing %q\n--- output ---\n%s", want, out)
		}
	}
	// Claude variant must mention parallel / fan-out / Task tool.
	if !strings.Contains(out, "Task") {
		t.Errorf("claude validator prompt should mention Task tool for sub-agent fan-out\n--- output ---\n%s", out)
	}
}

// TestBuildValidatorPrompt_NonClaude verifies sequential checking is described
// when provider is not "claude".
func TestBuildValidatorPrompt_NonClaude(t *testing.T) {
	t.Parallel()

	assertions := []ValidatorAssertionData{
		{ID: "b1", Text: "server starts", Position: 0},
	}
	task := Task{
		IssueID:            "issue-val-2",
		Role:               "validator",
		ValidatorAssertions: assertions,
	}
	out := BuildPrompt(task, "codex")

	for _, want := range []string{
		"b1",
		"server starts",
		"<multica-validation-result>",
		"assertion_id",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("non-claude validator prompt missing %q\n--- output ---\n%s", want, out)
		}
	}
}