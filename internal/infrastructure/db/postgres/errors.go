package postgres

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"payloop/internal/lib"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return lib.DatabaseError{
			Code:    lib.NoResults,
			Message: "Not found",
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return lib.DatabaseError{
				Code:    lib.ForeignKeyViolation,
				Message: "Foreign constraint violation",
				Err:     err,
			}
		case "23505": // unique_violation
			return lib.DatabaseError{
				Code:    lib.UniqueKeyViolation,
				Message: "Unique constraint violation",
				Err:     err,
			}
		case "23502": // not_null_violation
			return lib.DatabaseError{
				Code:    lib.NotNullViolation,
				Message: "Not null violation",
				Err:     err,
			}
		case "42P01": // undefined_table
			return lib.DatabaseError{
				Code:    lib.UnknownTable,
				Message: "Table not found",
				Err:     err,
			}
		default:
			return lib.DatabaseError{
				Code:    lib.GenericError,
				Message: pgErr.Message,
				Err:     err,
			}
		}
	}

	return lib.DatabaseError{
		Code:    lib.GenericError,
		Message: err.Error(),
		Err:     err,
	}
}
