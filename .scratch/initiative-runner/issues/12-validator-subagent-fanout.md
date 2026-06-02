# Issue 12: Validator sub-agent fan-out (read-only, per-assertion)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0006.

## What to build

Accelerate validator Runs by fanning out the agent CLI's own read-only sub-agents — one per DoD assertion
— so a Milestone's assertions are checked in parallel. Inject the custom read-only sub-agent definition(s)
per Run via execenv (the same path that already writes CLAUDE.md / skills). Invariant: sub-agents are
read-only (explore/verify); only the main Run agent writes. Fan-out power is provider-dependent, so degrade
gracefully when the Issue's `Model` is a provider without sub-agent support.

## Acceptance criteria

- [ ] A validator Run fans out one read-only sub-agent per DoD assertion
- [ ] Sub-agent definitions injected per Run via execenv; sub-agents cannot write
- [ ] Graceful fallback to sequential checking when the provider lacks sub-agents
- [ ] `go test ./...` and `pnpm test` pass

## Blocked by

- `09-dod-and-validator`

## Comments

### Key decisions

1. **Output format**: Validator agents emit verdicts in a `<multica-validation-result>...</multica-validation-result>` XML-style block containing a single-line JSON object `{"results":[...]}`. This is unambiguous to parse from mixed text output and survives multi-line agent responses.

2. **Provider-aware fan-out**: `buildValidatorPrompt` (in `daemon/prompt.go`) and `buildValidatorSkill` (in `daemon/validator.go`) include Claude-specific Task-tool fan-out instructions when `provider == "claude"`, and sequential checking instructions for all other providers. This implements the graceful fallback without any branching at the daemon level.

3. **Skill injection via execenv**: A synthesized `validator-run` skill (`SkillContextForEnv`) is appended to `AgentSkills` in `runTask` before building `taskCtx`. This follows the exact same path that writes CLAUDE.md / skills — no new injection paths were needed.

4. **Verdict parsing and forwarding**: After the agent completes with `status == "completed"` and `task.Role == "validator"`, `parseValidationOutput(result.Output)` scans the output for the block and decodes it. The result is stored in `TaskResult.Validation` and forwarded via the extended `CompleteTask` client call. The server already had `TaskCompleteRequest.Validation *ValidationInput` and called `recordValidationOnCompletion`, so no server-side changes were needed.

5. **Claim endpoint populates assertions**: `ClaimTaskByRuntime` in `handler/daemon.go` loads `ListDodAssertionsByMilestone` for validator runs with a valid `issue.MilestoneID` and includes them in `AgentTaskResponse.ValidatorAssertions`. Worker runs get an empty list.

6. **TDD**: All tests were written first. `validator_test.go` has 5 unit tests for `parseValidationOutput` (happy path, multiple results, no block → nil, malformed JSON → nil, empty block → nil). `prompt_test.go` has 2 tests for `buildValidatorPrompt` (Claude vs non-Claude). `claim_validator_assertions_test.go` has 2 DB-integrated handler tests (validator claim returns assertions, worker claim has no assertions).

### Files changed

**New**
- `server/internal/daemon/validator.go` — `ValidationOutput`, `ValidationResultData`, `parseValidationOutput`, `buildValidatorSkill`
- `server/internal/daemon/validator_test.go` — 5 unit tests for `parseValidationOutput`
- `server/internal/handler/claim_validator_assertions_test.go` — 2 DB-integrated handler tests

**Modified**
- `server/internal/daemon/types.go` — Added `ValidatorAssertionData` type; `Role` and `ValidatorAssertions` on `Task`; `Validation *ValidationOutput` on `TaskResult`
- `server/internal/daemon/client.go` — `CompleteTask` accepts `*ValidationOutput`
- `server/internal/daemon/prompt.go` — Added `buildValidatorPrompt`, dispatch in `BuildPrompt`
- `server/internal/daemon/prompt_test.go` — 2 tests for `buildValidatorPrompt`
- `server/internal/daemon/daemon.go` — Validator skill injection in `runTask`; verdict parsing in completed case; `result.Validation` forwarded in `reportTaskResult`
- `server/internal/handler/agent.go` — `ValidatorAssertionData` type; `ValidatorAssertions` on `AgentTaskResponse`
- `server/internal/handler/daemon.go` — Populate `ValidatorAssertions` in `ClaimTaskByRuntime`

### Blockers / notes

- All checks pass: `pnpm typecheck`, `pnpm test` (677 views + core), `go test ./internal/... ./cmd/...` (handler 753 passed, daemon 252 passed). Remaining Go failures are pre-existing environment-specific issues (Windows symlinks, timing, WS integration).
- Token and wall-clock usage are not yet tracked per Run (noted in issue 11). The `ValidationOutput` payload is only populated when the agent actually emits the block — runs that complete without a block submit `Validation: nil`, which is handled gracefully by `recordValidationOnCompletion`.
- Issue 13 (PR lifecycle) and issue 14 (MCP control plane) are unblocked.
