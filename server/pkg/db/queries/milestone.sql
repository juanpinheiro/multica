-- name: ListMilestonesByWorkspace :many
SELECT * FROM milestone
WHERE workspace_id = $1
ORDER BY feature_id, position ASC;

-- name: ListMilestonesByFeature :many
SELECT * FROM milestone
WHERE feature_id = $1
ORDER BY position ASC;

-- name: GetMilestone :one
SELECT * FROM milestone
WHERE id = $1;

-- name: CreateMilestone :one
INSERT INTO milestone (
    workspace_id, feature_id, title, position, validation_status
) VALUES (
    $1, $2, $3, $4, COALESCE(sqlc.narg('validation_status'), 'pending')
) RETURNING *;

-- name: UpdateMilestone :one
-- Control-plane edit (MCP / UI): title and ordering. validation_status is owned
-- by the execution plane (the validator Run) and is set via
-- SetMilestoneValidationStatus, not here.
UPDATE milestone SET
    title = COALESCE(sqlc.narg('title'), title),
    position = COALESCE(sqlc.narg('position')::int, position),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountMilestonesByFeature :one
SELECT count(*) FROM milestone
WHERE feature_id = $1;

-- name: SetMilestoneValidationStatus :one
-- Focused validation-status write, driven by the validator Run (issue 09).
-- 'passed' opens the claim gate for the next Milestone's Issues.
UPDATE milestone SET validation_status = $2, updated_at = now()
WHERE id = $1
RETURNING *;
