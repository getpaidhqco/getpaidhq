package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type SubscriptionRoutes struct {
	logger                 lib.Logger
	handler                lib.RequestHandler
	subscriptionController controllers.SubscriptionController
}

// Setup user routes
func (s SubscriptionRoutes) Setup() {
	s.logger.Info("Setting up Subscription")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/subscriptions/:id", s.subscriptionController.Get)
		api.POST("/subscriptions", s.subscriptionController.Create)
		api.PUT("/subscriptions/:id/pause", s.subscriptionController.Pause)
		api.PUT("/subscriptions/:id/cancel", s.subscriptionController.Cancel)
		api.PUT("/subscriptions/:id/resume", s.subscriptionController.Resume)
		api.PATCH("/subscriptions/:id", s.subscriptionController.Update)
	}
}

// NewSubscriptionRoutes creates new user controller
func NewSubscriptionRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	subscriptionController controllers.SubscriptionController,
) SubscriptionRoutes {
	return SubscriptionRoutes{
		handler:                handler,
		logger:                 logger,
		subscriptionController: subscriptionController,
	}
}
