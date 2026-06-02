-- name: ListIssues :many
SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
       i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
       i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at, i.updated_at, i.number, i.feature_id, i.milestone_id, i.metadata
FROM issue i
WHERE i.workspace_id = $1
  AND (sqlc.narg('status')::text IS NULL OR i.status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR i.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR i.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('assignee_ids')::uuid[] IS NULL OR i.assignee_id = ANY(sqlc.narg('assignee_ids')::uuid[]))
  AND (sqlc.narg('creator_id')::uuid IS NULL OR i.creator_id = sqlc.narg('creator_id'))
  AND (sqlc.narg('feature_id')::uuid IS NULL OR i.feature_id = sqlc.narg('feature_id'))
  AND (sqlc.narg('scheduled')::bool IS NULL OR (i.start_date IS NOT NULL OR i.due_date IS NOT NULL))
  AND (sqlc.narg('metadata_filter')::jsonb IS NULL OR i.metadata @> sqlc.narg('metadata_filter')::jsonb)
ORDER BY i.position ASC, i.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetIssue :one
SELECT * FROM issue
WHERE id = $1;

-- name: GetIssueInWorkspace :one
SELECT * FROM issue
WHERE id = $1 AND workspace_id = $2;

-- name: CreateIssue :one
INSERT INTO issue (
    workspace_id, title, description, status, priority,
    assignee_type, assignee_id, creator_type, creator_id,
    parent_issue_id, position, start_date, due_date, number, feature_id,
    repo_id, milestone_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetIssueByNumber :one
SELECT * FROM issue
WHERE workspace_id = $1 AND number = $2;

-- name: CountNonDoneMilestoneSiblings :one
-- Counts issues in the same Milestone that have not yet reached `done`,
-- excluding the issue that just transitioned. Returns 0 when the Milestone's
-- work is complete and the validator Run may be dispatched.
SELECT count(*)::bigint FROM issue
WHERE milestone_id = $1 AND id != $2 AND status != 'done';

-- name: CreateDodFollowUpIssue :one
-- A worker Issue created inside a Milestone when its DoD validation fails, so
-- the Initiative self-heals. Assigned to the agent that owns the work; status
-- 'backlog' keeps the Milestone non-done until the follow-up is resolved.
INSERT INTO issue (
    workspace_id, feature_id, milestone_id, title, description, status, priority,
    assignee_type, assignee_id, creator_type, creator_id, number, position
) VALUES (
    $1, $2, $3, $4, $5, 'backlog', 'high',
    'agent', $6, 'agent', $6, $7, 0
) RETURNING *;

-- name: UpdateIssue :one
UPDATE issue SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    assignee_type = sqlc.narg('assignee_type'),
    assignee_id = sqlc.narg('assignee_id'),
    position = COALESCE(sqlc.narg('position'), position),
    start_date = sqlc.narg('start_date'),
    due_date = sqlc.narg('due_date'),
    parent_issue_id = sqlc.narg('parent_issue_id'),
    feature_id = sqlc.narg('feature_id'),
    repo_id = sqlc.narg('repo_id'),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateIssueStatus :one
-- Workspace_id in the WHERE clause is a SQL-layer tenant guard; see DeleteIssue.
UPDATE issue SET
    status = $2,
    updated_at = now()
WHERE id = $1 AND workspace_id = $3
RETURNING *;

-- name: CreateIssueWithOrigin :one
INSERT INTO issue (
    workspace_id, title, description, status, priority,
    assignee_type, assignee_id, creator_type, creator_id,
    parent_issue_id, position, start_date, due_date, number, feature_id,
    repo_id, origin_type, origin_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
    sqlc.narg('repo_id'), sqlc.narg('origin_type'), sqlc.narg('origin_id')
) RETURNING *;

-- name: LockIssueDuplicateKey :exec
SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0));

-- name: FindActiveDuplicateIssue :one
SELECT * FROM issue
WHERE workspace_id = $1
  AND status NOT IN ('done', 'cancelled')
  AND feature_id IS NOT DISTINCT FROM sqlc.arg('feature_id')::uuid
  AND parent_issue_id IS NOT DISTINCT FROM sqlc.arg('parent_issue_id')::uuid
  AND lower(btrim(regexp_replace(title, '[[:space:]]+', ' ', 'g'))) = sqlc.arg('normalized_title')
ORDER BY created_at ASC
LIMIT 1;

-- name: DeleteIssue :exec
-- Defense-in-depth: the workspace_id predicate makes the tenant invariant a
-- SQL-layer guarantee rather than a handler-layer one. Handler loaders
-- (loadIssueForUser / GetIssueInWorkspace) already enforce membership today,
-- but a future loader bypass or a new caller skipping the loader would be
-- silently catastrophic without this guard. See incident #1661.
DELETE FROM issue WHERE id = $1 AND workspace_id = $2;

