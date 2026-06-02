-- name: ListFeatures :many
SELECT * FROM feature
WHERE workspace_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR priority = sqlc.narg('priority'))
ORDER BY created_at DESC;

-- name: GetFeature :one
SELECT * FROM feature
WHERE id = $1;

-- name: GetFeatureInWorkspace :one
SELECT * FROM feature
WHERE id = $1 AND workspace_id = $2;

-- name: CreateFeature :one
-- Mode and the budget/failure-tolerance fields are optional: omit them and the
-- DB defaults apply (hitl, no caps, tolerance 3) via the COALESCE fallbacks.
INSERT INTO feature (
    workspace_id, title, description, icon, status,
    lead_type, lead_id, priority, branch_slug,
    mode, budget_tokens, budget_runs, budget_seconds, failure_tolerance
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    COALESCE(sqlc.narg('mode')::text, 'hitl'),
    COALESCE(sqlc.narg('budget_tokens')::bigint, 0),
    COALESCE(sqlc.narg('budget_runs')::int, 0),
    COALESCE(sqlc.narg('budget_seconds')::bigint, 0),
    COALESCE(sqlc.narg('failure_tolerance')::int, 3)
) RETURNING *;

-- name: UpdateFeature :one
UPDATE feature SET
    title = COALESCE(sqlc.narg('title'), title),
    description = sqlc.narg('description'),
    icon = sqlc.narg('icon'),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    lead_type = sqlc.narg('lead_type'),
    lead_id = sqlc.narg('lead_id'),
    branch_slug = sqlc.narg('branch_slug'),
    mode = COALESCE(sqlc.narg('mode')::text, mode),
    budget_tokens = COALESCE(sqlc.narg('budget_tokens')::bigint, budget_tokens),
    budget_runs = COALESCE(sqlc.narg('budget_runs')::int, budget_runs),
    budget_seconds = COALESCE(sqlc.narg('budget_seconds')::bigint, budget_seconds),
    failure_tolerance = COALESCE(sqlc.narg('failure_tolerance')::int, failure_tolerance),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetIssueFeatureStatus :one
-- Returns the Initiative (feature) id and status for an issue, when the issue
-- belongs to one. Used by the execution plane to advance Initiative status as
-- its Runs claim and complete.
SELECT f.id AS feature_id, f.status
FROM issue i
JOIN feature f ON i.feature_id = f.id
WHERE i.id = $1;

-- name: SetFeatureStatus :one
-- Focused status write for the Initiative status state machine. Callers must
-- validate the transition via internal/initiative before calling.
UPDATE feature SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteFeature :exec
-- Defense-in-depth: workspace_id is a SQL-layer tenant guard. See DeleteIssue.
DELETE FROM feature WHERE id = $1 AND workspace_id = $2;

-- name: CountIssuesByFeature :one
SELECT count(*) FROM issue
WHERE feature_id = $1;

-- name: CountNonDoneFeatureSiblings :one
-- Counts issues in the same feature that have not yet reached `done`,
-- excluding the issue that just transitioned (already persisted as done).
-- Returns 0 when every sibling is done — the caller then fires the
-- feature-ready-for-review inbox notification.
SELECT count(*)::bigint FROM issue
WHERE feature_id = $1 AND id != $2 AND status != 'done';

-- name: GetFeatureOpenPR :one
-- Returns the first open or draft PR linked to any issue under the feature.
-- Used to include a PR link in the feature-ready-for-review notification.
SELECT gpr.id, gpr.pr_number, gpr.html_url, gpr.title
FROM github_pull_request gpr
JOIN issue_pull_request ipr ON ipr.pull_request_id = gpr.id
JOIN issue i ON i.id = ipr.issue_id
WHERE i.feature_id = $1 AND gpr.state IN ('open', 'draft')
ORDER BY gpr.pr_created_at ASC
LIMIT 1;

-- name: GetFeatureIssueStats :many
SELECT feature_id,
       count(*)::bigint AS total_count,
       count(*) FILTER (WHERE status IN ('done', 'cancelled'))::bigint AS done_count
FROM issue
WHERE feature_id = ANY(sqlc.arg('feature_ids')::uuid[])
GROUP BY feature_id;

-- name: ListFeatureIssueSummaries :many
-- Returns minimal issue data for cross-repo brief injection.
-- Used by the daemon claim handler to build the cross-repo context section
-- when a feature spans multiple repos.
SELECT i.id, i.title, i.number, i.repo_id,
       COALESCE(r.name, '') AS repo_name
FROM issue i
LEFT JOIN repo r ON r.id = i.repo_id AND r.workspace_id = i.workspace_id
WHERE i.feature_id = $1
ORDER BY i.number ASC;

-- name: ListInReviewIssuesWithMergedPRs :many
-- Returns issues belonging to in_review Initiatives that have at least one
-- merged linked PR. Used by the PR-merge poller to advance stale Initiatives
-- that were not caught by the GitHub webhook (local dev without webhook endpoint).
SELECT DISTINCT i.*
FROM issue i
JOIN feature f ON f.id = i.feature_id
JOIN issue_pull_request ipr ON ipr.issue_id = i.id
JOIN github_pull_request gpr ON gpr.id = ipr.pull_request_id
WHERE f.status = 'in_review'
  AND gpr.state = 'merged';
