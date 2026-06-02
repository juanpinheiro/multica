// Package decisionlog holds the pure domain logic for the Decision Log — the
// self-evolving layer of architectural decisions kept current by a retrospective
// Run at an Initiative boundary (see CONTEXT.md "Decision Log"). A retrospective
// Agent revisits technical decisions, records what was learned, and emits a
// structured block this package parses; the entries link back to the ADRs and
// CONTEXT terms they touch.
//
// Like dod and handoff, this module is pure (no I/O): the daemon parses the
// agent's output here and the handler persists the cleaned entries.
package decisionlog

import (
	"encoding/json"
	"strings"
)

const (
	blockOpen  = "<multica-decision-log>"
	blockClose = "</multica-decision-log>"
)

// Entry is one architectural decision a retrospective records. AdrRefs and
// ContextTerms are the links to the durable architecture docs: the ADR numbers
// (e.g. "0004") and the CONTEXT.md glossary terms (e.g. "Gate") the decision
// touches.
type Entry struct {
	Title        string   `json:"title"`
	Decision     string   `json:"decision"`
	Learning     string   `json:"learning,omitempty"`
	AdrRefs      []string `json:"adr_refs,omitempty"`
	ContextTerms []string `json:"context_terms,omitempty"`
}

// Output is the structured block a retrospective Agent emits on completion.
type Output struct {
	Entries []Entry `json:"entries"`
}

// Parse scans agent output for a <multica-decision-log> block and decodes the
// JSON inside. Returns nil when no block is found, the block is empty, or the
// JSON is invalid — mirrors the validator's parse so malformed agent output is a
// non-event rather than an error.
func Parse(output string) *Output {
	start := strings.Index(output, blockOpen)
	if start < 0 {
		return nil
	}
	inner := output[start+len(blockOpen):]
	end := strings.Index(inner, blockClose)
	if end < 0 {
		return nil
	}
	raw := strings.TrimSpace(inner[:end])
	if raw == "" {
		return nil
	}
	var o Output
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		return nil
	}
	return &o
}

// ValidEntries returns the cleaned, persistable entries from an Output: each is
// trimmed, its ref/term lists trimmed-deduped-of-empties, and entries missing a
// title or decision are dropped (defensive against agent output). Always returns
// a non-nil slice; ref/term lists on kept entries are non-nil too.
func ValidEntries(o *Output) []Entry {
	out := []Entry{}
	if o == nil {
		return out
	}
	for _, e := range o.Entries {
		title := strings.TrimSpace(e.Title)
		decision := strings.TrimSpace(e.Decision)
		if title == "" || decision == "" {
			continue
		}
		out = append(out, Entry{
			Title:        title,
			Decision:     decision,
			Learning:     strings.TrimSpace(e.Learning),
			AdrRefs:      cleanList(e.AdrRefs),
			ContextTerms: cleanList(e.ContextTerms),
		})
	}
	return out
}

// cleanList trims each item, drops empties, and dedupes while preserving order.
// Returns a non-nil empty slice when nothing survives.
func cleanList(items []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
