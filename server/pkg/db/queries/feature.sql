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
INSERT INTO feature (
    workspace_id, title, description, icon, status,
    lead_type, lead_id, priority, target_branch
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
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
    target_branch = sqlc.narg('target_branch'),
    updated_at = now()
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
