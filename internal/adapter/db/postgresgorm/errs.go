package postgresgorm

import (
	"errors"
	"fmt"
	errors2 "getpaidhq/internal/lib/errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"getpaidhq/internal/core/port"
)

// pgForeignKeyViolation is the SQLSTATE Postgres returns when a DELETE or
// UPDATE is blocked because the row is still referenced from another table.
const pgForeignKeyViolation = "23503"

// pgUniqueViolation is the SQLSTATE Postgres returns when an INSERT or
// UPDATE collides with a unique index.
const pgUniqueViolation = "23505"

// translateErr wraps gorm-specific errors as domain-level sentinels so
// callers can do `errors.Is(err, port.ErrNotFound)` without importing
// gorm. Every repo method that ends with First/Take/Last (i.e. expects
// exactly one row) MUST run its returned error through this helper.
//
// nil maps to nil — the helper is safe to call unconditionally on the
// error of a query.
//
// The original error is wrapped (%w) so callers that need the raw
// driver error still have access via errors.Unwrap / errors.As.
func translateErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", port.ErrNotFound, err)
	}
	return err
}

// asConflictOnFK converts a Postgres foreign-key violation (23503) into a
// typed ConflictError carrying msg, so handlers render a 409 with a clear,
// caller-supplied message instead of leaking the raw driver error as an
// opaque 400. The driver error is wrapped (%w via NewCustomError) so it stays
// in the chain for logs and errors.Is. Any other error — including nil — is
// returned unchanged.
func asConflictOnFK(err error, msg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return errors2.NewCustomError(errors2.ConflictError, msg, err)
	}
	return err
}

// asConflictOnUnique converts a Postgres unique violation (23505) into a
// typed ConflictError carrying msg, so handlers render a 409 with a clear,
// caller-supplied message instead of leaking the raw driver error. Same
// wrapping contract as asConflictOnFK.
func asConflictOnUnique(err error, msg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return errors2.NewCustomError(errors2.ConflictError, msg, err)
	}
	return err
}
