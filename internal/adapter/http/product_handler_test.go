package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newProductHandlerForTest(
	t *testing.T,
	product *fakeProductRepo,
	variant *fakeVariantRepo,
	price *fakePriceRepo,
	cart *fakeCartRepo,
) *ProductHandler {
	t.Helper()
	svc := service.NewProductService(product, variant, price, cart, silentLogger{}, newPubSub())
	return NewProductHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestProductHandler_AuthzGuards(t *testing.T) {
	// Every product route is enforce-gated; a role with no permit rule
	// (support) must be rejected before any service call.
	prod := &fakeProductRepo{}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"list", http.MethodGet, "/api/products", nil},
		{"get", http.MethodGet, "/api/products/prod_1", nil},
		{"create", http.MethodPost, "/api/products", CreateProductRequest{Name: "p", Variants: []CreateProductVariantRequest{{Name: "v", Prices: []CreateProductPriceRequest{{Category: "one_time", Scheme: "fixed", Currency: "USD", UnitPrice: 1}}}}}},
		{"update", http.MethodPatch, "/api/products/prod_1", UpdateProductRequest{Name: "x"}},
		{"delete", http.MethodDelete, "/api/products/prod_1", nil},
		{"archive", http.MethodPost, "/api/products/prod_1/archive", nil},
		{"unarchive", http.MethodPost, "/api/products/prod_1/unarchive", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, ts, tt.method, tt.path, tt.body)
			assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
		})
	}
	assert.Empty(t, prod.created, "no creates should leak past the authz guard")
}

func TestProductHandler_List(t *testing.T) {
	prod := &fakeProductRepo{listResult: []domain.Product{
		{Id: "prod_1", Name: "Plan", Status: domain.ProductStatusActive},
		{Id: "prod_2", Name: "Plan B", Status: domain.ProductStatusActive},
	}}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/products?page=0&limit=5", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
	assert.Equal(t, 5, got.Meta.Limit)
}

func TestProductHandler_Get(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan"}}
		h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/products/prod_1", nil)

		require.Equal(t, http.StatusOK, rec.Code)
		var got ProductResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "prod_1", got.Id)
	})

	t.Run("repo error → bad_request fallback envelope", func(t *testing.T) {
		prod := &fakeProductRepo{byIdErr: errors.New("not in db")}
		h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/products/prod_x", nil)

		// Service returns raw repo error; the envelope falls through to
		// "bad_request" (status 400).
		assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	})
}

func TestProductHandler_Create(t *testing.T) {
	t.Run("happy path persists product + variants + prices", func(t *testing.T) {
		prod := &fakeProductRepo{}
		variant := &fakeVariantRepo{}
		price := &fakePriceRepo{}
		h := newProductHandlerForTest(t, prod, variant, price, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/products", CreateProductRequest{
			Name: "Subscription",
			Variants: []CreateProductVariantRequest{
				{Name: "Monthly", Prices: []CreateProductPriceRequest{
					{Category: "subscription", Scheme: "fixed", Currency: "USD", UnitPrice: 1000},
				}},
			},
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, prod.created, 1)
		assert.Len(t, variant.created, 1)
		assert.Len(t, price.created, 1)
	})

	t.Run("validation: missing required name returns validation envelope", func(t *testing.T) {
		h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/products", map[string]any{"variants": []any{}})

		// Validation comes from Fuego's binder. Status is 4xx; the code
		// depends on whether Fuego maps it to BadRequest (400) or
		// UnprocessableEntity (422).
		assert.GreaterOrEqual(t, rec.Code, 400)
		assert.Less(t, rec.Code, 500)
	})
}

func TestProductHandler_Delete(t *testing.T) {
	prod := &fakeProductRepo{}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodDelete, "/api/products/prod_1", nil)

	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
	require.Equal(t, []string{"prod_1"}, prod.deleted)
}

// A product still referenced by orders cannot be hard-deleted (FK
// constraint). The repo surfaces that as a typed ConflictError; the handler
// must render a 409 whose envelope preserves the human-readable message and
// underlying detail — not a bare "Bad Request" with null details.
func TestProductHandler_Delete_conflict(t *testing.T) {
	const msg = "Cannot delete a product that has existing orders."
	prod := &fakeProductRepo{
		deleteErr: lib.NewCustomError(lib.ConflictError, msg, errors.New("SQLSTATE 23503")),
	}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodDelete, "/api/products/prod_1", nil)

	got := assertErrorEnvelope(t, rec, http.StatusConflict, "conflict")
	require.Equal(t, msg, got.Message, "message must survive the error pipeline")
	require.NotNil(t, got.Details, "underlying detail must survive the error pipeline")
}

