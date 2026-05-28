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

// sqlResolve evaluates the SQL branch-resolution expression used in the claim
// query against the same inputs as Go's Resolve. NULLIF strips empty strings
// from metadata so both sides skip them consistently. The query accepts:
//   - $1 :: text   — feature.target_branch (nullable)
//   - $2 :: jsonb  — issue.metadata (nullable JSONB)
//   - $3 :: text   — issue identifier (e.g. "MUL-487")
func sqlResolve(ctx context.Context, featureTargetBranch *string, metadata map[string]any, identifier string) (string, error) {
	var metaJSON []byte
	if metadata != nil {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const q = `SELECT COALESCE(NULLIF($1::text,''), NULLIF(($2::jsonb)->>'target_branch',''), 'issue/' || $3::text)`
	row := parityPool.QueryRow(ctx, q, featureTargetBranch, metaJSON, identifier)

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

	cases := []struct {
		name       string
		issue      feature.IssueForBranch
		f          *feature.FeatureForBranch
	}{
		{
			name:  "feature nil → derived branch",
			issue: feature.IssueForBranch{Identifier: "MUL-487", Metadata: nil},
			f:     nil,
		},
		{
			name:  "feature TargetBranch nil, no metadata → derived branch",
			issue: feature.IssueForBranch{Identifier: "MUL-100", Metadata: map[string]any{}},
			f:     &feature.FeatureForBranch{TargetBranch: nil},
		},
		{
			name:  "feature TargetBranch set → shared branch",
			issue: feature.IssueForBranch{Identifier: "MUL-300", Metadata: nil},
			f:     &feature.FeatureForBranch{TargetBranch: ptr("feature/auth-v2")},
		},
		{
			name: "issue Metadata target_branch set → per-issue override",
			issue: feature.IssueForBranch{
				Identifier: "MUL-400",
				Metadata:   map[string]any{"target_branch": "issue/my-override"},
			},
			f: &feature.FeatureForBranch{TargetBranch: nil},
		},
		{
			name: "both set → feature wins",
			issue: feature.IssueForBranch{
				Identifier: "MUL-500",
				Metadata:   map[string]any{"target_branch": "issue/per-issue"},
			},
			f: &feature.FeatureForBranch{TargetBranch: ptr("feature/shared")},
		},
		{
			name: "Metadata target_branch empty string → derived branch",
			issue: feature.IssueForBranch{
				Identifier: "MUL-600",
				Metadata:   map[string]any{"target_branch": ""},
			},
			f: &feature.FeatureForBranch{TargetBranch: nil},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			goBranch, _ := feature.Resolve(tc.issue, tc.f)

			var featureTargetBranch *string
			if tc.f != nil {
				featureTargetBranch = tc.f.TargetBranch
			}

			sqlBranch, err := sqlResolve(ctx, featureTargetBranch, tc.issue.Metadata, tc.issue.Identifier)
			if err != nil {
				t.Fatalf("sqlResolve: %v", err)
			}

			if goBranch != sqlBranch {
				t.Errorf("Go=%q SQL=%q — resolver drift detected", goBranch, sqlBranch)
			}
		})
	}
}
