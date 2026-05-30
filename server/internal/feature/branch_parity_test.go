package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multica-ai/multica/server/internal/feature"
)

// parityPool is initialized by TestMain when a DB is available.
var parityPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err == nil {
		if pingErr := pool.Ping(ctx); pingErr != nil {
			pool.Close()
			pool = nil
		}
	}
	if err != nil {
		pool = nil
	}
	parityPool = pool
	code := m.Run()
	if parityPool != nil {
		parityPool.Close()
	}
	os.Exit(code)
}

// sqlResolve calls the feature_resolve_branch SQL function — the SAME function
// the claim query (server/pkg/db/queries/agent.sql) uses to gate the branch.
// Calling the function directly (rather than a hand-copied mirror of its body)
// is what makes this a true parity test: any change to the function is seen
// here, and any change to Go's Resolve that diverges from it fails the test.
//
// Parameters mirror the function signature:
//   - $1 :: jsonb — issue.metadata (nullable)
//   - $2 :: uuid  — feature id, NULL when no feature (the feature-branch fallback)
//   - $3 :: text  — feature.branch_slug (nullable)
//   - $4 :: text  — workspace.issue_prefix (e.g. "MUL")
//   - $5 :: int   — issue.number
func sqlResolve(ctx context.Context, metadata map[string]any, featureID *string, branchSlug *string, issuePrefix string, issueNumber int) (string, error) {
	var metaJSON []byte
	if metadata != nil {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const q = `SELECT feature_resolve_branch($1::jsonb, $2::uuid, $3::text, $4::text, $5::int)`
	row := parityPool.QueryRow(ctx, q, metaJSON, featureID, branchSlug, issuePrefix, issueNumber)

	var result string
	if err := row.Scan(&result); err != nil {
		return "", err
	}
	return result, nil
}

func TestBranchResolverParityWithSQL(t *testing.T) {
	if parityPool == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	// A fixed feature UUID. The Go resolver's feature fallback uses
	// Feature.Identifier (the daemon sets it to the feature UUID), and the SQL
	// function's fallback uses feature_id::text — so the Go Feature.Identifier
	// must equal the UUID passed as featureID for the two to agree.
	const featureUUID = "11111111-1111-1111-1111-111111111111"

	cases := []struct {
		name         string
		metadata     map[string]any
		featPresent  bool
		branchSlug   *string
		issuePrefix  string
		issueNumber  int
	}{
		{
			name:        "feature nil → derived issue/<prefix>-<number>",
			metadata:    nil,
			featPresent: false,
			issuePrefix: "MUL",
			issueNumber: 487,
		},
		{
			name:        "feature present, branch_slug nil → feature/<feature uuid>",
			metadata:    nil,
			featPresent: true,
			branchSlug:  nil,
			issuePrefix: "MUL",
			issueNumber: 100,
		},
		{
			name:        "feature branch_slug set → feature/<slug>",
			metadata:    nil,
			featPresent: true,
			branchSlug:  ptr("auth-v2"),
			issuePrefix: "MUL",
			issueNumber: 300,
		},
		{
			name:        "metadata target_branch wins over feature branch_slug",
			metadata:    map[string]any{"target_branch": "issue/per-issue"},
			featPresent: true,
			branchSlug:  ptr("shared"),
			issuePrefix: "MUL",
			issueNumber: 500,
		},
		{
			name:        "metadata target_branch wins over feature with no branch_slug",
			metadata:    map[string]any{"target_branch": "issue/my-override"},
			featPresent: true,
			branchSlug:  nil,
			issuePrefix: "MUL",
			issueNumber: 400,
		},
		{
			name:        "metadata target_branch empty string → feature branch",
			metadata:    map[string]any{"target_branch": ""},
			featPresent: true,
			branchSlug:  nil,
			issuePrefix: "MUL",
			issueNumber: 600,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issue := feature.Issue{
				Identifier: tc.issuePrefix + "-" + fmt.Sprint(tc.issueNumber),
				Metadata:   tc.metadata,
			}

			var f *feature.Feature
			var featureID *string
			if tc.featPresent {
				f = &feature.Feature{Identifier: featureUUID, BranchSlug: tc.branchSlug}
				id := featureUUID
				featureID = &id
			}

			goBranch, _ := feature.Resolve(issue, f, nil)

			sqlBranch, err := sqlResolve(ctx, tc.metadata, featureID, tc.branchSlug, tc.issuePrefix, tc.issueNumber)
			if err != nil {
				t.Fatalf("sqlResolve: %v", err)
			}

			if goBranch != sqlBranch {
				t.Errorf("Go=%q SQL=%q — resolver drift detected", goBranch, sqlBranch)
			}
		})
	}
}