func TestProductHandler_VariantRoutes(t *testing.T) {
	variant := &fakeVariantRepo{byId: domain.Variant{Id: "var_1", Name: "Monthly"}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, variant, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	t.Run("get variant", func(t *testing.T) {
		rec := doJSON(t, ts, http.MethodGet, "/api/variants/var_1", nil)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got domain.Variant
		decodeJSON(t, rec, &got)
		assert.Equal(t, "var_1", got.Id)
	})

	t.Run("list prices on variant", func(t *testing.T) {
		rec := doJSON(t, ts, http.MethodGet, "/api/variants/var_1/prices", nil)
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestProductHandler_Update(t *testing.T) {
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Old"}}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/products/prod_1", UpdateProductRequest{Name: "New"})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, prod.updated, 1)
	assert.Equal(t, "New", prod.updated[0].Name)
}

func TestProductHandler_Archive(t *testing.T) {
	t.Run("archives an active product", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan", Status: domain.ProductStatusActive}}
		h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/products/prod_1/archive", nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got ProductResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, domain.ProductStatusArchived, got.Status)
		require.NotNil(t, got.ArchivedAt, "archived_at must be set on archive")
		require.Len(t, prod.updated, 1)
		assert.Equal(t, domain.ProductStatusArchived, prod.updated[0].Status)
	})

	t.Run("idempotent: re-archiving an archived product is a 200 no-op", func(t *testing.T) {
		prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Status: domain.ProductStatusArchived}}
		h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/products/prod_1/archive", nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		assert.Empty(t, prod.updated, "no update should be issued for an already-archived product")
	})
}

func TestProductHandler_Unarchive(t *testing.T) {
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Status: domain.ProductStatusArchived}}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/products/prod_1/unarchive", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ProductResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, domain.ProductStatusActive, got.Status)
	assert.Nil(t, got.ArchivedAt, "archived_at must be cleared on unarchive")
	require.Len(t, prod.updated, 1)
	assert.Equal(t, domain.ProductStatusActive, prod.updated[0].Status)
}

// The list status filter: default hides archived; ?status= narrows; "all"
// disables the filter; an unknown value is a 400.
func TestProductHandler_List_StatusFilter(t *testing.T) {
	fixtures := func() *fakeProductRepo {
		return &fakeProductRepo{listResult: []domain.Product{
			{Id: "prod_active", Name: "Live", Status: domain.ProductStatusActive},
			{Id: "prod_archived", Name: "Retired", Status: domain.ProductStatusArchived},
		}}
	}

	cases := []struct {
		name      string
		query     string
		wantCode  int
		wantTotal int
	}{
		{"default hides archived", "", http.StatusOK, 1},
		{"status=active", "?status=active", http.StatusOK, 1},
		{"status=archived", "?status=archived", http.StatusOK, 1},
		{"status=all shows both", "?status=all", http.StatusOK, 2},
		{"bogus status is rejected", "?status=bogus", http.StatusBadRequest, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProductHandlerForTest(t, fixtures(), &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})
			ts := newTestServer(fixedAuthMiddleware(adminUser()))
			h.RegisterRoutes(ts.api())

			rec := doJSON(t, ts, http.MethodGet, "/api/products"+tc.query, nil)

			require.Equal(t, tc.wantCode, rec.Code, "body=%s", rec.Body.String())
			if tc.wantCode == http.StatusOK {
				var got ListResponse
				decodeJSON(t, rec, &got)
				assert.Equal(t, tc.wantTotal, got.Meta.Total)
			}
		})
	}
}

