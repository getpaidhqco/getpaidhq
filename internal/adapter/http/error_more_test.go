package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/lib"
)

// Fills the remaining branches in ApiErrorSerializer + fromFuegoError +
// NewApiErrorFromError that error_test.go didn't reach.

func serializeErr(t *testing.T, err error) (int, ApiError) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ApiErrorSerializer(rec, req, err)
	var got ApiError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return rec.Code, got
}

func TestApiErrorSerializer_FuegoSubtypes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantApi  string
	}{
		{"BadRequestError", fuego.BadRequestError{Status: http.StatusBadRequest, Title: "bad"}, http.StatusBadRequest, string(lib.BadRequestError)},
		{"UnauthorizedError", fuego.UnauthorizedError{Status: http.StatusUnauthorized, Title: "no"}, http.StatusUnauthorized, string(lib.AuthenticationError)},
		{"ForbiddenError", fuego.ForbiddenError{Status: http.StatusForbidden, Title: "nope"}, http.StatusForbidden, string(lib.ForbiddenError)},
		{"ConflictError", fuego.ConflictError{Status: http.StatusConflict, Title: "clash"}, http.StatusConflict, string(lib.ConflictError)},
		{"NotFoundError", fuego.NotFoundError{Status: http.StatusNotFound, Title: "gone"}, http.StatusNotFound, string(lib.NotFoundError)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, got := serializeErr(t, c.err)
			assert.Equal(t, c.wantCode, code)
			assert.Equal(t, c.wantApi, got.Code)
		})
	}
}

func TestApiErrorSerializer_HTTPError_Status401_500(t *testing.T) {
	t.Run("401 maps to AuthenticationError", func(t *testing.T) {
		code, got := serializeErr(t, fuego.HTTPError{Status: http.StatusUnauthorized, Title: "auth"})
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, string(lib.AuthenticationError), got.Code)
	})
	t.Run("500 maps to InternalError", func(t *testing.T) {
		code, got := serializeErr(t, fuego.HTTPError{Status: http.StatusInternalServerError, Title: "boom"})
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Equal(t, string(lib.InternalError), got.Code)
	})
	t.Run("418 (unmapped) defaults to BadRequestError code, status 400", func(t *testing.T) {
		code, got := serializeErr(t, fuego.HTTPError{Status: http.StatusTeapot, Title: "🫖"})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, string(lib.BadRequestError), got.Code)
	})
}

func TestApiErrorSerializer_HTTPError_DetailAndErrorsFields(t *testing.T) {
	t.Run("Errors slice becomes Details", func(t *testing.T) {
		he := fuego.HTTPError{
			Status: http.StatusBadRequest,
			Title:  "validation",
			Errors: []fuego.ErrorItem{{Name: "field", Reason: "required"}},
		}
		_, got := serializeErr(t, he)
		assert.NotNil(t, got.Details, "Errors slice routed into Details")
	})
	t.Run("Detail string becomes Details when Errors empty", func(t *testing.T) {
		he := fuego.HTTPError{Status: http.StatusBadRequest, Title: "x", Detail: "human-readable"}
		_, got := serializeErr(t, he)
		assert.Equal(t, "human-readable", got.Details)
	})
	t.Run("Title empty falls back to Error()", func(t *testing.T) {
		he := fuego.HTTPError{Status: http.StatusBadRequest, Detail: "only-detail"}
		_, got := serializeErr(t, he)
		assert.NotEmpty(t, got.Message)
	})
}

func TestNewApiErrorFromError_ValidatorErrors(t *testing.T) {
	type req struct {
		Email string `validate:"required,email"`
	}
	err := validator.New().Struct(req{})
	require.Error(t, err)
	verrs, ok := err.(validator.ValidationErrors)
	require.True(t, ok)

	got := NewApiErrorFromError(verrs)
	assert.Equal(t, string(lib.BadRequestError), got.Code)
	assert.Equal(t, "Input validation failed", got.Message)
	require.NotNil(t, got.Details, "Details is the formatted field/message list")
}

func TestNewApiErrorFromError_CustomError_NilUnderlying(t *testing.T) {
	// A CustomError with Err==nil exercises the branch that returns
	// NewApiError(serr.Type, serr.Message, nil) instead of e.Err.Error().
	ce := lib.NewCustomError(lib.AuthenticationError, "you shall not pass", nil)
	got := NewApiErrorFromError(ce)
	assert.Equal(t, string(lib.AuthenticationError), got.Code)
	assert.Equal(t, "you shall not pass", got.Message)
	assert.Nil(t, got.Details, "no underlying error → nil details")
}

func TestApiErrorSerializer_ServerErrorEnvelope(t *testing.T) {
	// A plain `errors.New(...)` not wrapped in anything — goes through the
	// default branch and renders as BadRequest with message + details echoing
	// the underlying text.
	code, got := serializeErr(t, errors.New("db connection refused"))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "db connection refused", got.Message)
	assert.Equal(t, "db connection refused", got.Details)
}
