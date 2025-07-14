package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type SubscriptionRoutes struct {
	logger                 logger.Logger
	handler                lib.RequestHandler
	subscriptionController controllers.SubscriptionController
}

// Setup user routes
func (s SubscriptionRoutes) Setup() {
	s.logger.Info("Setting up Subscription")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/subscriptions", s.subscriptionController.Create)
		api.GET("/subscriptions", s.subscriptionController.List)
		api.GET("/subscriptions/:id", s.subscriptionController.Get)
		api.GET("/subscriptions/:id/payments", s.subscriptionController.ListPayments)
		api.PUT("/subscriptions/:id/pause", s.subscriptionController.Pause)
		api.PUT("/subscriptions/:id/cancel", s.subscriptionController.Cancel)
		api.PUT("/subscriptions/:id/resume", s.subscriptionController.Resume)
		api.PUT("/subscriptions/:id/billing-anchor", s.subscriptionController.UpdateBillingAnchor)
		api.PUT("/subscriptions/:id/change-plan", s.subscriptionController.ChangePlan)
		api.PATCH("/subscriptions/:id", s.subscriptionController.Update)
	}

}

// NewSubscriptionRoutes creates new user controller
func NewSubscriptionRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	subscriptionController controllers.SubscriptionController,
) SubscriptionRoutes {
	return SubscriptionRoutes{
		handler:                handler,
		logger:                 logger,
		subscriptionController: subscriptionController,
	}
}