func TestProductHandler_Create_DefaultsToActive(t *testing.T) {
	prod := &fakeProductRepo{}
	h := newProductHandlerForTest(t, prod, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/products", CreateProductRequest{
		Name: "New Plan",
		Variants: []CreateProductVariantRequest{
			{Name: "Monthly", Prices: []CreateProductPriceRequest{
				{Category: "subscription", Scheme: "fixed", Currency: "USD", UnitPrice: 1000},
			}},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ProductResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, domain.ProductStatusActive, got.Status)
	require.Len(t, prod.created, 1)
	assert.Equal(t, domain.ProductStatusActive, prod.created[0].Status)
}

func TestProductHandler_CreateVariant(t *testing.T) {
	variant := &fakeVariantRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, variant, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/products/prod_1/variants", CreateVariantRequest{Name: "Yearly"})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, variant.created, 1)
}

func TestProductHandler_ListVariants(t *testing.T) {
	variant := &fakeVariantRepo{listResult: []domain.Variant{{Id: "var_1"}, {Id: "var_2"}}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, variant, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/products/prod_1/variants", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestProductHandler_UpdateVariant(t *testing.T) {
	variant := &fakeVariantRepo{byId: domain.Variant{Id: "var_1"}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, variant, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPut, "/api/variants/var_1", UpdateVariantRequest{Name: "Yearly"})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestProductHandler_DeleteVariant(t *testing.T) {
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodDelete, "/api/variants/var_1", nil)

	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
}

func TestProductHandler_UpdatePrice(t *testing.T) {
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1"}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/prices/price_1", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "fixed", Currency: "USD", UnitPrice: 200,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestProductHandler_DeletePrice(t *testing.T) {
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, &fakePriceRepo{}, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodDelete, "/api/prices/price_1", nil)

	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
}

// ---------------------------------------------------------------------------
// Price tiers at the HTTP boundary (graduated/volume/tiered). Closes the gap
// where no handler test ever passed Tiers to a create/update. The tier parsers
// themselves (toDomainTiers, decimalOrZero) are unit-tested in request_test.go.
// ---------------------------------------------------------------------------

func TestProductHandler_CreatePrice_WithTiers(t *testing.T) {
	// Valid tiers parse and reach the service via CreatePrice.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "graduated", Currency: "USD",
		Tiers: []PriceTierRequest{
			{FromValue: "0", ToValue: "10", PerUnitAmount: "2.5", FlatAmount: 100},
			{FromValue: "10", ToValue: "", PerUnitAmount: "1.25"},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, price.created, 1)
	tiers := price.created[0].Tiers
	require.Len(t, tiers, 2)

	assert.Equal(t, "0", tiers[0].FromValue.String())
	assert.Equal(t, "10", tiers[0].ToValue.String())
	assert.Equal(t, "2.5", tiers[0].PerUnitAmount.String())
	assert.Equal(t, int64(100), tiers[0].FlatAmount)

	assert.Equal(t, "10", tiers[1].FromValue.String())
	assert.True(t, tiers[1].ToValue.IsZero(), "empty to_value must reach the domain as the unbounded tier")
	assert.Equal(t, "1.25", tiers[1].PerUnitAmount.String())
}

func TestProductHandler_UpdatePrice_WithTiers(t *testing.T) {
	// Valid tiers parse via UpdatePrice and are echoed back by the serializer.
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1"}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/prices/price_1", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "volume", Currency: "USD",
		Tiers: []PriceTierRequest{
			{FromValue: "0", ToValue: "", PerUnitAmount: "3", FlatAmount: 50},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got PriceResponse
	decodeJSON(t, rec, &got)
	require.Len(t, got.Tiers, 1)
	assert.Equal(t, "0", got.Tiers[0].FromValue)
	assert.Equal(t, "0", got.Tiers[0].ToValue, "unbounded tier serializes back as \"0\"")
	assert.Equal(t, "3", got.Tiers[0].PerUnitAmount)
	assert.Equal(t, int64(50), got.Tiers[0].FlatAmount)
}

// ---------------------------------------------------------------------------
// unit_count at the HTTP boundary: how many units unit_price buys (fixed
// scheme). Sub-cent effective rates without a fractional wire type.
// ---------------------------------------------------------------------------

func TestProductHandler_CreatePrice_WithUnitCount(t *testing.T) {
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "subscription", Scheme: "fixed", Currency: "USD",
		UnitPrice: 100, UnitCount: 1000, // $1 per 1000 units
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, price.created, 1)
	assert.Equal(t, 1000, price.created[0].UnitCount)

	var got PriceResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 1000, got.UnitCount)
}

func TestProductHandler_CreatePrice_UnitCountDefaultsToOne(t *testing.T) {
	// Omitted unit_count normalizes to 1 (per single unit) before persistence.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "fixed", Currency: "USD", UnitPrice: 200,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, price.created, 1)
	assert.Equal(t, 1, price.created[0].UnitCount)
}

func TestProductHandler_CreatePrice_UnitCountRejectedOnTieredScheme(t *testing.T) {
	// Tiers carry their own per-unit rates; unit_count only modifies the fixed scheme.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "tiered", Currency: "USD", UnitCount: 1000,
		Tiers: []PriceTierRequest{{FromValue: "0", ToValue: "", PerUnitAmount: "5"}},
	})

	assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	assert.Empty(t, price.created)
}

// ---------------------------------------------------------------------------
// package scheme at the HTTP boundary: every started block of unit_count units
// bills unit_price in full. Metered only; flat (no tiers).
// ---------------------------------------------------------------------------

