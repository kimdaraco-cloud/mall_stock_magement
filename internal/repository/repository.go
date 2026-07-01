// @ai-modified 2026-07-02 add shared repository errors and DB interface
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// DB is the subset of pgxpool.Pool / pgx.Tx that repositories use, so the
// same repository methods work inside and outside a transaction.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
