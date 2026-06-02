# GitHub App Setup — PR Merge Detection

Multica advances an Initiative from `in_review` to `done` when its PR is
merged. Two paths can observe the merge; choose the one that matches your
environment.

---

## Path A — GitHub webhook (lower latency, recommended for production)

Requires a public HTTPS endpoint that GitHub can reach.

### 1. Create a GitHub App

1. Go to **GitHub → Settings → Developer settings → GitHub Apps → New GitHub App**.
2. Set **Webhook URL** to `https://<your-host>/api/webhooks/github`.
3. Enable **Subscribe to events → Pull request**.
4. Set **Permissions → Pull requests → Read-only**.
5. Install the App on the target repository (or all repositories in the org).
6. Note the **App slug** (the part after `https://github.com/apps/`).
7. Generate and download a **webhook secret**.

### 2. Configure environment variables

```
GITHUB_APP_SLUG=<your-app-slug>
GITHUB_WEBHOOK_SECRET=<your-webhook-secret>
```

Restart the server. GitHub will now POST to `/api/webhooks/github` on every PR
event. When a PR linked to an `in_review` Initiative is merged, the server
advances it to `done` within seconds.

---

## Path B — poll-based fallback (local dev, no public endpoint required)

When `GITHUB_APP_SLUG` is not set (or the webhook is otherwise unreachable),
the server falls back to a poll-based detector that runs as a background
goroutine.

**No configuration required.** The poller starts automatically with the server.

### How it works

Every 60 seconds (backing off exponentially to 5 minutes when idle) the poller
queries the database for PRs in `merged` state that are linked to `in_review`
Initiatives, then calls the same `advanceFeaturesOnPRMerge` function the webhook
uses. Double-observation by both paths is safe — the function is idempotent.

### Timing

| Path       | Latency after merge            |
|------------|-------------------------------|
| Webhook    | Seconds (GitHub push latency)  |
| Poll       | Up to 60s (first tick); up to 5m when idle |

---

## Verifying the setup

After merging a PR linked to an `in_review` Initiative, confirm the Initiative
transitions to `done`:

```bash
# One-shot check via multica CLI
multica features list --status in_review

# Or watch the Mission Monitor at http://localhost:3000/<workspace>/features
```
