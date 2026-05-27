package handler

import (
	"context"
	"sync"
	"time"
)

// WebhookRateLimit is a coarse per-token sliding-window limiter.
//
// Defaults: 60 requests per 60s (1 RPS sustained, with bursts up to 60). The
// goal is "stop a misconfigured or malicious sender from hammering us
// indefinitely" — not "shape traffic to a precise budget" — so the
// implementation aims for cheap and good-enough rather than exact.
type WebhookRateLimit struct {
	Limit  int           // maximum requests per window
	Window time.Duration // sliding window length
}

func DefaultWebhookRateLimit() WebhookRateLimit {
	return WebhookRateLimit{Limit: 60, Window: time.Minute}
}

// DefaultWebhookIPRateLimit is the per-IP coarse budget applied BEFORE the
// trigger lookup. Set lower than the per-token budget on purpose: a single
// IP should rarely sustain more than 30 webhook deliveries / minute across
// all its tokens, while a malicious IP spraying random tokens hits this
// gate before it can probe Postgres.
func DefaultWebhookIPRateLimit() WebhookRateLimit {
	return WebhookRateLimit{Limit: 30, Window: time.Minute}
}

// WebhookRateLimiter is the contract implemented by the in-memory limiter.
//
// Allow returns true when the request is within budget for the given key,
// false when it should be rejected (HTTP 429).
type WebhookRateLimiter interface {
	Allow(ctx context.Context, key string) bool
}

// memoryWebhookRateLimiter keeps per-key timestamps in a slice and prunes them
// on every call. Adequate for single-node deployments.
type memoryWebhookRateLimiter struct {
	cfg WebhookRateLimit
	mu  sync.Mutex
	hit map[string][]time.Time
}

func NewMemoryWebhookRateLimiter(cfg WebhookRateLimit) WebhookRateLimiter {
	return &memoryWebhookRateLimiter{cfg: cfg, hit: make(map[string][]time.Time)}
}

// NewMemoryWebhookIPRateLimiter is the in-memory per-IP variant. Same per-key
// semantics as the per-token memory limiter.
func NewMemoryWebhookIPRateLimiter(cfg WebhookRateLimit) WebhookRateLimiter {
	return NewMemoryWebhookRateLimiter(cfg)
}

func (l *memoryWebhookRateLimiter) Allow(_ context.Context, key string) bool {
	if l.cfg.Limit <= 0 {
		return true
	}
	now := time.Now()
	cutoff := now.Add(-l.cfg.Window)

	l.mu.Lock()
	defer l.mu.Unlock()

	hits := l.hit[key]
	// Trim entries that fell out of the window.
	keep := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.cfg.Limit {
		l.hit[key] = keep
		return false
	}
	keep = append(keep, now)
	l.hit[key] = keep
	return true
}
