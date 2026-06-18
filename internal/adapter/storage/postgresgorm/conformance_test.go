//go:build integration

package postgresgorm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/storage/storagetest"
)

// gormRepoSet builds the conformance RepoSet from a GORM handle against dsn.
func gormRepoSet(t *testing.T, dsn string) storagetest.RepoSet {
	db, err := NewDatabase(dsn, nil, "silent")
	require.NoError(t, err)
	return storagetest.RepoSet{
		Org:          NewOrgRepo(db),
		Customer:     NewCustomerRepo(db),
		Product:      NewProductRepo(db),
		Variant:      NewVariantRepo(db),
		Price:        NewPriceRepo(db),
		Cart:         NewCartRepo(db),
		Order:        NewOrderRepo(db),
		Subscription: NewSubscriptionRepo(db),
		Payment:      NewPaymentRepo(db),
		Setting:      NewSettingRepo(db),
		Idempotency:  NewIdempotencyKeyRepo(db),
		Tx:           NewTxManager(db),
	}
}

// TestConformance runs the shared cross-driver suite against the GORM adapter,
// proving the suite is faithful and that both adapters produce identical results.
func TestConformance(t *testing.T) {
	storagetest.RunConformance(t, gormRepoSet)
}
