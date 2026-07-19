//go:build integration

package postgresgorm

import (
	"context"
	"getpaidhq/internal/lib/ids"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func newApiKey(orgId, name string) domain.ApiKey {
	id := ids.Generate("sk")
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.ApiKey{
		OrgId:     orgId,
		Id:        id,
		Name:      name,
		KeyHash:   "hash_" + id, // unique to satisfy the unique index
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestApiKeyRepo_CreateAndFindById(t *testing.T) {
	db := testDB(t)
	repo := NewApiKeyRepo(db)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	in := newApiKey(orgId, "ci-deploy")
	created, err := repo.Create(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, in.Id, created.Id)
	assert.Equal(t, "ci-deploy", created.Name)

	got, err := repo.FindById(context.Background(), orgId, in.Id)
	require.NoError(t, err)
	assert.Equal(t, in.Id, got.Id)
	assert.Equal(t, "ci-deploy", got.Name)
	assert.Equal(t, in.KeyHash, got.KeyHash)
}

func TestApiKeyRepo_FindByKey(t *testing.T) {
	db := testDB(t)
	repo := NewApiKeyRepo(db)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	in := newApiKey(orgId, "")
	_, err := repo.Create(context.Background(), in)
	require.NoError(t, err)

	got, err := repo.FindByKey(context.Background(), in.KeyHash)
	require.NoError(t, err)
	assert.Equal(t, in.Id, got.Id)
	assert.Equal(t, orgId, got.OrgId)
}

func TestApiKeyRepo_List_OrgScopedAndPaginated(t *testing.T) {
	db := testDB(t)
	repo := NewApiKeyRepo(db)
	orgA := uniqueOrg(t)
	orgB := uniqueOrg(t)
	cleanupOrg(t, db, orgA)
	cleanupOrg(t, db, orgB)

	// 3 keys in org A, 1 in org B. List(orgA) must see only A's.
	for i := 0; i < 3; i++ {
		_, err := repo.Create(context.Background(), newApiKey(orgA, ""))
		require.NoError(t, err)
	}
	_, err := repo.Create(context.Background(), newApiKey(orgB, ""))
	require.NoError(t, err)

	got, total, err := repo.List(context.Background(), orgA, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, got, 3)
	for _, k := range got {
		assert.Equal(t, orgA, k.OrgId, "list must not leak rows from another org")
	}
}

func TestApiKeyRepo_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewApiKeyRepo(db)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	in := newApiKey(orgId, "")
	_, err := repo.Create(context.Background(), in)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(context.Background(), orgId, in.Id))

	_, err = repo.FindById(context.Background(), orgId, in.Id)
	require.Error(t, err, "FindById must error after delete")
}
