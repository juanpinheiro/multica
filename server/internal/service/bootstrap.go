package service

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	SingletonUserID    = "00000000-0000-0000-0000-000000000001"
	SingletonUserEmail = "local@multica"
	SingletonUserName  = "You"
)

// dbExecer is the minimal interface for running a parameterized SQL statement.
// Both *pgxpool.Pool and pgx.Tx satisfy it.
type dbExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// BootstrapSingletonUser ensures the implicit singleton user exists in the
// database. Idempotent: safe to call on every startup.
func BootstrapSingletonUser(ctx context.Context, db dbExecer) error {
	const q = `
        INSERT INTO "user" (id, name, email)
        VALUES ($1, $2, $3)
        ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(ctx, q, SingletonUserID, SingletonUserName, SingletonUserEmail)
	if err != nil {
		return err
	}
	slog.Info("singleton user bootstrapped", "user_id", SingletonUserID)
	return nil
}