-- name: ListOpenIssues :many
SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
       i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
       i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at, i.updated_at, i.number, i.feature_id, i.metadata
FROM issue i
WHERE i.workspace_id = $1
  AND i.status NOT IN ('done', 'cancelled')
  AND (sqlc.narg('priority')::text IS NULL OR i.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR i.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('assignee_ids')::uuid[] IS NULL OR i.assignee_id = ANY(sqlc.narg('assignee_ids')::uuid[]))
  AND (sqlc.narg('creator_id')::uuid IS NULL OR i.creator_id = sqlc.narg('creator_id'))
  AND (sqlc.narg('feature_id')::uuid IS NULL OR i.feature_id = sqlc.narg('feature_id'))
  AND (sqlc.narg('metadata_filter')::jsonb IS NULL OR i.metadata @> sqlc.narg('metadata_filter')::jsonb)
ORDER BY i.position ASC, i.created_at DESC;

-- name: CountIssues :one
SELECT count(*) FROM issue i
WHERE i.workspace_id = $1
  AND (sqlc.narg('status')::text IS NULL OR i.status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR i.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR i.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('assignee_ids')::uuid[] IS NULL OR i.assignee_id = ANY(sqlc.narg('assignee_ids')::uuid[]))
  AND (sqlc.narg('creator_id')::uuid IS NULL OR i.creator_id = sqlc.narg('creator_id'))
  AND (sqlc.narg('feature_id')::uuid IS NULL OR i.feature_id = sqlc.narg('feature_id'))
  AND (sqlc.narg('scheduled')::bool IS NULL OR (i.start_date IS NOT NULL OR i.due_date IS NOT NULL))
  AND (sqlc.narg('metadata_filter')::jsonb IS NULL OR i.metadata @> sqlc.narg('metadata_filter')::jsonb);

-- name: ListChildIssues :many
SELECT * FROM issue
WHERE parent_issue_id = $1
ORDER BY position ASC, created_at DESC;

-- name: GetIssueByOrigin :one
-- Finds the issue stamped with a specific (origin_type, origin_id) pair.
-- Used by quick-create completion to deterministically locate the issue
-- produced by a given agent_task_queue.id — robust against concurrent
-- issue creates by the same agent (assignment task + quick-create both
-- running with max_concurrent_tasks > 1).
SELECT * FROM issue
WHERE workspace_id = $1
  AND origin_type = $2
  AND origin_id = $3
LIMIT 1;

-- name: CountCreatedIssueAssignees :many
-- Count agent assignees on issues in this workspace.
SELECT
  assignee_type,
  assignee_id,
  COUNT(*)::bigint as frequency
FROM issue
WHERE workspace_id = $1
  AND creator_id = $2
  AND assignee_type IS NOT NULL
  AND assignee_id IS NOT NULL
GROUP BY assignee_type, assignee_id;

-- name: ChildIssueProgress :many
SELECT parent_issue_id,
       COUNT(*)::bigint AS total,
       COUNT(*) FILTER (WHERE status IN ('done', 'cancelled'))::bigint AS done
FROM issue
WHERE workspace_id = $1
  AND parent_issue_id IS NOT NULL
GROUP BY parent_issue_id;

-- SearchIssues: moved to handler (dynamic SQL for multi-word search support).

-- name: SetIssueMetadataKey :one
-- Atomically sets a single key in the issue's metadata JSONB. The
-- workspace_id filter is the authorization gate — handler resolves the
-- issue first so this is also the tenant check.
UPDATE issue SET
    metadata = jsonb_set(metadata, ARRAY[sqlc.arg('key')::text], sqlc.arg('value')::jsonb),
    updated_at = now()
WHERE id = sqlc.arg('id') AND workspace_id = sqlc.arg('workspace_id')
RETURNING *;

-- name: DeleteIssueMetadataKey :one
-- Atomically removes a single key from the issue's metadata JSONB.
-- Deleting a missing key is a no-op (still returns the row).
UPDATE issue SET
    metadata = metadata - sqlc.arg('key')::text,
    updated_at = now()
WHERE id = sqlc.arg('id') AND workspace_id = sqlc.arg('workspace_id')
RETURNING *;

-- name: MarkIssueFirstExecuted :one
-- Flips first_executed_at from NULL to now() atomically. Returns the row if
-- this was the first time the issue was executed; no rows otherwise. The
-- analytics issue_executed event fires exactly when this returns a row —
-- retries and re-assignments hit the WHERE clause and no-op.
UPDATE issue
SET first_executed_at = now()
WHERE id = $1 AND first_executed_at IS NULL
RETURNING id, workspace_id, creator_type, creator_id, first_executed_at;
