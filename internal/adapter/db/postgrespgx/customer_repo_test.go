//go:build integration

package postgrespgx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/db/storagetest"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// poolForTest boots the shared container and opens a pgx pool against it.
func poolForTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := storagetest.StartPostgres(t)
	pool, err := NewDatabase(dsn, nil, "silent")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func seedOrgForTest(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	orgId := lib.GenerateId("org_test")
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := NewOrgRepo(pool).Create(context.Background(), domain.Org{
		Id: orgId, Name: "Test Org " + orgId, Country: "US", Timezone: "UTC",
		Status: domain.OrgStatusActive, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	return orgId
}

func TestCustomerRepo_PgxReference(t *testing.T) {
	pool := poolForTest(t)
	orgId := seedOrgForTest(t, pool)
	ctx := context.Background()
	repo := NewCustomerRepo(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	cust := domain.Customer{
		OrgId: orgId, Id: lib.GenerateId("cus"),
		FirstName: "Ada", LastName: "Lovelace",
		Email:          fmt.Sprintf("%s@example.com", lib.GenerateId("ada")),
		Phone:          "+15551234",
		BillingAddress: domain.Address{Line1: "1 Engine Way", City: "London", Country: "GB"},
		Metadata:       map[string]string{"tier": "gold"},
		CreatedAt:      now, UpdatedAt: now,
	}

	created, err := repo.Create(ctx, cust)
	require.NoError(t, err)
	require.Equal(t, cust.Email, created.Email)
	require.Equal(t, "", created.ExternalId, "unset external_id round-trips as empty (NULL in db)")
	require.Equal(t, "London", created.BillingAddress.City, "json column round-trips")
	require.Equal(t, "gold", created.Metadata["tier"], "metadata json round-trips")

	got, err := repo.FindById(ctx, orgId, cust.Id)
	require.NoError(t, err)
	require.Equal(t, created, got)

	byEmail, err := repo.FindByEmail(ctx, orgId, cust.Email)
	require.NoError(t, err)
	require.Equal(t, cust.Id, byEmail.Id)

	// not-found maps to the domain sentinel
	_, err = repo.FindById(ctx, orgId, "missing")
	require.ErrorIs(t, err, port.ErrNotFound)

	// update
	got.LastName = "Byron"
	updated, err := repo.Update(ctx, got)
	require.NoError(t, err)
	require.Equal(t, "Byron", updated.LastName)

	// batch + list
	batch, err := repo.FindByIds(ctx, orgId, []string{cust.Id})
	require.NoError(t, err)
	require.Len(t, batch, 1)

	list, total, err := repo.List(ctx, orgId, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, list, 1)
}

func TestCohortCRUD_PgxReference(t *testing.T) {
	pool := poolForTest(t)
	orgId := seedOrgForTest(t, pool)
	ctx := context.Background()
	repo := NewCustomerRepo(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	coh := domain.Cohort{
		OrgId: orgId, Id: lib.GenerateId("coh"), Name: "VIPs",
		Type: domain.CohortType("static"), Metadata: map[string]string{"k": "v"},
		CreatedAt: now, UpdatedAt: now,
	}
	created, err := repo.CreateCohort(ctx, coh)
	require.NoError(t, err)
	require.Equal(t, "VIPs", created.Name)

	created.Name = "VVIPs"
	updated, err := repo.UpdateCohort(ctx, created)
	require.NoError(t, err)
	require.Equal(t, "VVIPs", updated.Name)

	_, err = repo.DeleteCohort(ctx, updated)
	require.NoError(t, err)
	_, err = repo.FindCohortById(ctx, orgId, coh.Id)
	require.ErrorIs(t, err, port.ErrNotFound)
}
