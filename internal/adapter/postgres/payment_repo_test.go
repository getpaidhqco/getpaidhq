//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"getpaidhq/internal/core/port"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

func newPayment(orgId string) domain.Payment {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.Payment{
		OrgId:       orgId,
		Id:          lib.GenerateId("pay"),
		Psp:         domain.Paystack,
		PspId:       lib.GenerateId("psp"),
		Reference:   "REF-" + lib.GenerateId("r"),
		Status:      domain.PaymentStatusSucceeded,
		Recurring:   true,
		Currency:    "USD",
		Amount:      1999,
		PspFee:      59,
		NetAmount:   1940,
		Metadata:    map[string]string{"channel": "card"},
		CompletedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestPaymentRepo(t *testing.T) {
	db := testDB(t)
	repo := NewPaymentRepo(db)
	ctx := context.Background()

	t.Run("Create then FindById round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		p := newPayment(orgId)

		created, err := repo.Create(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, p.Id, created.Id)
		assert.Equal(t, int64(1999), created.Amount)
		assert.Equal(t, domain.PaymentStatusSucceeded, created.Status)
		assert.Equal(t, map[string]string{"channel": "card"}, created.Metadata)

		got, err := repo.FindById(ctx, orgId, p.Id)
		require.NoError(t, err)
		assert.Equal(t, p.PspId, got.PspId)
	})

	t.Run("Update mutates status", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		created, err := repo.Create(ctx, newPayment(orgId))
		require.NoError(t, err)

		created.Status = domain.PaymentStatusRefunded
		updated, err := repo.Update(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusRefunded, updated.Status)

		reread, err := repo.FindById(ctx, orgId, created.Id)
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusRefunded, reread.Status)
	})

	t.Run("FindById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindById(ctx, orgId, "missing")
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})

	t.Run("FindByPspId finds by psp id within org", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		created, err := repo.Create(ctx, newPayment(orgId))
		require.NoError(t, err)

		got, err := repo.FindByPspId(ctx, orgId, created.PspId)
		require.NoError(t, err)
		assert.Equal(t, created.Id, got.Id)
	})

	t.Run("ListByPspId matches psp + psp_id across orgs", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		p := newPayment(orgId)
		_, err := repo.Create(ctx, p)
		require.NoError(t, err)

		got, err := repo.ListByPspId(ctx, domain.Paystack, p.PspId)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, p.Id, got[0].Id)
	})

	t.Run("FindBySubscriptionId paginates and counts", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")
		for range 3 {
			p := newPayment(orgId)
			p.SubscriptionId = subId
			_, err := repo.Create(ctx, p)
			require.NoError(t, err)
		}
		// One payment on a different subscription must be excluded.
		other := newPayment(orgId)
		other.SubscriptionId = lib.GenerateId("sub")
		_, err := repo.Create(ctx, other)
		require.NoError(t, err)

		p := domain.Pagination{Limit: 2, SortBy: "created_at", SortDirection: "asc"}
		payments, count, err := repo.FindBySubscriptionId(ctx, orgId, subId, p)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Len(t, payments, 2)
		for _, pay := range payments {
			assert.Equal(t, subId, pay.SubscriptionId)
		}
	})

	t.Run("CreateRefund round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		pay, err := repo.Create(ctx, newPayment(orgId))
		require.NoError(t, err)

		refund := domain.Refund{
			OrgId:       orgId,
			Id:          lib.GenerateId("ref"),
			PspRefundId: lib.GenerateId("psprf"),
			PaymentId:   pay.Id,
			Amount:      999,
			Currency:    "USD",
			Reason:      "customer request",
			RefundedAt:  time.Now().UTC().Truncate(time.Microsecond),
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err := repo.CreateRefund(ctx, refund)
		require.NoError(t, err)
		assert.Equal(t, refund.Id, created.Id)
		assert.Equal(t, int64(999), created.Amount)
		assert.Equal(t, pay.Id, created.PaymentId)
	})

	t.Run("org-scoping isolates payments", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		created, err := repo.Create(ctx, newPayment(orgA))
		require.NoError(t, err)

		_, err = repo.FindById(ctx, orgB, created.Id)
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})
}
