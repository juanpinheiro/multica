-- name: CreateDecisionLogEntry :one
-- Written by the handler when a retrospective Run completes, one row per decision
-- the Agent recorded. The Decision Log view lists these via ListDecisionLogByFeature.
INSERT INTO decision_log (workspace_id, feature_id, run_id, title, decision, learning, adr_refs, context_terms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListDecisionLogByFeature :many
-- Returns an Initiative's recorded decisions, newest first — the Decision Log view.
SELECT * FROM decision_log
WHERE feature_id = $1
ORDER BY created_at DESC;

-- name: ListDecisionLogByWorkspace :many
-- Returns a workspace's recorded decisions across every Initiative, newest first.
-- Backs the cross-Initiative Decisions view.
SELECT * FROM decision_log
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
