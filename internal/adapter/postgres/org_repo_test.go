//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"

	"github.com/stretchr/testify/require"
)

func TestOrgRepo_ListIds(t *testing.T) {
	db := testDB(t)
	repo := NewOrgRepo(db)
	ctx := context.Background()

	// Generate the ID directly — repo.Create inserts the org row itself, so we
	// must not pre-seed it via uniqueOrg (which would cause a duplicate PK).
	orgId := lib.GenerateId("org_test")
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
