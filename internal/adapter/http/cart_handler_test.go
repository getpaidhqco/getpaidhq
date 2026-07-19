package handler

import (
	"errors"
	errors2 "getpaidhq/internal/lib/errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
)

func newCartHandlerForTest(
	t *testing.T,
	cart *fakeCartRepo,
	price *fakePriceRepo,
	product *fakeProductRepo,
) *CartHandler {
	t.Helper()
	svc := service.NewCartService(cart, price, silentLogger{}, product)
	return NewCartHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestCartHandler_AddProduct(t *testing.T) {
	t.Run("happy path appends an item and returns the recalculated cart", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Plan"}}
		price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1500}}
		h := newCartHandlerForTest(t, cart, price, prod)

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
		h := newCartHandlerForTest(t, &fakeCartRepo{}, &fakePriceRepo{}, &fakeProductRepo{})

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
		h := newCartHandlerForTest(t, cart, &fakePriceRepo{}, prod)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/add", AddItemRequest{
			ProductId: "prod_x", PriceId: "price_1", Quantity: 1,
		})

		assertErrorEnvelope(t, rec, http.StatusNotFound, "not_found")
	})

	t.Run("support role is denied by cedar — 403 envelope", func(t *testing.T) {
		// support has no permit rule for AddProductToCart → cedar denies
		// before the service runs.
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		h := newCartHandlerForTest(t, cart, &fakePriceRepo{}, &fakeProductRepo{})

		ts := newTestServer(fixedAuthMiddleware(supportUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/add", AddItemRequest{
			ProductId: "prod_1", PriceId: "price_1", Quantity: 1,
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(errors2.ForbiddenError))
		assert.Empty(t, cart.updated, "service must not run when authz denies")
	})
}

func TestCartHandler_RemoveItem(t *testing.T) {
	t.Run("happy path: OrgId comes from auth user, not from body", func(t *testing.T) {
		// The handler previously took OrgId from the request body, which let
		// any caller act against an arbitrary org by passing a different
		// OrgId. Now it ignores body.OrgId and uses authUser.OrgId. This test
		// passes a wrong OrgId in the body to pin that behavior.
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		h := newCartHandlerForTest(t, cart, &fakePriceRepo{}, &fakeProductRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser())) // ownerUser is in org_1
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/remove", RemoveItemRequest{
			OrgId: "other_org", // attacker-controlled value — must be ignored
			Id:    "ci_1",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, cart.updated, 1)
		assert.Equal(t, "org_1", cart.updated[0].OrgId, "service called with authUser.OrgId, not body.OrgId")
	})

	t.Run("support role is denied by cedar — 403 envelope", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		h := newCartHandlerForTest(t, cart, &fakePriceRepo{}, &fakeProductRepo{})

		ts := newTestServer(fixedAuthMiddleware(supportUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/carts/cart_1/remove", RemoveItemRequest{Id: "ci_1"})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(errors2.ForbiddenError))
		assert.Empty(t, cart.updated)
	})
}
