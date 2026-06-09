package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ---- fakes specific to ProductService ----

type fakeProductRepo struct {
	port.ProductRepository
	byId         domain.Product
	byIdErr      error
	createErr    error
	created      []domain.Product
	updated      []domain.Product
	deleted      []string
	listed       []domain.Product
	listStatuses []domain.ProductStatus
}

func (r *fakeProductRepo) Create(_ context.Context, p domain.Product) (domain.Product, error) {
	if r.createErr != nil {
		return domain.Product{}, r.createErr
	}
	r.created = append(r.created, p)
	return p, nil
}
func (r *fakeProductRepo) FindById(_ context.Context, _, _ string) (domain.Product, error) {
	if r.byIdErr != nil {
		return domain.Product{}, r.byIdErr
	}
	return r.byId, nil
}
func (r *fakeProductRepo) Find(_ context.Context, _ string, _ domain.Pagination, statuses []domain.ProductStatus) ([]domain.Product, int, error) {
	r.listStatuses = statuses
	return r.listed, len(r.listed), nil
}
func (r *fakeProductRepo) Update(_ context.Context, p domain.Product) (domain.Product, error) {
	r.updated = append(r.updated, p)
	return p, nil
}
func (r *fakeProductRepo) Delete(_ context.Context, _, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type fakeVariantRepo struct {
	port.VariantRepository
	byId    domain.Variant
	byIdErr error
	created []domain.Variant
	updated []domain.Variant
	deleted []string
}

func (r *fakeVariantRepo) Create(_ context.Context, v domain.Variant) (domain.Variant, error) {
	r.created = append(r.created, v)
	return v, nil
}
func (r *fakeVariantRepo) FindById(_ context.Context, _, _ string) (domain.Variant, error) {
	if r.byIdErr != nil {
		return domain.Variant{}, r.byIdErr
	}
	return r.byId, nil
}
func (r *fakeVariantRepo) Update(_ context.Context, v domain.Variant) (domain.Variant, error) {
	r.updated = append(r.updated, v)
	return v, nil
}
func (r *fakeVariantRepo) Delete(_ context.Context, _, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type fakePriceRepo struct {
	port.PriceRepository
	byId    domain.Price
	byIdErr error
	created []domain.Price
	updated []domain.Price
	deleted []string
}

func (r *fakePriceRepo) Create(_ context.Context, p domain.Price) (domain.Price, error) {
	r.created = append(r.created, p)
	return p, nil
}
func (r *fakePriceRepo) FindById(_ context.Context, _, _ string) (domain.Price, error) {
	if r.byIdErr != nil {
		return domain.Price{}, r.byIdErr
	}
	return r.byId, nil
}
func (r *fakePriceRepo) Update(_ context.Context, p domain.Price) (domain.Price, error) {
	r.updated = append(r.updated, p)
	return p, nil
}
func (r *fakePriceRepo) Delete(_ context.Context, _, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *fakePriceRepo) FindByIds(_ context.Context, _ string, ids []string) ([]domain.Price, error) {
	if r.byIdErr != nil {
		return nil, r.byIdErr
	}
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []domain.Price
	if want[r.byId.Id] {
		out = append(out, r.byId)
	}
	return out, nil
}

func (r *fakePriceRepo) FindByVariantIds(context.Context, string, []string) ([]domain.Price, error) {
	return nil, nil
}

func newProductService(prod port.ProductRepository, vr port.VariantRepository, pr port.PriceRepository, ps port.PubSub) *ProductService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	return NewProductService(prod, vr, pr, nil, silentLogger{}, ps)
}

func TestProductService_CreateProduct(t *testing.T) {
	t.Run("creates product, variants, prices and publishes", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Plan"}}
		vr := &fakeVariantRepo{}
		pr := &fakePriceRepo{}
		ps := &recordingPubSub{}
		svc := newProductService(prod, vr, pr, ps)

		got, err := svc.CreateProduct(context.Background(), "org_1", port.CreateProductInput{
			Name: "Plan",
			Variants: []port.CreateProductVariantInput{
				{Name: "Monthly", Prices: []port.CreateProductPriceInput{{Currency: "USD", UnitPrice: 1000}}},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "prod_1", got.Id)
		require.Len(t, prod.created, 1)
		require.Len(t, vr.created, 1)
		require.Len(t, pr.created, 1)
		assert.True(t, ps.hasTopic(port.TopicProductCreated))
	})

	t.Run("product create failure stops and does not publish", func(t *testing.T) {
		prod := &fakeProductRepo{createErr: errors.New("db down")}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		_, err := svc.CreateProduct(context.Background(), "org_1", port.CreateProductInput{Name: "Plan"})

		require.Error(t, err)
		assert.False(t, ps.hasTopic(port.TopicProductCreated))
	})
}

func TestProductService_UpdateProduct(t *testing.T) {
	t.Run("overwrites fields and publishes", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Old"}}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		got, err := svc.UpdateProduct(context.Background(), "org_1", "prod_1", port.UpdateProductInput{Name: "New", Description: "d"})

		require.NoError(t, err)
		assert.Equal(t, "New", got.Name)
		require.Len(t, prod.updated, 1)
		assert.True(t, ps.hasTopic(port.TopicProductUpdated))
	})

	t.Run("not found is rejected", func(t *testing.T) {
		prod := &fakeProductRepo{byIdErr: errors.New("missing")}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		_, err := svc.UpdateProduct(context.Background(), "org_1", "prod_x", port.UpdateProductInput{Name: "New"})

		require.Error(t, err)
		assert.Empty(t, prod.updated)
	})
}

func TestProductService_DeleteProduct(t *testing.T) {
	prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1"}}
	ps := &recordingPubSub{}
	svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

	err := svc.DeleteProduct(context.Background(), "org_1", "prod_1")

	require.NoError(t, err)
	assert.Equal(t, []string{"prod_1"}, prod.deleted)
	assert.True(t, ps.hasTopic(port.TopicProductDeleted))
}

func TestProductService_CreateProductPrice(t *testing.T) {
	t.Run("defaults empty intervals to none and publishes", func(t *testing.T) {
		pr := &fakePriceRepo{}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, &fakeVariantRepo{}, pr, ps)

		got, err := svc.CreateProductPrice(context.Background(), port.CreatePriceInput{OrgId: "org_1", Currency: "USD", UnitPrice: 500})

		require.NoError(t, err)
		assert.Equal(t, domain.BillingIntervalNone, got.BillingInterval)
		assert.Equal(t, domain.BillingIntervalNone, got.TrialInterval)
		require.Len(t, pr.created, 1)
		assert.True(t, ps.hasTopic(port.TopicPriceCreated))
	})
}

