//go:build integration

package postgrespgx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/db/storagetest"
)

// pgxRepoSet builds the conformance RepoSet from a pgx pool against dsn.
func pgxRepoSet(t *testing.T, dsn string) storagetest.RepoSet {
	pool, err := NewDatabase(dsn, nil)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return storagetest.RepoSet{
		Org:          NewOrgRepo(pool),
		Customer:     NewCustomerRepo(pool),
		Product:      NewProductRepo(pool),
		Variant:      NewVariantRepo(pool),
		Price:        NewPriceRepo(pool),
		Cart:         NewCartRepo(pool),
		Order:        NewOrderRepo(pool),
		Subscription: NewSubscriptionRepo(pool),
		Payment:      NewPaymentRepo(pool),
		Setting:      NewSettingRepo(pool),
		Idempotency:  NewIdempotencyKeyRepo(pool),
		Tx:           NewTxManager(pool),

		Invoice:           NewInvoiceRepo(pool),
		Dunning:           NewDunningRepo(pool),
		Coupon:            NewCouponRepo(pool),
		CouponCode:        NewCouponCodeRepo(pool),
		CouponReservation: NewCouponReservationRepo(pool),
		Discount:          NewDiscountRepo(pool),
		Meter:             NewMeterRepo(pool),
		Metadata:          NewMetadataStoreRepo(pool),
		Psp:               NewPspRepo(pool),
		ApiKey:            NewApiKeyRepo(pool),
		Webhook:           NewWebhookSubscriptionRepo(pool),
		Session:           NewSessionRepo(pool),
		PaymentMethod:     NewPaymentMethodRepo(pool),
		EventStore:        NewEventStore(pool),
		IdempotencyStore:  NewIdempotencyStore(pool, time.Minute, 24*time.Hour),
		Outbox:            NewOutboxRepo(pool),
	}
}

// TestConformance runs the shared cross-driver suite against the pgx adapter.
func TestConformance(t *testing.T) {
	storagetest.RunConformance(t, pgxRepoSet)
}
