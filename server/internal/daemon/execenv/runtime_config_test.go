package execenv

import (
	"strings"
	"testing"
)

// Sub-issue Creation section — after MUL-2538 the platform posts the
// child-done parent notification itself, so the brief no longer carries
// any parent-notification rule (per Bohan's call on PR #3055: delete the
// guidance entirely, do not replace it with a "do not post one" sentence
// — the agent should not be thinking about parent comments at all). All
// that remains is the `--status todo` vs `--status backlog` rule for
// creating sub-issues, which is unrelated to the notification path.

func TestSubIssueCreationSectionPresentForIssueRuns(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ctx  TaskContextForEnv
	}{
		{
			name: "assignment-triggered",
			ctx:  TaskContextForEnv{IssueID: "11111111-2222-3333-4444-555555555555"},
		},
		{
			name: "comment-triggered",
			ctx: TaskContextForEnv{
				IssueID:          "22222222-3333-4444-5555-666666666666",
				TriggerCommentID: "33333333-4444-5555-6666-777777777777",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)

			if !strings.Contains(out, "## Sub-issue Creation") {
				t.Fatalf("expected Sub-issue Creation section in %s brief", tc.name)
			}
			for _, want := range []string{
				"**Choosing `--status` when creating sub-issues.**",
				"`--status todo` = **start now**",
				"`--status backlog` = **wait**",
				"`multica issue status <child-id> todo`",
				"all `--status todo`",
				"`--status backlog` from the start",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("[%s] section missing %q", tc.name, want)
				}
			}
		})
	}
}

// The brief must no longer carry any parent-notification guidance. PR
// #2918 added a "Tell the parent when you finish a child" rule that
// turned into noise (self-mention loops, planner ack ping-pong,
// hardcoded `MUL-` prefix). PR #3055 first downgraded it to a "do NOT
// post one" guardrail, but Bohan's product call was to remove the
// guidance entirely rather than substitute a new prohibition. These
// canaries lock that in: any wording that re-introduces the
// parent-comment concept — positive, negative, or descriptive — must
// not come back through future edits.
func TestBriefHasNoParentNotificationGuidance(t *testing.T) {
	t.Parallel()
	cases := []TaskContextForEnv{
		{IssueID: "11111111-2222-3333-4444-555555555555"},
		{
			IssueID:          "22222222-3333-4444-5555-666666666666",
			TriggerCommentID: "33333333-4444-5555-6666-777777777777",
		},
	}
	for _, ctx := range cases {
		ctx := ctx
		out := buildMetaSkillContent("claude", ctx)

		// The pre-MUL-2538 phrasing instructed the agent to compose a
		// parent comment by hand — including a hardcoded `MUL-` prefix
		// and an assignee mention. The intermediate revision (PR #3055
		// before Bohan's call) instead told the agent NOT to post one.
		// Both framings must stay out.
		for _, banned := range []string{
			// Old "do it yourself" framing (PR #2918).
			"## Parent / Sub-issue Protocol",
			"**Tell the parent when you finish a child.**",
			"multica issue comment add <parent-id>",
			"with NO `--parent`",
			"link the child as `[MUL-",
			"`@mention` the parent's assignee",
			"`mention://agent/<id>`",
			"`mention://member/<id>`",
			"`mention://squad/<id>`",
			// Intermediate "do NOT do it yourself" framing (PR #3055
			// before Bohan's call) — also out per product direction.
			"**Do NOT post your own parent-notification comment.**",
			"Do NOT post your own parent-notification comment",
			"parent-notification comment",
			"system comment on the parent fires from the status transition",
			"re-trigger the parent's assignee for nothing",
			"platform posts a top-level system comment on the parent",
			// Earlier revisions split rules by trigger type or used
			// table/subsection layouts. None of those structures should
			// come back either.
			"| Parent assignee | Parent status |",
			"The same agent as yourself",
			"| Member or squad |",
			"### A. Notify the parent",
			"### B. Choose",
			"When this issue has `parent_issue_id`:",
			"**Closing out child work** (only if this issue has `parent_issue_id`)",
			"**Notify the parent** (only if this issue has `parent_issue_id`",
			"**Creating sub-issues** (applies to any issue-bound run)",
			"For parent/child work, use these best-effort rules",
			// The protocol must no longer emit a placeholder
			// `<this-issue-id>` status flip — the workflow above owns
			// that command with the real issue id substituted.
			"`multica issue status <this-issue-id> in_review`",
			// Non-existent CLI form Elon's earlier review flagged.
			"issue list --parent",
		} {
			if strings.Contains(out, banned) {
				t.Errorf("expected %q to be removed from the brief", banned)
			}
		}
	}
}

