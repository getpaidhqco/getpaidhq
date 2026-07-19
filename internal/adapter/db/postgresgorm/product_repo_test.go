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

func seedProduct(t *testing.T, repo *ProductRepo, orgId, name string, status domain.ProductStatus) domain.Product {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	p := domain.Product{
		OrgId:     orgId,
		Id:        ids.Generate("prod"),
		Name:      name,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := repo.Create(context.Background(), p)
	require.NoError(t, err)
	return created
}

func TestProductRepo_Find_StatusFilter(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)
	repo := &ProductRepo{db: db}

	active := seedProduct(t, repo, orgId, "Live", domain.ProductStatusActive)
	archived := seedProduct(t, repo, orgId, "Retired", domain.ProductStatusArchived)

	page := domain.Pagination{Page: 1, Limit: 50}

	t.Run("active filter excludes archived", func(t *testing.T) {
		got, total, err := repo.Find(context.Background(), orgId, page, []domain.ProductStatus{domain.ProductStatusActive})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, got, 1)
		assert.Equal(t, active.Id, got[0].Id)
	})

	t.Run("archived filter returns only archived", func(t *testing.T) {
		got, total, err := repo.Find(context.Background(), orgId, page, []domain.ProductStatus{domain.ProductStatusArchived})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, got, 1)
		assert.Equal(t, archived.Id, got[0].Id)
	})

	t.Run("nil filter returns all", func(t *testing.T) {
		got, total, err := repo.Find(context.Background(), orgId, page, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, got, 2)
	})
}

// Archiving and unarchiving must round-trip both status and archived_at through
// the full GORM Save/read path.
func TestProductRepo_ArchiveRoundTrip(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)
	repo := &ProductRepo{db: db}

	p := seedProduct(t, repo, orgId, "Plan", domain.ProductStatusActive)
	require.Nil(t, p.ArchivedAt)

	// Archive: set status + archived_at.
	now := time.Now().UTC().Truncate(time.Microsecond)
	p.Status = domain.ProductStatusArchived
	p.ArchivedAt = &now
	_, err := repo.Update(context.Background(), p)
	require.NoError(t, err)

	got, err := repo.FindById(context.Background(), orgId, p.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusArchived, got.Status)
	require.NotNil(t, got.ArchivedAt)
	assert.WithinDuration(t, now, *got.ArchivedAt, time.Second)

	// A name-only edit preserves the archived status (Update saves the full row).
	got.Name = "Plan (renamed)"
	_, err = repo.Update(context.Background(), got)
	require.NoError(t, err)
	reread, err := repo.FindById(context.Background(), orgId, p.Id)
	require.NoError(t, err)
	assert.Equal(t, "Plan (renamed)", reread.Name)
	assert.Equal(t, domain.ProductStatusArchived, reread.Status, "edit must not clear archived status")
	require.NotNil(t, reread.ArchivedAt)

	// Unarchive: clear archived_at back to NULL.
	reread.Status = domain.ProductStatusActive
	reread.ArchivedAt = nil
	_, err = repo.Update(context.Background(), reread)
	require.NoError(t, err)
	final, err := repo.FindById(context.Background(), orgId, p.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusActive, final.Status)
	assert.Nil(t, final.ArchivedAt, "unarchive must clear archived_at")
}
