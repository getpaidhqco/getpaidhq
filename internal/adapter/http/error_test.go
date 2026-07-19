package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	errors2 "getpaidhq/internal/lib/errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiError_StatusCode_Mapping(t *testing.T) {
	tests := []struct {
		code errors2.CustomErrorType
		want int
	}{
		{errors2.BadRequestError, http.StatusBadRequest},
		{errors2.NotFoundError, http.StatusNotFound},
		{errors2.ValidationError, http.StatusUnprocessableEntity},
		{errors2.InternalError, http.StatusInternalServerError},
		{errors2.AuthenticationError, http.StatusUnauthorized},
		{errors2.ForbiddenError, http.StatusForbidden},
		{errors2.ConflictError, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := NewApiError(tt.code, "msg", nil)
			assert.Equal(t, tt.want, e.StatusCode())
			assert.Equal(t, tt.want, e.GetHttpErrorCode())
		})
	}

	t.Run("unknown code falls back to 500", func(t *testing.T) {
		e := ApiError{Code: "totally_unrecognized"}
		assert.Equal(t, http.StatusInternalServerError, e.StatusCode())
	})
}

func TestApiErrorSerializer(t *testing.T) {
	t.Run("ApiError passes through unchanged", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, NewApiError(errors2.NotFoundError, "missing", nil))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var got ApiError
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, string(errors2.NotFoundError), got.Code)
		assert.Equal(t, "missing", got.Message)
	})

	t.Run("CustomError wrapped is mapped to ApiError envelope", func(t *testing.T) {
		// A bare CustomError (not pre-wrapped as ApiError) — hits the default
		// branch and NewApiErrorFromError which reads the type via errors.AsType.
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, errors2.NewCustomError(errors2.ValidationError, "bad", errors.New("under")))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		var got ApiError
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, string(errors2.ValidationError), got.Code)
		assert.Equal(t, "bad", got.Message)
	})

	t.Run("ForbiddenError code → 403 envelope", func(t *testing.T) {
		// Authz denial: handlers now return ForbiddenError instead of
		// AuthenticationError so 401 (authn failed) is distinct from 403
		// (action not permitted) on the wire.
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, NewApiError(errors2.ForbiddenError, "nope", nil))

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Fuego HTTPError with explicit Status is remapped to project code", func(t *testing.T) {
		// The serializer's fromFuegoError dispatches on the Status field of
		// the converted HTTPError; the project test pins the 422/404/401
		// translations. (When a Fuego sub-type is constructed without an
		// explicit Status the struct-conversion drops the type's overridden
		// StatusCode and the response falls through to 500 — that path is
		// documented but not asserted here.)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, fuego.HTTPError{Status: http.StatusNotFound, Title: "missing"})

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var got ApiError
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, string(errors2.NotFoundError), got.Code)
	})

	t.Run("Fuego HTTPError with 422 status maps to validation_error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, fuego.HTTPError{Status: http.StatusUnprocessableEntity, Title: "bad"})

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		var got ApiError
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, string(errors2.ValidationError), got.Code)
	})

	t.Run("generic error → bad_request fallback", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ApiErrorSerializer(rec, req, errors.New("plain old error"))

		// Code "bad_request" maps to 400.
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("nil error does not panic", func(t *testing.T) {
		// Defensive: nil shouldn't reach the serializer in practice, but a
		// guard against panics is cheap. Renders as 500 with the
		// internal_error code.
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		assert.NotPanics(t, func() {
			ApiErrorSerializer(rec, req, nil)
		})
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestNewApiErrorFromError(t *testing.T) {
	t.Run("preserves CustomError type", func(t *testing.T) {
		ce := errors2.NewCustomError(errors2.NotFoundError, "no such thing", errors.New("404 under"))
		got := NewApiErrorFromError(ce)
		assert.Equal(t, string(errors2.NotFoundError), got.Code)
		assert.Equal(t, "no such thing", got.Message)
		assert.Equal(t, "404 under", got.Details)
	})

	t.Run("plain error falls through to bad_request", func(t *testing.T) {
		got := NewApiErrorFromError(errors.New("oops"))
		assert.Equal(t, string(errors2.BadRequestError), got.Code)
	})

	t.Run("nil error renders as internal_error", func(t *testing.T) {
		got := NewApiErrorFromError(nil)
		assert.Equal(t, string(errors2.InternalError), got.Code)
	})

	t.Run("wrapped CustomError is still detected via Unwrap chain", func(t *testing.T) {
		// errors.AsType walks Unwrap, so a CustomError buried under
		// fmt.Errorf("...: %w", ...) must still surface its typed code.
		// This pins the behavior that depends on CustomError implementing
		// Unwrap, added together with this test.
		ce := errors2.NewCustomError(errors2.NotFoundError, "no such thing", nil)
		wrapped := fmt.Errorf("service.get: %w", ce)

		got := NewApiErrorFromError(wrapped)

		assert.Equal(t, string(errors2.NotFoundError), got.Code)
		assert.Equal(t, "no such thing", got.Message)
	})

	t.Run("wrapped lib.ErrNotFound maps to not_found", func(t *testing.T) {
		// Repositories return ErrNotFound (often wrapped); the serializer
		// recognizes it without each service having to translate.
		got := NewApiErrorFromError(fmt.Errorf("customer lookup: %w", errors2.ErrNotFound))
		assert.Equal(t, string(errors2.NotFoundError), got.Code)
	})
}