func TestProductService_Variants(t *testing.T) {
	t.Run("create publishes variant.created", func(t *testing.T) {
		vr := &fakeVariantRepo{}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, vr, &fakePriceRepo{}, ps)

		got, err := svc.CreateVariant(context.Background(), "org_1", "prod_1", port.CreateVariantInput{Name: "Monthly"})

		require.NoError(t, err)
		assert.Equal(t, "prod_1", got.ProductId)
		assert.True(t, ps.hasTopic(port.TopicVariantCreated))
	})

	t.Run("update overwrites and publishes", func(t *testing.T) {
		vr := &fakeVariantRepo{byId: domain.Variant{Id: "var_1", Name: "Old"}}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, vr, &fakePriceRepo{}, ps)

		got, err := svc.UpdateVariant(context.Background(), "org_1", "var_1", port.UpdateVariantInput{Name: "New"})

		require.NoError(t, err)
		assert.Equal(t, "New", got.Name)
		assert.True(t, ps.hasTopic(port.TopicVariantUpdated))
	})

	t.Run("delete publishes variant.deleted", func(t *testing.T) {
		vr := &fakeVariantRepo{byId: domain.Variant{Id: "var_1"}}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, vr, &fakePriceRepo{}, ps)

		err := svc.DeleteVariant(context.Background(), "org_1", "var_1")

		require.NoError(t, err)
		assert.Equal(t, []string{"var_1"}, vr.deleted)
		assert.True(t, ps.hasTopic(port.TopicVariantDeleted))
	})
}

