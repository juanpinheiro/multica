-- name: ListFeatureResources :many
SELECT * FROM feature_resource
WHERE feature_id = $1
ORDER BY position ASC, created_at ASC;

-- name: ListFeatureResourcesForProjects :many
SELECT * FROM feature_resource
WHERE feature_id = ANY(sqlc.arg('feature_ids')::uuid[])
ORDER BY feature_id, position ASC, created_at ASC;

-- name: GetFeatureResource :one
SELECT * FROM feature_resource
WHERE id = $1;

-- name: GetFeatureResourceInWorkspace :one
SELECT * FROM feature_resource
WHERE id = $1 AND workspace_id = $2;

-- name: CreateFeatureResource :one
INSERT INTO feature_resource (
    feature_id, workspace_id, resource_type, resource_ref, label, position, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: DeleteFeatureResource :exec
DELETE FROM feature_resource WHERE id = $1;

-- name: CountFeatureResources :one
SELECT count(*) FROM feature_resource WHERE feature_id = $1;

-- name: GetFeatureResourceCounts :many
SELECT feature_id, count(*)::bigint AS resource_count
FROM feature_resource
WHERE feature_id = ANY(sqlc.arg('feature_ids')::uuid[])
GROUP BY feature_id;
