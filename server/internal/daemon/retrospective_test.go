package daemon

import (
	"strings"
	"testing"
)

func TestBuildRetrospectivePrompt_IncludesInitiativeAndOutputFormat(t *testing.T) {
	got := buildRetrospectivePrompt(Task{
		Role:         "retrospective",
		FeatureID:    "feat-123",
		FeatureTitle: "Rate limiting",
	})
	if !strings.Contains(got, "feat-123") {
		t.Errorf("prompt missing initiative id:\n%s", got)
	}
	if !strings.Contains(got, "Rate limiting") {
		t.Errorf("prompt missing initiative title:\n%s", got)
	}
	if !strings.Contains(got, "<multica-decision-log>") {
		t.Errorf("prompt missing output block format:\n%s", got)
	}
	if !strings.Contains(got, "docs/adr/") || !strings.Contains(got, "CONTEXT.md") {
		t.Errorf("prompt should instruct updating the architecture docs:\n%s", got)
	}
}

func TestBuildRetrospectivePrompt_FlipsPRReadyWhenSharedBranch(t *testing.T) {
	got := buildRetrospectivePrompt(Task{
		Role:         "retrospective",
		FeatureID:    "feat-123",
		FeatureTitle: "Rate limiting",
		TargetBranch: "feature/rate-limit",
	})
	if !strings.Contains(got, "gh pr ready feature/rate-limit") {
		t.Errorf("prompt should instruct flipping the PR ready for review:\n%s", got)
	}
	if !strings.Contains(got, "Do NOT merge") {
		t.Errorf("prompt should reserve merge for the human gate:\n%s", got)
	}
}

func TestBuildRetrospectivePrompt_NoPRStepWithoutBranch(t *testing.T) {
	got := buildRetrospectivePrompt(Task{
		Role:         "retrospective",
		FeatureID:    "feat-123",
		FeatureTitle: "Rate limiting",
	})
	if strings.Contains(got, "gh pr ready") {
		t.Errorf("prompt must not reference a PR when there is no shared branch:\n%s", got)
	}
}

func TestBuildRetrospectiveSkill_HasNameAndOutputFormat(t *testing.T) {
	skill := buildRetrospectiveSkill()
	if skill.Name != "retrospective-run" {
		t.Errorf("skill name = %q, want retrospective-run", skill.Name)
	}
	if !strings.Contains(skill.Content, "<multica-decision-log>") {
		t.Errorf("skill missing output block format:\n%s", skill.Content)
	}
}
