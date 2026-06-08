package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// fakeCartRepo is shared by the cart and session tests in this package.
type fakeCartRepo struct {
	port.CartRepository
	cart      domain.Cart
	findErr   error
	createErr error
	created   []domain.Cart
	updated   []domain.Cart
}

func (r *fakeCartRepo) FindById(_ context.Context, _, _ string) (domain.Cart, error) {
	if r.findErr != nil {
		return domain.Cart{}, r.findErr
	}
	return r.cart, nil
}
func (r *fakeCartRepo) Create(_ context.Context, c domain.Cart) (domain.Cart, error) {
	if r.createErr != nil {
		return domain.Cart{}, r.createErr
	}
	if c.Id == "" {
		c.Id = "cart_generated"
	}
	r.created = append(r.created, c)
	return c, nil
}
func (r *fakeCartRepo) Update(_ context.Context, c domain.Cart) (domain.Cart, error) {
	r.updated = append(r.updated, c)
	r.cart = c
	return c, nil
}

func TestCartService_AddProduct(t *testing.T) {
	t.Run("appends a priced line item and persists the recalculated cart", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Plan"}}
		price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1000}}
		svc := NewCartService(cart, price, silentLogger{}, prod)

		got, err := svc.AddProduct(context.Background(), port.AddProductCommand{
			OrgId: "org_1", CartId: "cart_1", ProductId: "prod_1", PriceId: "price_1", Quantity: 3,
		})

		require.NoError(t, err)
		require.Len(t, got.Data.Items, 1)
		assert.Equal(t, int64(3000), got.Data.Items[0].Total, "unit price * qty")
		assert.Equal(t, int64(3000), got.Data.Total, "cart recalculated")
		require.Len(t, cart.updated, 1)
	})

	t.Run("missing cart is surfaced", func(t *testing.T) {
		cart := &fakeCartRepo{findErr: errors.New("no cart")}
		svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, &fakeProductRepo{})

		_, err := svc.AddProduct(context.Background(), port.AddProductCommand{OrgId: "org_1", CartId: "cart_x"})
		require.Error(t, err)
		assert.Empty(t, cart.updated)
	})

	t.Run("missing product is rejected", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{Id: "cart_1"}}
		prod := &fakeProductRepo{byIdErr: errors.New("no product")}
		svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, prod)

		_, err := svc.AddProduct(context.Background(), port.AddProductCommand{OrgId: "org_1", CartId: "cart_1", ProductId: "prod_x"})
		require.Error(t, err)
		assert.Empty(t, cart.updated)
	})

	t.Run("archived product cannot be added (409 conflict)", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusArchived}}
		price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1000}}
		svc := NewCartService(cart, price, silentLogger{}, prod)

		_, err := svc.AddProduct(context.Background(), port.AddProductCommand{
			OrgId: "org_1", CartId: "cart_1", ProductId: "prod_1", PriceId: "price_1", Quantity: 1,
		})

		require.Error(t, err)
		var ce lib.CustomError
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, lib.ConflictError, ce.Type)
		assert.Empty(t, cart.updated, "archived product must not reach the cart")
	})

	t.Run("missing price is rejected", func(t *testing.T) {
		cart := &fakeCartRepo{cart: domain.Cart{Id: "cart_1"}}
		prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1"}}
		price := &fakePriceRepo{byIdErr: errors.New("no price")}
		svc := NewCartService(cart, price, silentLogger{}, prod)

		_, err := svc.AddProduct(context.Background(), port.AddProductCommand{OrgId: "org_1", CartId: "cart_1", ProductId: "prod_1", PriceId: "price_x"})
		require.Error(t, err)
		assert.Empty(t, cart.updated)
	})
}

func cartWithItem(itemId string, qty int64) domain.Cart {
	c := domain.Cart{OrgId: "org_1", Id: "cart_1"}
	c.Data.Items = []domain.CartLineItem{{
		Id: itemId, ProductId: "prod_1", UnitPrice: 1000, Quantity: qty,
		SubTotal: 1000 * qty, Total: 1000 * qty,
	}}
	return c
}

func TestCartService_RemoveItem(t *testing.T) {
	cart := &fakeCartRepo{cart: cartWithItem("ci_1", 2)}
	svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, &fakeProductRepo{})

	got, err := svc.RemoveItem(context.Background(), port.RemoveItemCommand{OrgId: "org_1", CartId: "cart_1", Id: "ci_1"})

	require.NoError(t, err)
	assert.Empty(t, got.Data.Items, "the item was removed")
	assert.Equal(t, int64(0), got.Data.Total)
	require.Len(t, cart.updated, 1)
}

func TestCartService_AdjustItem(t *testing.T) {
	t.Run("adjusts the quantity by matching the line item id and persists", func(t *testing.T) {
		// AdjustQuantity matches on the line item Id, so the command's ProductId
		// must equal the stored item Id for the adjustment to take.
		cart := &fakeCartRepo{cart: cartWithItem("ci_1", 1)}
		svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, &fakeProductRepo{})

		got, err := svc.AdjustItem(context.Background(), port.AdjustCommand{OrgId: "org_1", CartId: "cart_1", ProductId: "ci_1", Quantity: 5})

		require.NoError(t, err)
		require.Len(t, got.Data.Items, 1)
		assert.Equal(t, int64(5), got.Data.Items[0].Quantity)
		assert.Equal(t, int64(5000), got.Data.Total, "recalculated after adjust")
		require.Len(t, cart.updated, 1)
	})

	t.Run("unknown item is rejected without persistence", func(t *testing.T) {
		cart := &fakeCartRepo{cart: cartWithItem("ci_1", 1)}
		svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, &fakeProductRepo{})

		_, err := svc.AdjustItem(context.Background(), port.AdjustCommand{OrgId: "org_1", CartId: "cart_1", ProductId: "nope", Quantity: 5})

		require.ErrorIs(t, err, domain.ErrItemNotFound)
		assert.Empty(t, cart.updated)
	})
}

func TestCartService_GetCart(t *testing.T) {
	cart := &fakeCartRepo{cart: domain.Cart{OrgId: "org_1", Id: "cart_1"}}
	svc := NewCartService(cart, &fakePriceRepo{}, silentLogger{}, &fakeProductRepo{})

	got, err := svc.GetCart("org_1", "cart_1")
	require.NoError(t, err)
	assert.Equal(t, "cart_1", got.Id)
}
