package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
)

func newCartHandlerForTest(
	cart *fakeCartRepo,
	price *fakePriceRepo,
	product *fakeProductRepo,
) *CartHandler {
	svc := service.NewCartService(cart, price, silentLogger{}, product)
	return NewCartHandler(svc, silentLogger{})
}

func TestCartHandler_AddProduct(t *testing.T) {
	// The cart handler doesn't enforce cedar today, so we drive it through the
	// fixed-auth middleware only to satisfy AuthUserFrom for the OrgId pull.
	t.Run("happy path appends an item and returns the recalculated cart", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Plan"}}
		price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1500}}
		h := newCartHandlerForTest(cart, price, prod)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/add", AddItemRequest{
			ProductId: "prod_1", PriceId: "price_1", Quantity: 2,
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, cart.updated, 1, "cart was persisted")
		assert.Len(t, cart.updated[0].Data.Items, 1)
	})

	t.Run("validation: missing product_id rejected by Fuego/validator", func(t *testing.T) {
		// quantity is required-tagged via the int validator; missing both
		// product_id and price_id triggers the validation envelope.
		h := newCartHandlerForTest(&fakeCartRepo{}, &fakePriceRepo{}, &fakeProductRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/add", map[string]any{})

		// Fuego applies its built-in body validator (no custom validator set)
		// and emits a bad-request error; the project serializer maps the
		// Fuego HTTPError → ApiError.
		assert.GreaterOrEqual(t, rec.Code, 400)
		assert.Less(t, rec.Code, 500)
	})

	t.Run("missing product surfaces a not_found envelope", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{Id: "cart_1"}}
		prod := &fakeProductRepo{byIdErr: errors.New("not in db")}
		h := newCartHandlerForTest(cart, &fakePriceRepo{}, prod)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/add", AddItemRequest{
			ProductId: "prod_x", PriceId: "price_1", Quantity: 1,
		})

		assertErrorEnvelope(t, rec, http.StatusNotFound, "not_found")
	})
}

func TestCartHandler_RemoveItem(t *testing.T) {
	cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
	h := newCartHandlerForTest(cart, &fakePriceRepo{}, &fakeProductRepo{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/remove", RemoveItemRequest{
		OrgId: "org_1", Id: "ci_1",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	// Cart.Update is called even when the item is not present (RemoveItem is
	// a no-op that still persists). The handler doesn't error on that.
	require.Len(t, cart.updated, 1)
}
