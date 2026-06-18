//go:build integration

package postgresgorm

import (
	"context"
	"errors"
	"testing"
	"time"

	"getpaidhq/internal/core/port"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

func TestCustomerRepo(t *testing.T) {
	db := testDB(t)
	repo := NewCustomerRepo(db)
	ctx := context.Background()

	t.Run("Create then FindById round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		in := domain.Customer{
			OrgId:     orgId,
			Id:        lib.GenerateId("cus"),
			FirstName: "Grace",
			LastName:  "Hopper",
			Email:     lib.GenerateId("grace") + "@example.com",
			BillingAddress: domain.Address{
				Line1:   "1 Compiler Rd",
				City:    "Arlington",
				Country: "US",
			},
			Metadata:  map[string]string{"vip": "true"},
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err := repo.Create(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, in.Id, created.Id)
		assert.Equal(t, "Grace", created.FirstName)
		// Serialized JSON columns round-trip.
		assert.Equal(t, "1 Compiler Rd", created.BillingAddress.Line1)
		assert.Equal(t, map[string]string{"vip": "true"}, created.Metadata)

		got, err := repo.FindById(ctx, orgId, in.Id)
		require.NoError(t, err)
		assert.Equal(t, in.Email, got.Email)
		assert.Equal(t, domain.Country("US"), got.BillingAddress.Country)
	})

	t.Run("FindByEmail matches the seeded email", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)

		got, err := repo.FindByEmail(ctx, orgId, cust.Email)
		require.NoError(t, err)
		assert.Equal(t, cust.Id, got.Id)
	})

	t.Run("Update mutates fields", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)

		cust.FirstName = "Renamed"
		cust.Phone = "+19998887777"
		updated, err := repo.Update(ctx, cust)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", updated.FirstName)

		reread, err := repo.FindById(ctx, orgId, cust.Id)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", reread.FirstName)
		assert.Equal(t, "+19998887777", reread.Phone)
	})

	t.Run("FindById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindById(ctx, orgId, "nope")
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})

	t.Run("List paginates and counts within org", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		for range 3 {
			seedCustomer(t, db, orgId)
		}
		p := domain.Pagination{Limit: 2, Offset: 0, SortBy: "created_at", SortDirection: "asc"}
		got, count, err := repo.List(ctx, orgId, p)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Len(t, got, 2)
	})

	t.Run("FindPaymentMethodById round-trips and is org-scoped", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		cust := seedCustomer(t, db, orgA)

		pm := domain.PaymentMethod{
			OrgId:      orgA,
			Id:         lib.GenerateId("pm"),
			Status:     domain.PaymentMethodStatusActive,
			Psp:        string(domain.Paystack),
			Name:       "Visa ****4242",
			CustomerId: cust.Id,
			Type:       domain.PaymentMethodTypeCard,
			Token:      "tok_123",
			Details:    map[string]any{"brand": "visa", "last4": "4242"},
			ExpireAt:   time.Now().UTC().AddDate(2, 0, 0).Truncate(time.Microsecond),
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		}
		pmRow := paymentMethodRowFromDomain(pm)
		require.NoError(t, db.Create(&pmRow).Error)

		got, err := repo.FindPaymentMethodById(ctx, orgA, pm.Id)
		require.NoError(t, err)
		assert.Equal(t, pm.Id, got.Id)
		assert.Equal(t, "Visa ****4242", got.Name)
		assert.Equal(t, domain.PaymentMethodStatusActive, got.Status)

		// org B cannot see org A's payment method.
		_, err = repo.FindPaymentMethodById(ctx, orgB, pm.Id)
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})

	t.Run("org-scoping isolates customers", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		cust := seedCustomer(t, db, orgA)

		_, err := repo.FindById(ctx, orgB, cust.Id)
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})
}
