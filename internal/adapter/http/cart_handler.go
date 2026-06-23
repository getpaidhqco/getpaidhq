package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type CartHandler struct {
	cartService *service.CartService
	logger      port.Logger
	authz       port.Authz
}

func NewCartHandler(cartService *service.CartService, logger port.Logger, authz port.Authz) *CartHandler {
	return &CartHandler{cartService: cartService, logger: logger, authz: authz}
}

func (o *CartHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/carts", option.Tags("Carts"))
	fuego.Post(g, "/{id}/add", o.AddProduct, option.Summary("Add a product to a cart"), option.OperationID("addProductToCart"))
	fuego.Post(g, "/{id}/remove", o.RemoveItem, option.Summary("Remove an item from a cart"), option.OperationID("removeItemFromCart"))
}

func (o *CartHandler) AddProduct(c fuego.ContextWithBody[AddItemRequest]) (CartResponse, error) {
	authUser := AuthUserFrom(c)
	if !o.authz.Enforce(authUser, port.ActionAddProductToCart, "") {
		return CartResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return CartResponse{}, err
	}
	cart, err := o.cartService.AddProduct(c.Context(), port.AddProductCommand{
		OrgId:     authUser.OrgId,
		CartId:    c.PathParam("id"),
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	})
	if err != nil {
		return CartResponse{}, NewApiErrorFromError(err)
	}
	return ToCartResponse(cart), nil
}

func (o *CartHandler) RemoveItem(c fuego.ContextWithBody[RemoveItemRequest]) (CartResponse, error) {
	authUser := AuthUserFrom(c)
	if !o.authz.Enforce(authUser, port.ActionRemoveItemFromCart, "") {
		return CartResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return CartResponse{}, err
	}
	// OrgId comes from the authenticated user, never from the request body.
	// The previous code took `input.OrgId` from the deserialized payload,
	// which let any authenticated user remove items from a cart in any org
	// by passing a different OrgId.
	cart, err := o.cartService.RemoveItem(c.Context(), port.RemoveItemCommand{
		OrgId:  authUser.OrgId,
		CartId: c.PathParam("id"),
		Id:     input.Id,
	})
	if err != nil {
		return CartResponse{}, NewApiErrorFromError(err)
	}
	return ToCartResponse(cart), nil
}