// Comment-triggered briefs must NOT carry any unconditional status-flip
// command targeting the current issue. Previous revisions had a
// dedicated protocol step that wrote `multica issue status <this-issue-id> in_review`;
// the comment-triggered workflow rule "Do NOT change the issue status
// unless the comment explicitly asks for it" must remain the source of
// truth (Elon's blocking review on PR #2918).
func TestCommentTriggeredProtocolDoesNotForceInReview(t *testing.T) {
	t.Parallel()
	ctx := TaskContextForEnv{
		IssueID:          "55555555-6666-7777-8888-999999999999",
		TriggerCommentID: "66666666-7777-8888-9999-aaaaaaaaaaaa",
	}
	out := buildMetaSkillContent("claude", ctx)

	if strings.Contains(out, "`multica issue status <this-issue-id> in_review`") {
		t.Errorf("comment-triggered brief must not contain a placeholder `<this-issue-id> in_review` flip — that conflicts with the comment-triggered \"do not change status unless asked\" rule")
	}

	const guardrail = "Do NOT change the issue status unless the comment explicitly asks for it"
	if !strings.Contains(out, guardrail) {
		t.Errorf("expected the comment-triggered workflow guardrail %q to be present", guardrail)
	}
}

// Assignment-triggered briefs are the inverse boundary: when the agent
// owns the issue lifecycle, the brief AS A WHOLE must still tell it to
// flip to in_review on completion. The flip lives in the
// assignment-triggered workflow above (with the real id substituted).
func TestAssignmentTriggeredProtocolStillFlipsInReview(t *testing.T) {
	t.Parallel()
	const issueID = "77777777-8888-9999-aaaa-bbbbbbbbbbbb"
	ctx := TaskContextForEnv{IssueID: issueID}
	out := buildMetaSkillContent("claude", ctx)

	want := "`multica issue status " + issueID + " in_review`"
	if !strings.Contains(out, want) {
		t.Errorf("assignment-triggered brief must still flip to in_review on completion (expected %q in the workflow above)", want)
	}
}

// The sub-issue creation rule must reach top-level parents that have no
// `parent_issue_id` of their own — that is where the `todo` vs `backlog`
// decision matters most. The section must not gate on this issue being
// a child, and must not even mention `parent_issue_id`.
func TestSubIssueCreationSectionIsUnconditional(t *testing.T) {
	t.Parallel()
	ctx := TaskContextForEnv{
		IssueID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}
	out := buildMetaSkillContent("claude", ctx)

	const header = "## Sub-issue Creation"
	start := strings.Index(out, header)
	if start == -1 {
		t.Fatalf("sub-issue creation section missing")
	}
	rest := out[start:]
	end := strings.Index(rest[len(header):], "\n## ")
	var section string
	if end == -1 {
		section = rest
	} else {
		section = rest[:len(header)+end]
	}

	if strings.Contains(section, "parent_issue_id") {
		t.Errorf("Sub-issue Creation section must not reference `parent_issue_id` — it applies to any issue-bound run, including top-level parents:\n%s", section)
	}
}

