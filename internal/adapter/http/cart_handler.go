package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type CartHandler struct {
	cartService *service.CartService
	logger      port.Logger
}

func NewCartHandler(cartService *service.CartService, logger port.Logger) *CartHandler {
	return &CartHandler{cartService: cartService, logger: logger}
}

func (o *CartHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/carts", option.Tags("Carts"))
	fuego.Post(g, "/{id}/add", o.AddProduct, option.Summary("Add a product to a cart"))
	fuego.Post(g, "/{id}/remove", o.RemoveItem, option.Summary("Remove an item from a cart"))
}

func (o *CartHandler) AddProduct(c fuego.ContextWithBody[AddItemRequest]) (CartResponse, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return CartResponse{}, err
	}
	cart, err := o.cartService.AddProduct(c.Context(), domain.AddProductCommand{
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
	input, err := c.Body()
	if err != nil {
		return CartResponse{}, err
	}
	cart, err := o.cartService.RemoveItem(c.Context(), domain.RemoveItemCommand{
		OrgId:  input.OrgId,
		CartId: c.PathParam("id"),
		Id:     input.Id,
	})
	if err != nil {
		return CartResponse{}, NewApiErrorFromError(err)
	}
	return ToCartResponse(cart), nil
}
