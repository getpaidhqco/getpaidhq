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

// fakeApiKeyRepo is a controllable port.ApiKeyRepository. Only FindByKey is
// exercised by the middleware; the rest satisfy the interface.
type fakeApiKeyRepo struct {
	byKey     map[string]domain.ApiKey
	findErr   error
	keyLookup string
}

func (f *fakeApiKeyRepo) FindByKey(_ context.Context, key string) (domain.ApiKey, error) {
	f.keyLookup = key
	if f.findErr != nil {
		return domain.ApiKey{}, f.findErr
	}
	k, ok := f.byKey[key]
	if !ok {
		return domain.ApiKey{}, errors.New("not found")
	}
	return k, nil
}

func (f *fakeApiKeyRepo) FindById(context.Context, string, string) (domain.ApiKey, error) {
	return domain.ApiKey{}, errors.New("not implemented")
}
func (f *fakeApiKeyRepo) Create(_ context.Context, e domain.ApiKey) (domain.ApiKey, error) {
	return e, nil
}
func (f *fakeApiKeyRepo) Update(_ context.Context, e domain.ApiKey) (domain.ApiKey, error) {
	return e, nil
}
func (f *fakeApiKeyRepo) Delete(context.Context, string, string) error { return nil }

func newAuth(repo port.ApiKeyRepository) port.Authenticator {
	return NewApiKeyMiddleware(noopLogger{}, lib.Env{}, repo)
}

func TestApiKeyMiddleware_Authenticate(t *testing.T) {
	t.Run("valid key resolves to an admin AuthUser scoped to the org", func(t *testing.T) {
		repo := &fakeApiKeyRepo{byKey: map[string]domain.ApiKey{
			"live_abc": {OrgId: "org_42", Id: "key_1", Key: "live_abc"},
		}}
		user, err := newAuth(repo).Authenticate(context.Background(), "live_abc")

		require.NoError(t, err)
		assert.Equal(t, "org_42", user.OrgId)
		assert.Equal(t, "key_1", user.Id)
		assert.Equal(t, "", user.Email)
		assert.Equal(t, port.RoleAdmin, user.PrimaryRole)
		assert.Equal(t, []port.UserRole{port.RoleAdmin}, user.Roles)
		assert.Equal(t, "live_abc", repo.keyLookup, "the raw token is used as the key")
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

	t.Run("unknown key propagates the repo lookup error", func(t *testing.T) {
		repo := &fakeApiKeyRepo{byKey: map[string]domain.ApiKey{}}
		user, err := newAuth(repo).Authenticate(context.Background(), "missing")

		require.Error(t, err)
		assert.Equal(t, port.AuthUser{}, user)
	})

	t.Run("repo error is propagated verbatim", func(t *testing.T) {
		sentinel := errors.New("db down")
		repo := &fakeApiKeyRepo{findErr: sentinel}
		_, err := newAuth(repo).Authenticate(context.Background(), "any")

		require.ErrorIs(t, err, sentinel)
	})
}
