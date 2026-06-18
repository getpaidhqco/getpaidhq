//go:build integration

package postgrespgx

import (
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/storage/storagetest"
)

// pgxRepoSet builds the conformance RepoSet from a pgx pool against dsn.
func pgxRepoSet(t *testing.T, dsn string) storagetest.RepoSet {
	pool, err := NewDatabase(dsn, nil, "silent")
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
	}
}

// TestConformance runs the shared cross-driver suite against the pgx adapter.
func TestConformance(t *testing.T) {
	storagetest.RunConformance(t, pgxRepoSet)
}
