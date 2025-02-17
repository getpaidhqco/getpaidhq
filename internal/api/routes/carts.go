package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type CartsRoutes struct {
	logger          logger.Logger
	handler         lib.RequestHandler
	cartsController controllers.CartController
}

// Setup user routes
func (s CartsRoutes) Setup() {
	s.logger.Info("Setting up Carts")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/carts/:id/add", s.cartsController.AddProduct)
		api.POST("/carts/:id/remove", s.cartsController.RemoveItem)
	}
}

// NewCartsRoutes creates new user controller
func NewCartsRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	cartsController controllers.CartController,
) CartsRoutes {
	return CartsRoutes{
		handler:         handler,
		logger:          logger,
		cartsController: cartsController,
	}
}
