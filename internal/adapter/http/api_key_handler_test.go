package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
)

// listingApiKeyRepo extends the existing testhelpers fakeApiKeyRepo with List
// + Delete behavior (the testhelpers fake only models Create). Same package,
// so we can add methods to fakeApiKeyRepo here without redeclaring it.
type listingApiKeyRepo struct {
	*fakeApiKeyRepo
	listResult []domain.ApiKey
	listTotal  int
	deletedIds []string
}

func (r *listingApiKeyRepo) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.ApiKey, int, error) {
	return r.listResult, r.listTotal, nil
}

func (r *listingApiKeyRepo) Delete(_ context.Context, _ string, id string) error {
	r.deletedIds = append(r.deletedIds, id)
	return nil
}

const apiKeyTestPepper = "handler_test_pepper"

func newApiKeyHandlerForTest(t *testing.T, repo *listingApiKeyRepo) *ApiKeyHandler {
	t.Helper()
	svc := service.NewApiKeyService(repo, apiKeyTestPepper, silentLogger{})
	return NewApiKeyHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestApiKeyHandler_Create_ReturnsKeyOnce(t *testing.T) {
	repo := &listingApiKeyRepo{fakeApiKeyRepo: &fakeApiKeyRepo{}}
	h := newApiKeyHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/api-keys", CreateApiKeyInput{Name: "ci-deploy"})

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var got ApiKeyCreateResponse
	decodeJSON(t, rec, &got)

	assert.True(t, strings.HasPrefix(got.Id, "sk_"), "id must be sk-prefixed")
	assert.Equal(t, "ci-deploy", got.Name)
	assert.True(t, strings.HasPrefix(got.Key, got.Id+"_"), "key must be <id>_<secret>")
	// Stored hash must NOT equal the raw key.
	require.Len(t, repo.created, 1)
	assert.NotEqual(t, got.Key, repo.created[0].KeyHash)

	// Response must not leak the hash on the wire, regardless of DTO.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	assert.NotContains(t, raw, "key_hash", "wire response must not contain key_hash")
}

func TestApiKeyHandler_Create_NameIsOptional(t *testing.T) {
	repo := &listingApiKeyRepo{fakeApiKeyRepo: &fakeApiKeyRepo{}}
	h := newApiKeyHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/api-keys", CreateApiKeyInput{})

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

func TestApiKeyHandler_List_ReturnsMetadataNotSecrets(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := &listingApiKeyRepo{
		fakeApiKeyRepo: &fakeApiKeyRepo{},
		listResult: []domain.ApiKey{
			{OrgId: "org_1", Id: "sk_a", Name: "ci", KeyHash: "h1", CreatedAt: now, UpdatedAt: now},
			{OrgId: "org_1", Id: "sk_b", Name: "webhook", KeyHash: "h2", CreatedAt: now, UpdatedAt: now},
		},
		listTotal: 2,
	}
	h := newApiKeyHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/api-keys?page=0&limit=20", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	// Raw wire check: no key_hash, no key, anywhere.
	body := rec.Body.String()
	assert.NotContains(t, body, "key_hash")
	assert.NotContains(t, body, "\"key\"")

	var got struct {
		Data []ApiKeyResponse `json:"data"`
		Meta Meta             `json:"meta"`
	}
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
	require.Len(t, got.Data, 2)
	assert.Equal(t, "sk_a", got.Data[0].Id)
	assert.Equal(t, "ci", got.Data[0].Name)
}

func TestApiKeyHandler_Delete_ReturnsNoContent(t *testing.T) {
	repo := &listingApiKeyRepo{fakeApiKeyRepo: &fakeApiKeyRepo{}}
	h := newApiKeyHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodDelete, "/api/api-keys/sk_xyz", nil)

	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, []string{"sk_xyz"}, repo.deletedIds)
}

func TestApiKeyHandler_Member_Denied(t *testing.T) {
	// Cedar grants CreateApiKey to owner + admin only; member is denied.
	repo := &listingApiKeyRepo{fakeApiKeyRepo: &fakeApiKeyRepo{}}
	h := newApiKeyHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(memberUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/api-keys", CreateApiKeyInput{Name: "x"})

	assertErrorEnvelope(t, rec, http.StatusForbidden, "forbidden")
}
