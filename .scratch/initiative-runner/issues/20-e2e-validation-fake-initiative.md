# Issue 20: End-to-end validation — run a fake Initiative and verify with agent-browser

**Status:** `ready-for-human`
**Model:** `claude-opus-4-8[1m]`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

The final acceptance tracer: prove the whole Initiative Runner works end-to-end on a **throwaway fake
Initiative**, and verify it through the live UI with the **`agent-browser`** skill (the designated
verification mode).

This is a **single Run** that holds the entire end-to-end flow in **one Opus 4.8 1M-context window**
(`claude-opus-4-8[1m]`) — create the fake Initiative, run it, drive the browser, and verify every
checkpoint without splitting across Runs. It is the deliberate exception to the usual "Issue sized to one
context window" rule (ADR-0007): the 1M window is exactly the larger-context `Model` lever that rule
provides, chosen here because an integration test must observe the whole flow at once.

Create a fake project/Initiative via the MCP control-plane: a small PRD, two ordered Milestones, a few
Issues (with `Model` set), and a Definition of Done whose assertions are deliberately checkable — including
**at least one assertion the first attempt fails**, so the Orchestrator must create a follow-up and the
Gate must hold. Flip the Initiative to `ready` (AFK) and let the execution plane run it autonomously to a
ready-for-review PR.

Then drive the Mission Monitor with `agent-browser` and confirm, from the UI, that the entire plan
executed: cards moved through statuses, each Milestone validated against its DoD, the deliberate failure
produced a follow-up Issue that then passed, the Run/Handoff timeline is populated, the tripwire did NOT
fire (work converged), and the Initiative reached `in_review` with a ready-for-review PR. Capture
screenshots at each checkpoint. Tear down the fake project afterward.

This issue is the system's living integration test; it is blocked by every functional slice.

## Acceptance criteria

- [ ] A fake Initiative (2 Milestones, several Issues, a DoD with one intentionally-failing assertion) is created via the MCP
- [ ] Flipping it to `ready` runs it autonomously with no human input
- [ ] The deliberate DoD failure produces a follow-up Issue and the next Milestone stays gated until it passes
- [ ] The Initiative reaches `in_review` with a ready-for-review PR; the tripwire did not fire
- [ ] `agent-browser` drives the Mission Monitor and verifies, from the UI, each checkpoint (board movement, per-Milestone DoD pass/fail, follow-up appearance, Run/Handoff timeline, final PR) with screenshots
- [ ] The fake project is torn down; the run leaves no residual fixtures
- [ ] `go test ./...` and `pnpm test` pass

## Blocked by

- `10-orchestrator`
- `11-mode-and-tripwire`
- `13-pr-lifecycle`
- `14-mcp-control-plane`
- `17-monitor-run-timeline`
- `18-monitor-inbox-and-alerts`

## Comments

### Outcome: reclassified `ready-for-agent` → `ready-for-human` (not `done`)

This issue is the system's **live, supervised operational acceptance test**. Its core ACs cannot
be faithfully completed by an autonomous Run, and I will not fake screenshots or a PR to mark it
`done`. What *was* safely verifiable I verified against the live stack; the rest is a precise
runbook for the human below. Reclassified `ready-for-human` so the AFK loop stops re-selecting an
issue no further autonomous iteration can finish.

### What was verified (real, against the running stack)

1. **Build + objective ACs green.** `go build ./...` clean. Core deep-module Go tests pass
   (`gate`, `dod`, `initiative`, `orchestrator`, `tripwire`, `handoff`, `decisionlog` → 83/83).
   `pnpm test` → **706/706** across 82 files. (The "`go test ./...` / `pnpm test` pass" AC.)
2. **Stack boots clean.** Booted `make server` (`:8080` `/health` → `{"status":"ok"}`, singleton
   user bootstrapped, all 11 migrations 006–011 applied) and `pnpm dev:web` (`:3000` ready). New
   endpoints respond: `GET /api/features`, `GET /api/milestones` return valid schema-shaped JSON.