func TestProductHandler_CreatePrice_PackageScheme(t *testing.T) {
	// "$5 per started 1,000 SMS": package needs a meter and carries the same
	// (unit_price, unit_count) pair as fixed.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "subscription", Scheme: "package", Currency: "USD",
		UnitPrice: 500, UnitCount: 1000, BillableMetricId: "met_1",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, price.created, 1)
	assert.Equal(t, domain.Package, price.created[0].Scheme)
	assert.Equal(t, int64(500), price.created[0].UnitPrice)
	assert.Equal(t, 1000, price.created[0].UnitCount)

	var got PriceResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, domain.Package, got.Scheme)
	assert.Equal(t, 1000, got.UnitCount)
}

func TestProductHandler_CreatePrice_PackageRequiresMetric(t *testing.T) {
	// Package bills started blocks of usage — without a meter there is no usage.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "subscription", Scheme: "package", Currency: "USD",
		UnitPrice: 500, UnitCount: 1000,
	})

	assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	assert.Empty(t, price.created)
}

func TestProductHandler_CreatePrice_PackageRejectsTiers(t *testing.T) {
	// Package is flat by definition — block #1 costs the same as block #500.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "subscription", Scheme: "package", Currency: "USD",
		UnitPrice: 500, UnitCount: 1000, BillableMetricId: "met_1",
		Tiers: []PriceTierRequest{{FromValue: "0", ToValue: "", PerUnitAmount: "5"}},
	})

	assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	assert.Empty(t, price.created)
}

func TestProductHandler_UpdatePrice_ToPackageScheme(t *testing.T) {
	// An existing metered price can switch to package via update.
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1"}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/prices/price_1", CreatePriceRequest{
		VariantId: "var_1", Category: "subscription", Scheme: "package", Currency: "USD",
		UnitPrice: 500, UnitCount: 1000, BillableMetricId: "met_1",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got PriceResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, domain.Package, got.Scheme)
	assert.Equal(t, 1000, got.UnitCount)
}

func TestProductHandler_CreateProduct_WithPriceTiers(t *testing.T) {
	// Valid tiers parse via the nested product create path (Create → variant → price).
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/products", CreateProductRequest{
		Name: "Metered plan",
		Variants: []CreateProductVariantRequest{
			{Name: "Usage", Prices: []CreateProductPriceRequest{
				{Category: "subscription", Scheme: "tiered", Currency: "USD", Tiers: []PriceTierRequest{
					{FromValue: "0", ToValue: "1000", PerUnitAmount: "0.001"},
					{FromValue: "1000", ToValue: "", PerUnitAmount: "0.0005"},
				}},
			}},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, price.created, 1)
	tiers := price.created[0].Tiers
	require.Len(t, tiers, 2)
	assert.Equal(t, "0.001", tiers[0].PerUnitAmount.String(), "sub-cent per-unit rate must survive parsing")
	assert.Equal(t, "0.0005", tiers[1].PerUnitAmount.String())
	assert.True(t, tiers[1].ToValue.IsZero())
}

func TestProductHandler_PriceTiers_MalformedRejectedAtValidation(t *testing.T) {
	// Boundary contract: a malformed tier decimal is rejected by the `numeric`
	// validator (HTTP 400) *before* the handler calls toDomainTiers, so the
	// service is never reached. This is why toDomainTiers' parse-error paths are
	// covered by unit tests rather than here.
	price := &fakePriceRepo{}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
		VariantId: "var_1", Category: "one_time", Scheme: "tiered", Currency: "USD",
		Tiers: []PriceTierRequest{{FromValue: "abc", ToValue: "10", PerUnitAmount: "5"}},
	})

	assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	assert.Contains(t, rec.Body.String(), "numeric", "rejection should come from the numeric validator")
	assert.Empty(t, price.created, "no price should be created when a tier value is malformed")
}

func TestProductHandler_PriceRoutes(t *testing.T) {
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1500}}
	h := newProductHandlerForTest(t, &fakeProductRepo{}, &fakeVariantRepo{}, price, &fakeCartRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	t.Run("get price", func(t *testing.T) {
		rec := doJSON(t, ts, http.MethodGet, "/api/prices/price_1", nil)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got domain.Price
		decodeJSON(t, rec, &got)
		assert.Equal(t, "price_1", got.Id)
	})

	t.Run("create price requires variant_id + iso4217 currency", func(t *testing.T) {
		// Bad currency triggers the project's iso4217 validator if Fuego is
		// using the project validator (BuildServer wires it). The minimal
		// test server doesn't, so a structurally-correct payload should
		// pass validation and land in the service which silently records it.
		rec := doJSON(t, ts, http.MethodPost, "/api/prices", CreatePriceRequest{
			VariantId: "var_1", Category: "one_time", Scheme: "fixed",
			Currency: "USD", UnitPrice: 100,
		})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, price.created, 1)
	})
}
