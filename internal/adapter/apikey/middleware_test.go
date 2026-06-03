package apikey

import (
	"context"
	"errors"
	"testing"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }

const testPepper = "test-pepper-deadbeef"

// fakeApiKeyRepo is a controllable port.ApiKeyRepository. Only FindByKey is
// exercised by the middleware; the rest satisfy the interface. The middleware
// hashes incoming raw keys with the configured pepper before lookup, so the
// map keys here are the HMAC hashes, not the raw tokens.
type fakeApiKeyRepo struct {
	byHash    map[string]domain.ApiKey
	findErr   error
	keyLookup string
}

func (f *fakeApiKeyRepo) FindByKey(_ context.Context, keyHash string) (domain.ApiKey, error) {
	f.keyLookup = keyHash
	if f.findErr != nil {
		return domain.ApiKey{}, f.findErr
	}
	k, ok := f.byHash[keyHash]
	if !ok {
		return domain.ApiKey{}, errors.New("not found")
	}
	return k, nil
}

func (f *fakeApiKeyRepo) FindById(context.Context, string, string) (domain.ApiKey, error) {
	return domain.ApiKey{}, errors.New("not implemented")
}
func (f *fakeApiKeyRepo) List(context.Context, string, domain.Pagination) ([]domain.ApiKey, int, error) {
	return nil, 0, errors.New("not implemented")
}
func (f *fakeApiKeyRepo) Create(_ context.Context, e domain.ApiKey) (domain.ApiKey, error) {
	return e, nil
}
func (f *fakeApiKeyRepo) Update(_ context.Context, e domain.ApiKey) (domain.ApiKey, error) {
	return e, nil
}
func (f *fakeApiKeyRepo) Delete(context.Context, string, string) error { return nil }

func newAuth(repo port.ApiKeyRepository) port.Authenticator {
	return NewApiKeyMiddleware(noopLogger{}, lib.Env{ApiKeyPepper: testPepper}, repo)
}

func TestApiKeyMiddleware_Authenticate(t *testing.T) {
	t.Run("valid key resolves to an admin AuthUser scoped to the org", func(t *testing.T) {
		hash, err := lib.HashApiKey("live_abc", testPepper)
		require.NoError(t, err)
		repo := &fakeApiKeyRepo{byHash: map[string]domain.ApiKey{
			hash: {OrgId: "org_42", Id: "key_1", KeyHash: hash},
		}}
		user, err := newAuth(repo).Authenticate(context.Background(), "live_abc")

		require.NoError(t, err)
		assert.Equal(t, "org_42", user.OrgId)
		assert.Equal(t, "key_1", user.Id)
		assert.Equal(t, "", user.Email)
		assert.Equal(t, port.RoleAdmin, user.PrimaryRole)
		assert.Equal(t, []port.UserRole{port.RoleAdmin}, user.Roles)
		assert.Equal(t, hash, repo.keyLookup, "the HMAC hash is used as the lookup key, not the raw token")
	})

	t.Run("empty token is rejected without touching the repo", func(t *testing.T) {
		repo := &fakeApiKeyRepo{}
		user, err := newAuth(repo).Authenticate(context.Background(), "")

		require.Error(t, err)
		assert.Equal(t, port.AuthUser{}, user)
		var ce lib.CustomError
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, lib.AuthenticationError, ce.Type)
		assert.Equal(t, "", repo.keyLookup, "repo must not be queried for an empty token")
	})

	t.Run("unknown key returns an opaque authentication error", func(t *testing.T) {
		repo := &fakeApiKeyRepo{byHash: map[string]domain.ApiKey{}}
		user, err := newAuth(repo).Authenticate(context.Background(), "missing")

		require.Error(t, err)
		assert.Equal(t, port.AuthUser{}, user)
		var ce lib.CustomError
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, lib.AuthenticationError, ce.Type, "unknown key must look identical to any other authn failure")
	})

	t.Run("repo error is hidden behind an opaque authentication error", func(t *testing.T) {
		sentinel := errors.New("db down")
		repo := &fakeApiKeyRepo{findErr: sentinel}
		_, err := newAuth(repo).Authenticate(context.Background(), "any")

		require.Error(t, err)
		var ce lib.CustomError
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, lib.AuthenticationError, ce.Type, "internal failures must not leak as a distinct error to the caller")
	})

	t.Run("missing API_KEY_PEPPER fails closed", func(t *testing.T) {
		repo := &fakeApiKeyRepo{}
		auth := NewApiKeyMiddleware(noopLogger{}, lib.Env{ApiKeyPepper: ""}, repo)
		_, err := auth.Authenticate(context.Background(), "any")

		require.Error(t, err)
		assert.Equal(t, "", repo.keyLookup, "no repo lookup when pepper is missing")
	})
}
