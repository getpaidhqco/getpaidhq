package postgrespgx

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// pgForeignKeyViolation is the SQLSTATE Postgres returns when a DELETE or
// UPDATE is blocked because the row is still referenced from another table.
const pgForeignKeyViolation = "23503"

// pgUniqueViolation is the SQLSTATE Postgres returns when an INSERT or UPDATE
// collides with a unique index.
const pgUniqueViolation = "23505"

// translateErr maps pgx-specific errors onto domain-level sentinels so callers
// can do `errors.Is(err, port.ErrNotFound)` without importing pgx — the
// pgx-side mirror of the gorm adapter's translateErr. Every method that does a
// single-row QueryRow().Scan(...) MUST run its returned error through this.
//
// nil maps to nil. The original error is wrapped (%w) so callers needing the
// raw driver error still have access via errors.Unwrap / errors.As.
func translateErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %w", port.ErrNotFound, err)
	}
	return err
}

// asConflictOnFK converts a Postgres foreign-key violation (23503) into a typed
// ConflictError carrying msg, so handlers render a 409 with a clear message
// instead of leaking the raw driver error. The driver error stays in the chain
// (%w via NewCustomError). Any other error — including nil — is returned
// unchanged. Identical contract to the gorm adapter.
func asConflictOnFK(err error, msg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return lib.NewCustomError(lib.ConflictError, msg, err)
	}
	return err
}

// asConflictOnUnique converts a Postgres unique violation (23505) into a typed
// ConflictError carrying msg. Same wrapping contract as asConflictOnFK.
func asConflictOnUnique(err error, msg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return lib.NewCustomError(lib.ConflictError, msg, err)
	}
	return err
}