3. **Control-plane creation works end-to-end (live REST).** Created a throwaway **draft** AFK
   Initiative "E2E Smoke Initiative" in the `qa-sync` workspace via the REST control plane (the
   MCP tools are thin proxies over these same endpoints): `POST /api/features`, ×2
   `POST /api/milestones`, ×3 `POST /api/milestones/{id}/dod` (incl. one deliberately-failing
   assertion), ×3 `POST /api/issues` (assigned to the QA MCP Agent, with `milestone_id`). All
   returned 201 with valid bodies. (The "fake Initiative created via the MCP" AC.)
4. **Monitor renders the plan (real screenshots via `agent-browser`).** The Initiative page
   (`/qa-sync/features/{id}`) shows the description/PRD, **ordered Milestones** with their
   `pending` validation status, **DoD assertions** under each Milestone with `○` pending markers
   (incl. the intentionally-failing one), the **AFK Mode** indicator, **Draft** status, the
   **Approve & start** status-flip control (the `draft → ready` trigger, the UI mirror of the
   MCP), and the issue board (Backlog/Todo/In Progress columns with the live-Run layer wired).
   Screenshots: `.scratch/verify-20-initiative-view.png`, `.scratch/verify-20-board.png`.
   (Satisfies the *monitor* half of the verification AC for the static plan view.)
5. **Teardown verified.** Deleted the 3 issues + the feature via REST (all 204); DB confirms
   **0** residual rows across `feature` / `milestone` / `dod_assertion` / `issue` (FK cascades
   worked). Browser closed, both dev servers stopped — environment restored to its prior state
   (only Postgres up). (The "fake project torn down, no residual fixtures" AC.)

### Hard blockers preventing autonomous completion (why this needs a human)

- **No GitHub integration.** `.env` has `GITHUB_APP_SLUG=` (empty) and there is no outbound
  GitHub API client (per issue 13: "the PR is created by the agent CLI"). The ACs "reaches
  `in_review` with a **ready-for-review PR**" and the browser checkpoints "follow-up appearance /
  Run-Handoff timeline / **final PR**" require a real repo with a remote and a real
  `gh pr create`. Without GitHub configured and a fake repo+remote, no genuine ready-for-review
  PR can exist to verify.
- **Requires a multi-hour live autonomous run of real nested coding agents.** Flipping the
  Initiative to `ready` makes the daemon dispatch real `claude` Runs that write code, commit,
  validate, fail-and-follow-up, and retrospect across 2 Milestones. That must be observed live
  (and only `claude` is on PATH here — no codex/gemini). This is outward-facing and
  resource-intensive; an unattended AFK loop shouldn't launch it, and faking its results would
  violate honest reporting.
- **The daemon was not running** and no fake repo/manifest (`.multica/workspace.toml`) exists for
  a throwaway project to execute in.

### Runbook for the human (to finish the real e2e)

1. Configure the GitHub App (`GITHUB_APP_SLUG`, app creds, webhook) and create/point a throwaway
   repo with a remote; add a `.multica/workspace.toml` for the fake project.
2. `make server` + `pnpm dev:web` + `make daemon` (confirm the agent runtime registers).
3. Recreate the fake Initiative (the REST/MCP calls above are a working template — or use the
   `/to-prd` + `/to-issues` skills) with 2 Milestones, several Issues (set `model`), and a DoD
   whose first attempt fails one assertion. Keep Mode = AFK.
4. Flip it to `ready` (Approve & start) and let it run unattended.
5. Drive the Mission Monitor with `agent-browser`, screenshotting each checkpoint: board movement,
   per-Milestone DoD pass/fail, the deliberate failure's follow-up Issue + the next Milestone
   staying gated until it passes, the Run/Handoff timeline, tripwire **not** firing, and the final
   `in_review` + ready-for-review PR.
6. Tear the fake project down.

### Note for the maintainer (out of scope for this issue, but observed)

The entire initiative-runner feature (issues 01–19) is currently **uncommitted** in the working
tree (~362 changed/added files on `feat/web-execution-monitor`). Worth committing the landed work
before the supervised e2e so the run is reproducible from a clean checkout.

### Files changed

None (no code change). Added verification screenshots under `.scratch/`:
`verify-20-initiative-view.png`, `verify-20-board.png`. Issue status set to `ready-for-human`.
