//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"

	"github.com/stretchr/testify/require"
)

func TestOrgRepo_ListIds(t *testing.T) {
	db := testDB(t)
	repo := NewOrgRepo(db)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	_, err := repo.Create(ctx, domain.Org{
		Id:        orgId,
		Name:      "List Test",
		Status:    domain.OrgStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	ids, err := repo.ListIds(ctx)
	require.NoError(t, err)
	require.Contains(t, ids, orgId)
}
