-- name: CreateHandoff :one
-- Called by the handler when a worker Run (role=worker) completes an Issue.
-- The Orchestrator reads these rows via ListHandoffsByIssue on wake.
INSERT INTO handoff (workspace_id, issue_id, run_id, done, left_undone, commands, discoveries)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListHandoffsByIssue :many
-- Returns all Handoffs for an Issue ordered oldest-first so the caller can
-- pass the slice directly to handoff.LatestState.
SELECT * FROM handoff
WHERE issue_id = $1
ORDER BY created_at ASC;

-- name: GetLatestHandoffByIssue :one
-- Fast path for callers that only need the most recent Handoff.
SELECT * FROM handoff
WHERE issue_id = $1
ORDER BY created_at DESC
LIMIT 1;
