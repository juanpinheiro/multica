-- name: CreateDodAssertion :one
INSERT INTO dod_assertion (workspace_id, feature_id, milestone_id, text, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListDodAssertionsByMilestone :many
SELECT * FROM dod_assertion
WHERE milestone_id = $1
ORDER BY position ASC;

-- name: ListDodAssertionsByFeature :many
SELECT * FROM dod_assertion
WHERE feature_id = $1
ORDER BY position ASC;

-- name: ListDodAssertionsByIssue :many
-- The per-Issue view of the Definition of Done: the assertions of the Issue's
-- Milestone. An Issue with no Milestone has no derived Acceptance Criteria.
SELECT a.* FROM dod_assertion a
JOIN issue i ON i.milestone_id = a.milestone_id
WHERE i.id = $1
ORDER BY a.position ASC;

-- name: CreateDodAssertionResult :one
INSERT INTO dod_assertion_result (workspace_id, assertion_id, run_id, passed, detail)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: MaxMilestoneValidationFailures :one
-- The worst single Milestone's validation-failure count in an Initiative: for
-- each Milestone, how many distinct validator Runs recorded at least one failing
-- verdict; returns the maximum across the Initiative (0 when none). Feeds the
-- failure-tolerance arm of the Tripwire/Budget safety net.
SELECT COALESCE(MAX(failures), 0)::bigint FROM (
    SELECT a.milestone_id, count(DISTINCT r.run_id) AS failures
    FROM dod_assertion_result r
    JOIN dod_assertion a ON a.id = r.assertion_id
    WHERE a.feature_id = $1 AND r.passed = false
    GROUP BY a.milestone_id
) per_milestone;

-- name: ListLatestDodResultsByMilestone :many
-- The latest verdict per assertion for a Milestone (newest wins). Feeds the pure
-- dod.MilestoneSatisfied evaluation and the monitor's pass/fail display.
SELECT DISTINCT ON (r.assertion_id) r.*
FROM dod_assertion_result r
JOIN dod_assertion a ON a.id = r.assertion_id
WHERE a.milestone_id = $1
ORDER BY r.assertion_id, r.created_at DESC;
