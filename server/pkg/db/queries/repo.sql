-- name: CreateRepo :one
INSERT INTO repo (
    workspace_id, name, remote_url, local_path, default_branch
) VALUES (
    $1, $2, $3, $4, COALESCE(sqlc.narg('default_branch'), 'main')
) RETURNING *;

-- name: ListReposInWorkspace :many
SELECT * FROM repo
WHERE workspace_id = $1
ORDER BY name ASC;

-- name: GetRepoInWorkspace :one
SELECT * FROM repo
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteRepo :execrows
-- Defense-in-depth: workspace_id is a SQL-layer tenant guard. See DeleteIssue.
-- Returns the affected row count so the handler can distinguish a real delete
-- from a no-op (unknown id / cross-workspace) and answer 404 instead of a
-- misleading 204 (the #1661 silent-zero-rows trap).
DELETE FROM repo WHERE id = $1 AND workspace_id = $2;
