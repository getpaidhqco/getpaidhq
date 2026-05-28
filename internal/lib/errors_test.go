package lib

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomError_Error(t *testing.T) {
	t.Run("formats with type, message, and wrapped err", func(t *testing.T) {
		ce := NewCustomError(NotFoundError, "subscription missing", errors.New("under"))
		assert.Equal(t, "not_found: subscription missing: under", ce.Error())
	})

	t.Run("omits trailing colon when no wrapped err", func(t *testing.T) {
		ce := NewCustomError(ValidationError, "currency required", nil)
		assert.Equal(t, "validation_error: currency required", ce.Error())
	})
}

func TestCustomError_Unwrap(t *testing.T) {
	t.Run("errors.Is sees the wrapped sentinel", func(t *testing.T) {
		// This is the load-bearing behavior: handlers and services can
		// check errors.Is(err, ErrNotFound) without caring whether a
		// CustomError sits in the middle of the chain.
		ce := NewCustomError(NotFoundError, "customer missing", ErrNotFound)
		assert.True(t, errors.Is(ce, ErrNotFound))
	})

	t.Run("errors.Is is false when no underlying err", func(t *testing.T) {
		ce := NewCustomError(NotFoundError, "customer missing", nil)
		assert.False(t, errors.Is(ce, ErrNotFound))
	})

	t.Run("errors.As extracts CustomError from wrapped chain", func(t *testing.T) {
		ce := NewCustomError(ValidationError, "bad input", nil)
		wrapped := fmt.Errorf("service.create: %w", ce)

		var got CustomError
		require.True(t, errors.As(wrapped, &got))
		assert.Equal(t, ValidationError, got.Type)
		assert.Equal(t, "bad input", got.Message)
	})

	t.Run("errors.AsType extracts CustomError from wrapped chain", func(t *testing.T) {
		// Several services use errors.AsType[CustomError] specifically;
		// pin that path so refactors to Unwrap can't silently break it.
		ce := NewCustomError(BadRequestError, "missing field", nil)
		wrapped := fmt.Errorf("wrap1: %w", fmt.Errorf("wrap2: %w", ce))

		got, ok := errors.AsType[CustomError](wrapped)
		require.True(t, ok)
		assert.Equal(t, BadRequestError, got.Type)
	})

	t.Run("Unwrap returns the stored err directly", func(t *testing.T) {
		under := errors.New("under")
		ce := NewCustomError(InternalError, "boom", under)
		assert.Equal(t, under, ce.Unwrap())
	})
}

func TestErrorTypeConstants(t *testing.T) {
	// Pin the wire-format strings: these values land in the API's `code`
	// field, so renaming one is a breaking change for client switches.
	cases := map[CustomErrorType]string{
		BadRequestError:     "bad_request",
		NotFoundError:       "not_found",
		ValidationError:     "validation_error",
		InternalError:       "internal_error",
		AuthenticationError: "auth_error",
		ForbiddenError:      "forbidden",
		ConflictError:       "conflict",
	}
	for got, want := range cases {
		t.Run(want, func(t *testing.T) {
			assert.Equal(t, want, string(got))
		})
	}
}
