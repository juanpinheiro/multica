package daemon

import (
	"testing"
)

func TestParseValidationOutput_HappyPath(t *testing.T) {
	t.Parallel()

	output := "some preamble\n<multica-validation-result>{\"results\":[{\"assertion_id\":\"abc-123\",\"passed\":true,\"detail\":\"all good\"}]}</multica-validation-result>\ntrailing text"
	got := parseValidationOutput(output)
	if got == nil {
		t.Fatal("expected non-nil ValidationOutput, got nil")
	}
	if len(got.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got.Results))
	}
	if got.Results[0].AssertionID != "abc-123" {
		t.Errorf("AssertionID = %q, want abc-123", got.Results[0].AssertionID)
	}
	if !got.Results[0].Passed {
		t.Errorf("Passed = false, want true")
	}
	if got.Results[0].Detail != "all good" {
		t.Errorf("Detail = %q, want all good", got.Results[0].Detail)
	}
}

func TestParseValidationOutput_MultipleResults(t *testing.T) {
	t.Parallel()

	output := `<multica-validation-result>{"results":[{"assertion_id":"a1","passed":true},{"assertion_id":"a2","passed":false,"detail":"broke"}]}</multica-validation-result>`
	got := parseValidationOutput(output)
	if got == nil {
		t.Fatal("expected non-nil ValidationOutput, got nil")
	}
	if len(got.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got.Results))
	}
	if got.Results[1].AssertionID != "a2" {
		t.Errorf("second AssertionID = %q, want a2", got.Results[1].AssertionID)
	}
	if got.Results[1].Passed {
		t.Errorf("second Passed = true, want false")
	}
	if got.Results[1].Detail != "broke" {
		t.Errorf("second Detail = %q, want broke", got.Results[1].Detail)
	}
}

func TestParseValidationOutput_NoBlock_ReturnsNil(t *testing.T) {
	t.Parallel()

	output := "agent completed without a validation block"
	got := parseValidationOutput(output)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestParseValidationOutput_MalformedJSON_ReturnsNil(t *testing.T) {
	t.Parallel()

	output := "<multica-validation-result>not-valid-json</multica-validation-result>"
	got := parseValidationOutput(output)
	if got != nil {
		t.Errorf("expected nil for malformed JSON, got %+v", got)
	}
}

func TestParseValidationOutput_EmptyBlock_ReturnsNil(t *testing.T) {
	t.Parallel()

	output := "<multica-validation-result></multica-validation-result>"
	got := parseValidationOutput(output)
	if got != nil {
		t.Errorf("expected nil for empty block, got %+v", got)
	}
}