// Workspace Context block: workspace.context (the per-workspace system prompt
// owners set in Settings → General) must reach the brief as `## Workspace
// Context` for every task kind so agents see a consistent shared system prompt
// regardless of how they were triggered. Empty content must skip the heading
// entirely — bare headings would just add noise.
func TestWorkspaceContextRenderedAcrossTaskKinds(t *testing.T) {
	t.Parallel()
	const wsContext = "All comments must be in English. Prefer concise PR descriptions."
	cases := []struct {
		name string
		ctx  TaskContextForEnv
	}{
		{
			name: "assignment-triggered",
			ctx: TaskContextForEnv{
				IssueID:          "11111111-2222-3333-4444-555555555555",
				WorkspaceContext: wsContext,
			},
		},
		{
			name: "comment-triggered",
			ctx: TaskContextForEnv{
				IssueID:          "22222222-3333-4444-5555-666666666666",
				TriggerCommentID: "33333333-4444-5555-6666-777777777777",
				WorkspaceContext: wsContext,
			},
		},
		{
			name: "chat",
			ctx: TaskContextForEnv{
				ChatSessionID:    "chat-1",
				WorkspaceContext: wsContext,
			},
		},
		{
			name: "quick-create",
			ctx: TaskContextForEnv{
				QuickCreatePrompt: "create me an issue",
				WorkspaceContext:  wsContext,
			},
		},
		{
			name: "autopilot run-only",
			ctx: TaskContextForEnv{
				AutopilotRunID:   "run-1",
				WorkspaceContext: wsContext,
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)

			if !strings.Contains(out, "## Workspace Context") {
				t.Fatalf("[%s] expected `## Workspace Context` heading", tc.name)
			}
			if !strings.Contains(out, wsContext) {
				t.Errorf("[%s] brief missing workspace context body %q", tc.name, wsContext)
			}
			// The block must precede Available Commands so it acts as
			// background framing, not a footer hidden below CLI usage.
			ctxIdx := strings.Index(out, "## Workspace Context")
			cmdsIdx := strings.Index(out, "## Available Commands")
			if ctxIdx == -1 || cmdsIdx == -1 || ctxIdx > cmdsIdx {
				t.Errorf("[%s] `## Workspace Context` must appear above `## Available Commands` (ctx=%d, cmds=%d)", tc.name, ctxIdx, cmdsIdx)
			}
		})
	}
}

func TestWorkspaceContextHeadingSkippedWhenEmpty(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ctx  TaskContextForEnv
	}{
		{
			name: "empty string",
			ctx: TaskContextForEnv{
				IssueID:          "11111111-2222-3333-4444-555555555555",
				WorkspaceContext: "",
			},
		},
		{
			name: "whitespace only",
			ctx: TaskContextForEnv{
				IssueID:          "11111111-2222-3333-4444-555555555555",
				WorkspaceContext: "   \n\t  \r\n",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)
			if strings.Contains(out, "## Workspace Context") {
				t.Errorf("[%s] empty workspace context must NOT emit the heading", tc.name)
			}
		})
	}
}

// TestSharedBranchSectionPresentWhenIsSharedBranch verifies that the
// "## Shared branch" safety section appears in the brief exactly when
// IsSharedBranch is true, and is absent otherwise. The section guards
// against force-pushes and history rewrites when multiple sibling issues
// converge on the same feature branch.
func TestSharedBranchSectionPresentWhenIsSharedBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		ctx         TaskContextForEnv
		wantSection bool
	}{
		{
			name: "shared branch - section present",
			ctx: TaskContextForEnv{
				IssueID:        "11111111-2222-3333-4444-555555555555",
				TargetBranch:   "feature/auth-v2",
				IsSharedBranch: true,
			},
			wantSection: true,
		},
		{
			name: "isolated branch - no section",
			ctx: TaskContextForEnv{
				IssueID:        "22222222-3333-4444-5555-666666666666",
				TargetBranch:   "issue/MUL-123",
				IsSharedBranch: false,
			},
			wantSection: false,
		},
		{
			name: "no target branch - no section",
			ctx: TaskContextForEnv{
				IssueID: "33333333-4444-5555-6666-777777777777",
			},
			wantSection: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)
			hasSection := strings.Contains(out, "## Shared branch")
			if hasSection != tc.wantSection {
				t.Errorf("IsSharedBranch=%v: expected section=%v, got=%v\n--- output ---\n%s",
					tc.ctx.IsSharedBranch, tc.wantSection, hasSection, out)
			}
		})
	}
}

