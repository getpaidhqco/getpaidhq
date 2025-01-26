package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type OrderRoutes struct {
	logger          lib.Logger
	handler         lib.RequestHandler
	orderController controllers.OrderController
}

// Setup user routes
func (s OrderRoutes) Setup() {
	s.logger.Info("Setting up routes")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/orders", s.orderController.CreateOrder)
	}
}

// NewOrderRoutes creates new user controller
func NewOrderRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	orderController controllers.OrderController,
) OrderRoutes {
	return OrderRoutes{
		handler:         handler,
		logger:          logger,
		orderController: orderController,
	}
}
