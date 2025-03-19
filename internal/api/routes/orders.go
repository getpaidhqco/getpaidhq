package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type OrderRoutes struct {
	logger          logger.Logger
	handler         lib.RequestHandler
	orderController controllers.OrderController
}

// Setup user routes
func (s OrderRoutes) Setup() {
	s.logger.Info("Setting up routes")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/orders", s.orderController.CreateOrder)
		api.POST("/orders/:id/complete", s.orderController.CompleteOrder)
		api.GET("/orders/:id/subscriptions", s.orderController.ListSubscriptions)
	}
}

// NewOrderRoutes creates new user controller
func NewOrderRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	orderController controllers.OrderController,
) OrderRoutes {
	return OrderRoutes{
		handler:         handler,
		logger:          logger,
		orderController: orderController,
	}
}