// TestSharedBranchSectionContent verifies that the section body matches
// the spec verbatim: branch name is present, force-push rule, rebase
// rule, pull-before-push rule, and PR append rule.
func TestSharedBranchSectionContent(t *testing.T) {
	t.Parallel()
	const branch = "feature/auth-v2"
	ctx := TaskContextForEnv{
		IssueID:        "44444444-5555-6666-7777-888888888888",
		TargetBranch:   branch,
		IsSharedBranch: true,
	}
	out := buildMetaSkillContent("claude", ctx)

	for _, want := range []string{
		"## Shared branch",
		"`" + branch + "`",
		"Other issues of this feature also push there",
		"Do not `git push --force`",
		"git rebase -i",
		"git pull --rebase",
		"### Consolidated PR model",
		"do NOT open a new PR",
		"## Implements",
		"### Status workflow under shared branch",
		"NOT `in_review`",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("shared branch section missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestSharedBranchProtocolMarksDoneNotInReview verifies that when the issue
// rides a shared feature branch, the assignment workflow tells the agent to
// flip to `done` (not `in_review`) on completion. The dependency gate keys on
// `done`, so leaving the issue at `in_review` would stall every dependent
// issue and break the "1 feature = 1 PR, reviewed once at the end" model.
func TestSharedBranchProtocolMarksDoneNotInReview(t *testing.T) {
	t.Parallel()
	const issueID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	ctx := TaskContextForEnv{
		IssueID:        issueID,
		TargetBranch:   "feature/auth-v2",
		IsSharedBranch: true,
	}
	out := buildMetaSkillContent("claude", ctx)

	wantDone := "`multica issue status " + issueID + " done`"
	if !strings.Contains(out, wantDone) {
		t.Errorf("shared-branch brief must flip to `done`, expected %q\n--- output ---\n%s", wantDone, out)
	}
	unwantedInReview := "`multica issue status " + issueID + " in_review`"
	if strings.Contains(out, unwantedInReview) {
		t.Errorf("shared-branch brief must NOT flip to `in_review` (got %q in output)\n--- output ---\n%s", unwantedInReview, out)
	}
}

// TestSharedBranchPRConsolidationGuidance verifies that the brief tells the
// agent to use a feature-level PR title and lists all sibling issues under a
// `## Implements` section. Subsequent agents (later issues of the same
// feature) must not open a new PR — they append commits to the existing one.
func TestSharedBranchPRConsolidationGuidance(t *testing.T) {
	t.Parallel()
	const featureID = "fea7feaf-fea7-feaf-fea7-feaffea7feaf"
	const featureTitle = "Connect Telegram"
	ctx := TaskContextForEnv{
		IssueID:        "11111111-2222-3333-4444-555555555555",
		TargetBranch:   "feature/telegram",
		IsSharedBranch: true,
		FeatureID:      featureID,
		FeatureTitle:   featureTitle,
	}
	out := buildMetaSkillContent("claude", ctx)

	for _, want := range []string{
		"multica issue list --feature " + featureID,
		"`feat: " + featureTitle + "`",
		"## Implements",
		"do NOT open a new PR",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("shared-branch PR guidance missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestRepoCheckoutInstructionMentionsTargetBranchWhenShared verifies that
// the repos section tells the agent to use --ref <branch> when a shared
// target branch is set, so the agent picks up sibling-issue commits.
func TestRepoCheckoutInstructionMentionsTargetBranchWhenShared(t *testing.T) {
	t.Parallel()
	const branch = "feature/auth-v2"
	ctx := TaskContextForEnv{
		IssueID:      "55555555-6666-7777-8888-999999999999",
		TargetBranch: branch,
		IsSharedBranch: true,
		Repos: []RepoContextForEnv{
			{URL: "https://github.com/org/repo.git"},
		},
	}
	out := buildMetaSkillContent("claude", ctx)
	if !strings.Contains(out, "--ref "+branch) {
		t.Errorf("repos section must mention '--ref %s' when IsSharedBranch is true\n--- output ---\n%s", branch, out)
	}
}

// TestRepoCheckoutInstructionNoRefWhenIsolated verifies that the repos
// section does NOT inject --ref <derived-branch> for isolated issue tasks
// (where the branch is derived as "issue/MUL-123" and doesn't exist on
// the remote yet).
func TestRepoCheckoutInstructionNoRefWhenIsolated(t *testing.T) {
	t.Parallel()
	ctx := TaskContextForEnv{
		IssueID:        "66666666-7777-8888-9999-aaaaaaaaaaaa",
		TargetBranch:   "issue/MUL-456",
		IsSharedBranch: false,
		Repos: []RepoContextForEnv{
			{URL: "https://github.com/org/repo.git"},
		},
	}
	out := buildMetaSkillContent("claude", ctx)
	if strings.Contains(out, "--ref issue/MUL-456") {
		t.Errorf("repos section must NOT inject --ref for isolated branches that don't exist on remote\n--- output ---\n%s", out)
	}
}

func TestSubIssueCreationSectionSkippedForNonIssueModes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ctx  TaskContextForEnv
	}{
		{
			name: "chat",
			ctx:  TaskContextForEnv{ChatSessionID: "chat-1"},
		},
		{
			name: "quick-create",
			ctx:  TaskContextForEnv{QuickCreatePrompt: "create me an issue"},
		},
		{
			name: "autopilot run-only",
			ctx:  TaskContextForEnv{AutopilotRunID: "run-1"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)
			if strings.Contains(out, "## Sub-issue Creation") {
				t.Errorf("%s mode must NOT emit the Sub-issue Creation section", tc.name)
			}
		})
	}
}

// TestCrossRepoContextSectionPresent verifies that the "## Cross-repo context"
// section appears when the task carries cross-repo sibling data, and lists each
// sibling's identifier, title, and repo name.
func TestCrossRepoContextSectionPresent(t *testing.T) {
	t.Parallel()
	ctx := TaskContextForEnv{
		IssueID: "11111111-2222-3333-4444-555555555555",
		CrossRepoSiblings: []CrossRepoSiblingContext{
			{IssueIdentifier: "MUL-42", IssueTitle: "Add auth endpoints", RepoName: "backend"},
			{IssueIdentifier: "MUL-44", IssueTitle: "Add QA tests", RepoName: "qa"},
		},
	}
	out := buildMetaSkillContent("claude", ctx)

	for _, want := range []string{
		"## Cross-repo context",
		"MUL-42",
		"Add auth endpoints",
		"backend",
		"MUL-44",
		"Add QA tests",
		"qa",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("cross-repo context section missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestCrossRepoContextSectionAbsentWithoutSiblings verifies that no cross-repo
// section is emitted when there are no cross-repo siblings.
func TestCrossRepoContextSectionAbsentWithoutSiblings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ctx  TaskContextForEnv
	}{
		{
			name: "no siblings",
			ctx:  TaskContextForEnv{IssueID: "11111111-2222-3333-4444-555555555555"},
		},
		{
			name: "empty siblings slice",
			ctx: TaskContextForEnv{
				IssueID:           "22222222-3333-4444-5555-666666666666",
				CrossRepoSiblings: []CrossRepoSiblingContext{},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := buildMetaSkillContent("claude", tc.ctx)
			if strings.Contains(out, "## Cross-repo context") {
				t.Errorf("[%s] cross-repo context section must not appear without siblings\n--- output ---\n%s", tc.name, out)
			}
		})
	}
}

// TestCrossRepoCheckoutUsesSpecificURLWhenAvailable verifies that when both
// RepoRemoteURL and IsSharedBranch are set, the repos checkout instruction
// uses the specific URL from RepoRemoteURL instead of the generic <url>
// placeholder.
func TestCrossRepoCheckoutUsesSpecificURLWhenAvailable(t *testing.T) {
	t.Parallel()
	const (
		remoteURL = "github.com/voce/backend"
		branch    = "feature/auth-v2"
	)
	ctx := TaskContextForEnv{
		IssueID:        "11111111-2222-3333-4444-555555555555",
		TargetBranch:   branch,
		IsSharedBranch: true,
		RepoRemoteURL:  remoteURL,
		Repos: []RepoContextForEnv{
			{URL: remoteURL, Description: "backend"},
		},
	}
	out := buildMetaSkillContent("claude", ctx)
	want := "multica repo checkout " + remoteURL + " --ref " + branch
	if !strings.Contains(out, want) {
		t.Errorf("repos section must emit specific checkout command %q\n--- output ---\n%s", want, out)
	}
}
