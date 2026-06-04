package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// recordingApiKeyRepo is a controllable port.ApiKeyRepository. Distinct from
// fakeApiKeyRepo in org_test.go so the two test suites can evolve independently.
type recordingApiKeyRepo struct {
	port.ApiKeyRepository
	created    []domain.ApiKey
	createErr  error
	listResult []domain.ApiKey
	listTotal  int
	listErr    error
	deletedIds []string
	deleteErr  error
}

func (r *recordingApiKeyRepo) Create(_ context.Context, k domain.ApiKey) (domain.ApiKey, error) {
	if r.createErr != nil {
		return domain.ApiKey{}, r.createErr
	}
	r.created = append(r.created, k)
	return k, nil
}

func (r *recordingApiKeyRepo) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.ApiKey, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	return r.listResult, r.listTotal, nil
}

func (r *recordingApiKeyRepo) Delete(_ context.Context, _ string, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deletedIds = append(r.deletedIds, id)
	return nil
}

const testApiKeyPepper = "test_pepper_do_not_reuse"

func TestApiKeyService_Create_HappyPath(t *testing.T) {
	repo := &recordingApiKeyRepo{}
	svc := NewApiKeyService(repo, testApiKeyPepper, silentLogger{})

	out, err := svc.Create(context.Background(), "org_1", "ci-deploy")
	require.NoError(t, err)

	// Surface contract.
	assert.True(t, strings.HasPrefix(out.ApiKey.Id, "sk_"), "id must be sk-prefixed; got %q", out.ApiKey.Id)
	assert.Equal(t, "ci-deploy", out.ApiKey.Name)
	assert.NotEmpty(t, out.Key, "raw key must be returned once on create")
	assert.True(t, strings.HasPrefix(out.Key, out.ApiKey.Id+"_"), "raw key must be <id>_<secret>")
	// The hash is present on the returned struct (it's the persisted entity),
	// but the domain JSON tag `json:"-"` and the handler's response DTO keep
	// it off the wire — that's enforced in api_key_handler_test.go.

	// Storage contract — only the hash, never the raw key.
	require.Len(t, repo.created, 1)
	stored := repo.created[0]
	assert.Equal(t, "org_1", stored.OrgId)
	assert.Equal(t, out.ApiKey.Id, stored.Id)
	assert.Equal(t, "ci-deploy", stored.Name)
	assert.NotEqual(t, out.Key, stored.KeyHash, "the stored hash must not equal the raw key")

	// Hash must verify under the configured pepper.
	mac := hmac.New(sha256.New, []byte(testApiKeyPepper))
	mac.Write([]byte(out.Key))
	want := hex.EncodeToString(mac.Sum(nil))
	assert.Equal(t, want, stored.KeyHash, "stored hash must be HMAC-SHA256(pepper, raw)")
}

func TestApiKeyService_Create_OptionalName(t *testing.T) {
	repo := &recordingApiKeyRepo{}
	svc := NewApiKeyService(repo, testApiKeyPepper, silentLogger{})

	out, err := svc.Create(context.Background(), "org_1", "")
	require.NoError(t, err)
	assert.Empty(t, out.ApiKey.Name)
	require.Len(t, repo.created, 1)
	assert.Empty(t, repo.created[0].Name)
}

func TestApiKeyService_Create_MissingPepperFails(t *testing.T) {
	// Empty pepper means HashApiKey returns ErrMissingApiKeyPepper.
	svc := NewApiKeyService(&recordingApiKeyRepo{}, "", silentLogger{})

	_, err := svc.Create(context.Background(), "org_1", "x")
	require.Error(t, err)
}

func TestApiKeyService_Create_RepoErrorBubblesUp(t *testing.T) {
	repo := &recordingApiKeyRepo{createErr: errors.New("db down")}
	svc := NewApiKeyService(repo, testApiKeyPepper, silentLogger{})

	_, err := svc.Create(context.Background(), "org_1", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func TestApiKeyService_List_PassesThroughOrgScope(t *testing.T) {
	repo := &recordingApiKeyRepo{
		listResult: []domain.ApiKey{{OrgId: "org_1", Id: "sk_a"}, {OrgId: "org_1", Id: "sk_b"}},
		listTotal:  2,
	}
	svc := NewApiKeyService(repo, testApiKeyPepper, silentLogger{})

	got, total, err := svc.List(context.Background(), "org_1", domain.Pagination{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, got, 2)
}

func TestApiKeyService_Delete_PassesThrough(t *testing.T) {
	repo := &recordingApiKeyRepo{}
	svc := NewApiKeyService(repo, testApiKeyPepper, silentLogger{})

	require.NoError(t, svc.Delete(context.Background(), "org_1", "sk_xyz"))
	assert.Equal(t, []string{"sk_xyz"}, repo.deletedIds)
}
