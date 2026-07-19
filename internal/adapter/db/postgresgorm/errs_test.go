package postgresgorm

import (
	"errors"
	"fmt"
	errors2 "getpaidhq/internal/lib/errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// asConflictOnFK must turn a Postgres foreign-key violation (SQLSTATE 23503)
// into a typed ConflictError carrying the supplied message, so handlers render
// a 409 instead of leaking the raw driver error as an opaque 400.
func TestAsConflictOnFK(t *testing.T) {
	const msg = "Cannot delete a product that has existing orders."

	t.Run("foreign-key violation becomes a ConflictError", func(t *testing.T) {
		fk := &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}
		got := asConflictOnFK(fk, msg)

		var ce errors2.CustomError
		require.True(t, errors.As(got, &ce), "want a lib.CustomError, got %T", got)
		require.Equal(t, errors2.ConflictError, ce.Type)
		require.Equal(t, msg, ce.Message)
		require.ErrorIs(t, got, fk, "underlying driver error must remain in the chain")
	})

	t.Run("foreign-key violation wrapped by GORM is still detected", func(t *testing.T) {
		fk := &pgconn.PgError{Code: "23503"}
		got := asConflictOnFK(fmt.Errorf("gorm: %w", fk), msg)

		var ce errors2.CustomError
		require.True(t, errors.As(got, &ce))
		require.Equal(t, errors2.ConflictError, ce.Type)
	})

	t.Run("other Postgres errors pass through unchanged", func(t *testing.T) {
		other := &pgconn.PgError{Code: "23505"} // unique_violation
		require.Equal(t, error(other), asConflictOnFK(other, msg))
	})

	t.Run("non-pg errors pass through unchanged", func(t *testing.T) {
		plain := errors.New("boom")
		require.Equal(t, plain, asConflictOnFK(plain, msg))
	})

	t.Run("nil passes through", func(t *testing.T) {
		require.NoError(t, asConflictOnFK(nil, msg))
	})
}
