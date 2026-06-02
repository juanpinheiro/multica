package decisionlog

import "testing"

func TestParse_SingleEntry(t *testing.T) {
	out := Parse(`prose before
<multica-decision-log>
{"entries":[{"title":"Keep Gate thin","decision":"SQL enforces, Go specifies","learning":"two layers stay in sync","adr_refs":["0004"],"context_terms":["Gate"]}]}
</multica-decision-log>
prose after`)
	if out == nil {
		t.Fatal("Parse returned nil, want one entry")
	}
	if len(out.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(out.Entries))
	}
	e := out.Entries[0]
	if e.Title != "Keep Gate thin" || e.Decision != "SQL enforces, Go specifies" {
		t.Errorf("entry mismatch: %+v", e)
	}
	if len(e.AdrRefs) != 1 || e.AdrRefs[0] != "0004" {
		t.Errorf("adr_refs = %v", e.AdrRefs)
	}
	if len(e.ContextTerms) != 1 || e.ContextTerms[0] != "Gate" {
		t.Errorf("context_terms = %v", e.ContextTerms)
	}
}

func TestParse_MultipleEntries(t *testing.T) {
	out := Parse(`<multica-decision-log>{"entries":[{"title":"a","decision":"d1"},{"title":"b","decision":"d2"}]}</multica-decision-log>`)
	if out == nil || len(out.Entries) != 2 {
		t.Fatalf("want 2 entries, got %+v", out)
	}
}

func TestParse_NoBlock(t *testing.T) {
	if got := Parse("no block here"); got != nil {
		t.Errorf("Parse = %+v, want nil", got)
	}
}

func TestParse_EmptyBlock(t *testing.T) {
	if got := Parse("<multica-decision-log>\n\n</multica-decision-log>"); got != nil {
		t.Errorf("Parse = %+v, want nil", got)
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	if got := Parse("<multica-decision-log>{not json}</multica-decision-log>"); got != nil {
		t.Errorf("Parse = %+v, want nil", got)
	}
}

func TestParse_UnterminatedBlock(t *testing.T) {
	if got := Parse(`<multica-decision-log>{"entries":[]}`); got != nil {
		t.Errorf("Parse = %+v, want nil", got)
	}
}

func TestValidEntries_NilInput(t *testing.T) {
	got := ValidEntries(nil)
	if got == nil {
		t.Fatal("ValidEntries(nil) = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestValidEntries_TrimsAndDedupesRefs(t *testing.T) {
	got := ValidEntries(&Output{Entries: []Entry{{
		Title:        "  Title  ",
		Decision:     " Decision ",
		Learning:     "  learned  ",
		AdrRefs:      []string{" 0004 ", "0004", "", "0005"},
		ContextTerms: []string{"Gate", "Gate ", ""},
	}}})
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1", len(got))
	}
	e := got[0]
	if e.Title != "Title" || e.Decision != "Decision" || e.Learning != "learned" {
		t.Errorf("trim failed: %+v", e)
	}
	if len(e.AdrRefs) != 2 || e.AdrRefs[0] != "0004" || e.AdrRefs[1] != "0005" {
		t.Errorf("adr_refs dedupe/trim failed: %v", e.AdrRefs)
	}
	if len(e.ContextTerms) != 1 || e.ContextTerms[0] != "Gate" {
		t.Errorf("context_terms dedupe/trim failed: %v", e.ContextTerms)
	}
}

func TestValidEntries_DropsEntriesMissingTitleOrDecision(t *testing.T) {
	got := ValidEntries(&Output{Entries: []Entry{
		{Title: "", Decision: "d"},
		{Title: "t", Decision: ""},
		{Title: "  ", Decision: "  "},
		{Title: "ok", Decision: "ok"},
	}})
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1 (only the valid one)", len(got))
	}
	if got[0].Title != "ok" {
		t.Errorf("kept wrong entry: %+v", got[0])
	}
}

func TestValidEntries_NonNilEmptyRefSlices(t *testing.T) {
	got := ValidEntries(&Output{Entries: []Entry{{Title: "t", Decision: "d"}}})
	if got[0].AdrRefs == nil || got[0].ContextTerms == nil {
		t.Errorf("ref slices must be non-nil: %+v", got[0])
	}
}
