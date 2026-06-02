package daemon

import (
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

// buildRetrospectiveSkill synthesizes the skill injected into retrospective Runs.
// It tells the Agent to revisit the Initiative's technical decisions, update the
// architecture docs (docs/adr/ and CONTEXT.md) where a decision changed, and emit
// a structured Decision Log block.
func buildRetrospectiveSkill() execenv.SkillContextForEnv {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: retrospective-run\n")
	b.WriteString("description: Retrospective run instructions for the Decision Log\n")
	b.WriteString("---\n\n")
	b.WriteString(retrospectiveBody)
	return execenv.SkillContextForEnv{
		Name:        "retrospective-run",
		Description: "Decision Log retrospective for finished Initiatives",
		Content:     b.String(),
	}
}

// retrospectiveBody is the shared instruction text for the retrospective skill
// and prompt: revisit decisions, update the docs, emit the output block.
const retrospectiveBody = `# Retrospective Run

You are a retrospective agent. The Initiative's work is complete and under review.
Your job is to maintain the self-evolving Decision Log: revisit the technical
decisions made during this Initiative, record what was learned, and keep the
architecture docs current.

## Steps

1. Review the Initiative's issues, handoffs, and the resulting diff to find the
   architectural decisions that were actually made or changed.
2. When a decision changed or sharpened an existing one, update the relevant file
   under ` + "`docs/adr/`" + ` and/or the glossary in ` + "`CONTEXT.md`" + `. Edit the docs in
   place — do not invent new conventions.
3. For each decision worth remembering, link it back to the ADR numbers and the
   CONTEXT.md glossary terms it touches.

## Required Output

After updating the docs, emit exactly one block in this format:

<multica-decision-log>
{"entries":[{"title":"<short title>","decision":"<what was decided>","learning":"<what was learned>","adr_refs":["0004"],"context_terms":["Gate"]}]}
</multica-decision-log>

` + "`adr_refs`" + ` are ADR numbers (e.g. "0004") and ` + "`context_terms`" + ` are CONTEXT.md
glossary terms (e.g. "Gate"). Both are optional per entry. Emit an empty entries
list if there is nothing new to record.
`

// buildRetrospectivePrompt constructs the prompt for retrospective Runs.
func buildRetrospectivePrompt(task Task) string {
	var b strings.Builder
	b.WriteString("You are running as a retrospective agent for a Multica workspace.\n\n")
	if task.FeatureID != "" {
		b.WriteString("Initiative ID: " + task.FeatureID + "\n")
	}
	if task.FeatureTitle != "" {
		b.WriteString("Initiative: " + task.FeatureTitle + "\n")
	}
	b.WriteString("\n")
	b.WriteString(retrospectiveBody)
	b.WriteString(finalizePRBody(task))
	return b.String()
}

// finalizePRBody appends the ready-for-review step. The retrospective runs at the
// in_review boundary — the Initiative's Definition of Done is green — so the one
// PR per Initiative, opened as a draft while work was in flight, is flipped to
// ready-for-review here. This is the agent-driven half of the draft→ready
// lifecycle (the server has no GitHub client; PR operations are the agent CLI's
// job, ADR-0005 / issue 13). No branch ⇒ no shared PR to finalize.
func finalizePRBody(task Task) string {
	if task.TargetBranch == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Finalize the PR\n\n")
	b.WriteString("This Initiative's Definition of Done is green. If an open **draft** PR exists for the feature branch `")
	b.WriteString(task.TargetBranch)
	b.WriteString("`, mark it ready for review so a human reviews a validated change:\n\n")
	b.WriteString("```\ngh pr ready ")
	b.WriteString(task.TargetBranch)
	b.WriteString("\n```\n\n")
	b.WriteString("Run this from the checked-out repository. If the PR is already ready-for-review, or no PR exists yet, skip it. Do NOT merge the PR — review and merge are the human's gate.\n")
	return b.String()
}
