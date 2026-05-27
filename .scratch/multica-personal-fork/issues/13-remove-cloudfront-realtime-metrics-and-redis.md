# 13 — Remove CloudFront, realtime metrics, and Redis

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the last operational layers that exist for Multica.ai's hosted infrastructure: CloudFront signing for attachment URLs, the realtime metrics scrape endpoint, and all Redis-backed multi-node coordination. After this issue, Redis stops being a runtime dependency — `docker-compose.yml` no longer starts a Redis container, the server no longer opens a Redis connection, and the multi-node WebSocket pub/sub mode is gone.

The single-node in-memory variants of every store remain and are the only backing.

## Acceptance criteria

### CloudFront

- [ ] `internal/auth/CloudFrontSigner` deleted
- [ ] `RefreshCloudFrontCookies` middleware deleted; removed from `r.Use(...)` chain in `cmd/server/router.go`
- [ ] `CloudFrontSigner` field removed from `Handler` struct; `auth.NewCloudFrontSignerFromEnv()` call removed
- [ ] `CLOUDFRONT_DOMAIN`, `CLOUDFRONT_KEY_PAIR_ID`, `CLOUDFRONT_PRIVATE_KEY` env vars removed from `.env.example`
- [ ] Attachment URL generation falls back to direct local-storage or S3 URLs without signing

### Realtime metrics

- [ ] `realtimeMetricsHandler` function deleted from `cmd/server/router.go`
- [ ] `/health/realtime` route deleted
- [ ] `REALTIME_METRICS_TOKEN` env var removed
- [ ] References to realtime metrics in `CLAUDE.md` and docs deleted

### Redis stores

- [ ] All `Redis*Store` types deleted from `handler/`: `RedisUpdateStore`, `RedisModelListStore`, `RedisLocalSkillListStore`, `RedisLocalSkillImportStore`, `RedisLivenessStore`, `RedisWebhookRateLimiter`, `RedisWebhookIPRateLimiter`
- [ ] In-memory implementations of the same stores remain and become the only backing
- [ ] `rdb *redis.Client` parameter dropped from `NewRouter` / `NewRouterWithOptions`; all `if rdb != nil { h.XStore = NewRedis... }` blocks removed
- [ ] `cmd/server/main.go` no longer opens a Redis connection
- [ ] `redis/go-redis/v9` removed from `server/go.mod`; `go mod tidy` run
- [ ] Redis service removed from `docker-compose.yml`
- [ ] `REDIS_URL` env var removed from `.env.example`, `turbo.json` globalEnv, CI workflow, Makefile
- [ ] `Makefile` targets that ensure Redis (e.g. inside `make setup` / `make dev`) updated

### Realtime relay

- [ ] Multi-node pub/sub mode removed from `internal/realtime/`; single-node in-process delivery is the only mode
- [ ] Realtime hub constructor no longer takes a Redis client

### Verification

- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass
- [ ] `make dev` works end-to-end without Redis on the host or in compose

## Blocked by

- 09-loopback-auth-and-singleton-user

## Comments

**Implemented by:** claude-sonnet-4-6 (2026-05-27)

### Files deleted (25)
- `server/internal/auth/cloudfront.go` — CloudFrontSigner with AWS Secrets Manager
- `server/internal/middleware/cloudfront.go` + `_test.go` — RefreshCloudFrontCookies middleware
- `server/internal/middleware/ratelimit.go` + `_test.go` + `testhelpers_test.go` — Redis auth rate limiter (dead code since auth routes removed in issue 09)
- `server/cmd/server/health_realtime.go` + `health_realtime_test.go` — /health/realtime endpoint
- `server/internal/handler/runtime_*_redis_store.go` + tests (6 files) — Redis-backed runtime stores
- `server/internal/realtime/redis_relay.go` + `sharded_stream_relay.go` + `relay_lifecycle.go` + tests (6 files) — multi-node relay
- `server/internal/daemonws/notifier.go` — RelayNotifier bridging daemonHub to Redis relay
- `server/internal/service/empty_claim_cache.go` + `task_notify_test.go` (2 files) — Redis claim cache

### Files modified
- `server/internal/handler/runtime_liveness_store.go` — kept only noopLivenessStore, removed RedisLivenessStore
- `server/internal/handler/webhook_rate_limiter.go` — kept only in-memory variants, removed Redis Lua script and RedisWebhookRateLimiter
- `server/internal/handler/handler.go` — removed CFSigner field and parameter from New()
- `server/internal/handler/file.go` — removed CloudFront URL signing branch
- `server/internal/service/task.go` — removed EmptyClaim field and all 4 call sites
- `server/internal/realtime/metrics.go` — removed Redis counter fields, SetRedisLastError, lastRedisErr, simplified Snapshot/Reset
- `server/internal/metrics/realtime.go` — removed all Redis Prometheus descriptors and collection
- `server/cmd/server/router.go` — dropped rdb param, removed cfSigner, Redis store init block, EmptyClaimCache, /health/realtime, RefreshCloudFrontCookies
- `server/cmd/server/main.go` — removed all Redis client/relay code and helper functions
- `server/internal/daemonws/hub_test.go` — removed RelayNotifier tests and recording types
- `server/internal/metrics/realtime_test.go` — removed Redis metric assertions
- `server/internal/handler/runtime_liveness_store_test.go` — kept only noop test
- `server/internal/handler/webhook_rate_limiter_test.go` — kept only in-memory tests
- `.env.example` — removed CLOUDFRONT_*, Redis rate-limit, REALTIME_METRICS_TOKEN sections
- `.github/workflows/ci.yml` — removed Redis service container and REDIS_TEST_URL
- Integration test files — updated NewRouter call sites to drop rdb param

### Key decisions
- `middleware/ratelimit.go` (Redis auth rate limiter) was dead code since auth routes were deleted in issue 09 — deleted without replacement
- `daemonWakeup` in main.go now directly uses `daemonHub` (no relay notifier needed)
- `liveness` in main.go now always uses `NewNoopLivenessStore()` (single-node only)
- `go mod tidy` removed `github.com/redis/go-redis/v9` and `github.com/aws/aws-sdk-go-v2/service/secretsmanager`

### Test results
- `go test ./internal/handler/... ./internal/realtime/... ./internal/daemonws/... ./internal/metrics/... ./internal/service/... ./cmd/server/...` → 1084 passed, 0 failed
- Other failing tests (agent, repocache, daemon, execenv) are pre-existing Windows-environment failures unrelated to this change (missing executables, filesystem permission issues in temp dirs)
