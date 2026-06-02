package handler

import (
	"context"
	"log/slog"
	"time"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// PRMergeClassification is the result of classifying a PR's merge state.
type PRMergeClassification int

const (
	PRStateMerged  PRMergeClassification = iota
	PRStateOpen
	PRStateDraft
	PRStateClosed
	PRStateUnknown
)

// ClassifyPRMergeState classifies a github_pull_request.state value into one
// of the four observable states. Unknown or unrecognised strings resolve to
// PRStateUnknown — never falsely reporting a merge.
func ClassifyPRMergeState(state string) PRMergeClassification {
	switch state {
	case "merged":
		return PRStateMerged
	case "open":
		return PRStateOpen
	case "draft":
		return PRStateDraft
	case "closed":
		return PRStateClosed
	default:
		return PRStateUnknown
	}
}

const (
	defaultPollInterval   = 60 * time.Second
	defaultPollMaxBackoff = 5 * time.Minute
)

// PRMergePoller periodically scans for merged PRs linked to in_review
// Initiatives and drives advanceFeaturesOnPRMerge when a merge is detected.
// It backs off when nothing changes so a quiet AFK run doesn't waste CPU.
//
// This is the local-dev fallback for environments without a reachable GitHub
// webhook endpoint. When webhooks are configured the poller is redundant but
// harmless — advancement is idempotent via the dedup guard in
// advanceFeaturesOnPRMerge.
type PRMergePoller struct {
	queries    *db.Queries
	handler    *Handler
	interval   time.Duration
	maxBackoff time.Duration
}

// NewPRMergePoller creates a poller with the default 60s base interval and
// 5m max backoff.
func NewPRMergePoller(queries *db.Queries, h *Handler) *PRMergePoller {
	return &PRMergePoller{
		queries:    queries,
		handler:    h,
		interval:   defaultPollInterval,
		maxBackoff: defaultPollMaxBackoff,
	}
}

// Run starts the polling loop and blocks until ctx is cancelled. It fires the
// first tick after interval, then backs off exponentially when idle (no
// candidate Initiatives found) up to maxBackoff, resetting to interval on
// activity.
func (p *PRMergePoller) Run(ctx context.Context) {
	current := p.interval
	t := time.NewTimer(current)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if p.tick(ctx) > 0 {
				current = p.interval
			} else {
				current = min(current*2, p.maxBackoff)
			}
			t.Reset(current)
		}
	}
}

// tick performs one poll cycle: queries for issues in in_review features with
// merged PRs and calls advanceFeaturesOnPRMerge. Returns the number of
// candidate issues found (>0 means at least one Initiative had a merged PR).
func (p *PRMergePoller) tick(ctx context.Context) int {
	issues, err := p.queries.ListInReviewIssuesWithMergedPRs(ctx)
	if err != nil {
		slog.Warn("pr merge poller: query failed", "error", err)
		return 0
	}
	if len(issues) == 0 {
		return 0
	}
	p.handler.advanceFeaturesOnPRMerge(ctx, issues)
	return len(issues)
}