func TestProductService_Prices(t *testing.T) {
	t.Run("update overwrites and publishes", func(t *testing.T) {
		pr := &fakePriceRepo{byId: domain.Price{Id: "price_1", Label: "Old"}}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, &fakeVariantRepo{}, pr, ps)

		got, err := svc.UpdatePrice(context.Background(), "org_1", "price_1", port.CreatePriceInput{Label: "New", Currency: "USD", UnitPrice: 700})

		require.NoError(t, err)
		assert.Equal(t, "New", got.Label)
		assert.Equal(t, domain.BillingIntervalNone, got.BillingInterval)
		assert.True(t, ps.hasTopic(port.TopicPriceUpdated))
	})

	t.Run("delete publishes price.deleted", func(t *testing.T) {
		pr := &fakePriceRepo{byId: domain.Price{Id: "price_1"}}
		ps := &recordingPubSub{}
		svc := newProductService(&fakeProductRepo{}, &fakeVariantRepo{}, pr, ps)

		err := svc.DeletePrice(context.Background(), "org_1", "price_1")

		require.NoError(t, err)
		assert.Equal(t, []string{"price_1"}, pr.deleted)
		assert.True(t, ps.hasTopic(port.TopicPriceDeleted))
	})

	t.Run("get surfaces not-found", func(t *testing.T) {
		pr := &fakePriceRepo{byIdErr: errors.New("missing")}
		svc := newProductService(&fakeProductRepo{}, &fakeVariantRepo{}, pr, nil)

		_, err := svc.GetPrice(context.Background(), "org_1", "price_x")
		require.Error(t, err)
	})
}

func TestProductService_ArchiveProduct(t *testing.T) {
	t.Run("archives an active product, sets archived_at, and publishes", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusActive}}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		got, err := svc.ArchiveProduct(context.Background(), "org_1", "prod_1")

		require.NoError(t, err)
		assert.Equal(t, domain.ProductStatusArchived, got.Status)
		require.NotNil(t, got.ArchivedAt)
		require.Len(t, prod.updated, 1)
		assert.Equal(t, domain.ProductStatusArchived, prod.updated[0].Status)
		assert.True(t, ps.hasTopic(port.TopicProductArchived))
	})

	t.Run("idempotent: already archived does not update or publish", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusArchived}}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		got, err := svc.ArchiveProduct(context.Background(), "org_1", "prod_1")

		require.NoError(t, err)
		assert.Equal(t, domain.ProductStatusArchived, got.Status)
		assert.Empty(t, prod.updated)
		assert.False(t, ps.hasTopic(port.TopicProductArchived))
	})
}

func TestProductService_UnarchiveProduct(t *testing.T) {
	t.Run("returns an archived product to active and clears archived_at", func(t *testing.T) {
		now := time.Now().UTC()
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusArchived, ArchivedAt: &now}}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		got, err := svc.UnarchiveProduct(context.Background(), "org_1", "prod_1")

		require.NoError(t, err)
		assert.Equal(t, domain.ProductStatusActive, got.Status)
		assert.Nil(t, got.ArchivedAt)
		require.Len(t, prod.updated, 1)
		assert.True(t, ps.hasTopic(port.TopicProductUnarchived))
	})

	t.Run("idempotent: already active does not update or publish", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusActive}}
		ps := &recordingPubSub{}
		svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, ps)

		_, err := svc.UnarchiveProduct(context.Background(), "org_1", "prod_1")

		require.NoError(t, err)
		assert.Empty(t, prod.updated)
		assert.False(t, ps.hasTopic(port.TopicProductUnarchived))
	})
}

func TestProductService_CreateProduct_DefaultsActive(t *testing.T) {
	prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1"}}
	svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, nil)

	_, err := svc.CreateProduct(context.Background(), "org_1", port.CreateProductInput{Name: "Plan"})

	require.NoError(t, err)
	require.Len(t, prod.created, 1)
	assert.Equal(t, domain.ProductStatusActive, prod.created[0].Status)
}

func TestProductService_List_PassesStatusFilter(t *testing.T) {
	prod := &fakeProductRepo{}
	svc := newProductService(prod, &fakeVariantRepo{}, &fakePriceRepo{}, nil)

	_, _, err := svc.List(context.Background(), "org_1", domain.Pagination{Page: 1, Limit: 10}, []domain.ProductStatus{domain.ProductStatusArchived})

	require.NoError(t, err)
	assert.Equal(t, []domain.ProductStatus{domain.ProductStatusArchived}, prod.listStatuses)
}
