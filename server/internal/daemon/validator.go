package daemon

import (
	"encoding/json"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

// ValidationOutput carries per-assertion verdicts the validator agent emits.
type ValidationOutput struct {
	Results []ValidationResultData `json:"results"`
}

// ValidationResultData is a single assertion verdict inside a ValidationOutput.
type ValidationResultData struct {
	AssertionID string `json:"assertion_id"`
	Passed      bool   `json:"passed"`
	Detail      string `json:"detail,omitempty"`
}

const (
	validationResultOpen  = "<multica-validation-result>"
	validationResultClose = "</multica-validation-result>"
)

// parseValidationOutput scans agent output for a <multica-validation-result>
// block and decodes the JSON inside. Returns nil when no block is found, when
// the block is empty, or when the JSON is invalid.
func parseValidationOutput(output string) *ValidationOutput {
	start := strings.Index(output, validationResultOpen)
	if start < 0 {
		return nil
	}
	inner := output[start+len(validationResultOpen):]
	end := strings.Index(inner, validationResultClose)
	if end < 0 {
		return nil
	}
	raw := strings.TrimSpace(inner[:end])
	if raw == "" {
		return nil
	}
	var vo ValidationOutput
	if err := json.Unmarshal([]byte(raw), &vo); err != nil {
		return nil
	}
	return &vo
}

// buildValidatorSkill synthesizes the skill injected into validator Runs.
// For Claude, it includes sub-agent fan-out instructions via the Task tool.
// For other providers, sequential checking instructions are used instead.
func buildValidatorSkill(assertions []ValidatorAssertionData, provider string) execenv.SkillContextForEnv {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: validator-run\n")
	b.WriteString("description: Validator run instructions for Definition of Done checking\n")
	b.WriteString("---\n\n")
	b.WriteString("# Validator Run\n\n")
	b.WriteString("You are a validator agent. Your job is to check whether each Definition of Done assertion has been satisfied.\n\n")

	if len(assertions) > 0 {
		b.WriteString("## Assertions to Check\n\n")
		for _, a := range assertions {
			b.WriteString("- ")
			b.WriteString("[")
			b.WriteString(a.ID)
			b.WriteString("] ")
			b.WriteString(a.Text)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if provider == "claude" {
		b.WriteString("## Checking Strategy (Claude)\n\n")
		b.WriteString("Use the Task tool to fan out one read-only sub-agent per assertion in parallel. ")
		b.WriteString("Each sub-agent checks exactly one assertion and returns a pass/fail verdict with an optional detail string.\n\n")
		b.WriteString("Spawn all sub-agents concurrently, then wait for all of them to finish before producing the final output block.\n\n")
	} else {
		b.WriteString("## Checking Strategy\n\n")
		b.WriteString("Check each assertion sequentially. For each assertion, verify the condition and record whether it passes or fails with a brief explanation.\n\n")
	}

	b.WriteString("## Required Output\n\n")
	b.WriteString("After checking all assertions, emit exactly one output block using this format:\n\n")
	b.WriteString("```\n")
	b.WriteString("<multica-validation-result>\n")
	b.WriteString("{\"results\":[{\"assertion_id\":\"<id>\",\"passed\":<true|false>,\"detail\":\"<optional explanation>\"}]}\n")
	b.WriteString("</multica-validation-result>\n")
	b.WriteString("```\n\n")
	b.WriteString("Every assertion must appear in the results list. Do not omit any assertion ID.\n")

	return execenv.SkillContextForEnv{
		Name:        "validator-run",
		Description: "Definition of Done assertion checking for validator Runs",
		Content:     b.String(),
	}
}
