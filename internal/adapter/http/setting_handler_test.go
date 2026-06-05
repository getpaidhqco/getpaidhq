package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type fakeSettingRepoHTTP struct {
	port.SettingRepository
	items   map[string]domain.Setting
	deleted []string
	createN int
	upsertN int
}

func newFakeSettingRepoHTTP() *fakeSettingRepoHTTP {
	return &fakeSettingRepoHTTP{items: map[string]domain.Setting{}}
}
func key(p, i string) string { return p + "|" + i }

func (r *fakeSettingRepoHTTP) Create(_ context.Context, s domain.Setting) (domain.Setting, error) {
	r.createN++
	r.items[key(s.ParentId, s.Id)] = s
	return s, nil
}
func (r *fakeSettingRepoHTTP) Upsert(_ context.Context, s domain.Setting) (domain.Setting, error) {
	r.upsertN++
	r.items[key(s.ParentId, s.Id)] = s
	return s, nil
}
func (r *fakeSettingRepoHTTP) FindById(_ context.Context, _, parentId, id string) (domain.Setting, error) {
	if s, ok := r.items[key(parentId, id)]; ok {
		return s, nil
	}
	return domain.Setting{}, port.ErrNotFound
}
func (r *fakeSettingRepoHTTP) List(_ context.Context, _, parentId string, _ domain.Pagination) ([]domain.Setting, int, error) {
	var out []domain.Setting
	for _, s := range r.items {
		if parentId == "" || s.ParentId == parentId {
			out = append(out, s)
		}
	}
	return out, len(out), nil
}
func (r *fakeSettingRepoHTTP) Delete(_ context.Context, _, parentId, id string) error {
	r.deleted = append(r.deleted, key(parentId, id))
	delete(r.items, key(parentId, id))
	return nil
}

func newSettingHandlerForTest(t *testing.T, repo *fakeSettingRepoHTTP) *SettingHandler {
	t.Helper()
	return NewSettingHandler(service.NewSettingService(repo, silentLogger{}), silentLogger{}, newRealAuthz(t))
}

func TestSettingHandler_AuthzGuards(t *testing.T) {
	repo := newFakeSettingRepoHTTP()
	h := newSettingHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())
	cases := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/settings", CreateSettingRequest{Id: "k"}},
		{http.MethodGet, "/api/settings", nil},
		{http.MethodGet, "/api/settings/p/k", nil},
		{http.MethodPut, "/api/settings/p/k", UpdateSettingRequest{Value: "v"}},
		{http.MethodDelete, "/api/settings/p/k", nil},
	}
	for _, tc := range cases {
		rec := doJSON(t, ts, tc.method, tc.path, tc.body)
		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	}
	assert.Zero(t, repo.createN)
}

func TestSettingHandler_CRUD(t *testing.T) {
	repo := newFakeSettingRepoHTTP()
	h := newSettingHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	// Create
	rec := doJSON(t, ts, http.MethodPost, "/api/settings", CreateSettingRequest{ParentId: "dunning", Id: "max_retries", Type: "int", Value: "5"})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var created SettingResponse
	decodeJSON(t, rec, &created)
	assert.Equal(t, "max_retries", created.Id)
	assert.Equal(t, 1, repo.createN)

	// Get
	rec = doJSON(t, ts, http.MethodGet, "/api/settings/dunning/max_retries", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var got SettingResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "5", got.Value)

	// Update (upsert)
	rec = doJSON(t, ts, http.MethodPut, "/api/settings/dunning/max_retries", UpdateSettingRequest{Type: "int", Value: "8"})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, repo.upsertN)

	// List
	rec = doJSON(t, ts, http.MethodGet, "/api/settings?parent_id=dunning", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var list ListResponse
	decodeJSON(t, rec, &list)
	assert.Equal(t, 1, list.Meta.Total)

	// Delete
	rec = doJSON(t, ts, http.MethodDelete, "/api/settings/dunning/max_retries", nil)
	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, []string{key("dunning", "max_retries")}, repo.deleted)
}
