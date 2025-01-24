package controllers

import (
	"payloop/internal/lib"
	"payloop/internal/services"
)

// CartController data type
type CartController struct {
	service services.CartService
	logger  lib.Logger
}

// NewCartController creates new Cart controller
func NewCartController(CartService services.CartService, logger lib.Logger) CartController {
	return CartController{
		service: CartService,
		logger:  logger,
	}
}
