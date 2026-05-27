package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBootstrapSingletonUser_CreatesUser(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	// Delete the singleton user if it exists (cleanup state from previous runs)
	pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, SingletonUserID)
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, SingletonUserID)
	})

	if err := BootstrapSingletonUser(ctx, pool); err != nil {
		t.Fatalf("BootstrapSingletonUser: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM "user" WHERE id = $1`, SingletonUserID).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("want 1 singleton user, got %d", count)
	}
}

func TestBootstrapSingletonUser_Idempotent(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, SingletonUserID)
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, SingletonUserID)
	})

	if err := BootstrapSingletonUser(ctx, pool); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if err := BootstrapSingletonUser(ctx, pool); err != nil {
		t.Fatalf("second bootstrap (idempotency): %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM "user" WHERE id = $1`, SingletonUserID).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("want 1 singleton user after 2 bootstraps, got %d", count)
	}
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Skipping DB test: %v\n", err)
		t.Skip("database not available")
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skip("database not reachable")
	}
	t.Cleanup(pool.Close)
	return pool
}
