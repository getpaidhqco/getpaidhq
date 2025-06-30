package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type DunningRoutes struct {
	logger           logger.Logger
	handler          lib.RequestHandler
	dunningController controllers.DunningController
}

// Setup dunning routes
func (d DunningRoutes) Setup() {
	d.logger.Info("Setting up Dunning routes")
	api := d.handler.Gin.Group("/api")
	{
		// Campaign routes
		api.GET("/dunning/campaigns", d.dunningController.ListCampaigns)
		api.GET("/dunning/campaigns/:id", d.dunningController.GetCampaign)
		api.PATCH("/dunning/campaigns/:id", d.dunningController.UpdateCampaign)
		
		// Campaign attempts routes
		api.GET("/dunning/campaigns/:id/attempts", d.dunningController.ListCampaignAttempts)
		api.POST("/dunning/campaigns/:id/attempts", d.dunningController.TriggerManualAttempt)
		
		// Campaign communications routes
		api.GET("/dunning/campaigns/:id/communications", d.dunningController.ListCampaignCommunications)
		
		// Payment token routes
		api.POST("/payment-tokens/verify", d.dunningController.VerifyPaymentToken)
		api.POST("/payment-tokens/activate", d.dunningController.ActivatePaymentToken)
		api.POST("/admin/subscriptions/:id/payment-tokens", d.dunningController.CreatePaymentToken)
		
		// Configuration routes
		api.GET("/dunning/configurations", d.dunningController.ListConfigurations)
		api.GET("/dunning/configurations/:id", d.dunningController.GetConfiguration)
		api.POST("/dunning/configurations", d.dunningController.CreateConfiguration)
		api.PATCH("/dunning/configurations/:id", d.dunningController.UpdateConfiguration)
		
		// Customer dunning history routes
		api.GET("/customers/:id/dunning-history", d.dunningController.GetCustomerDunningHistory)
	}
}

// NewDunningRoutes creates new dunning routes
func NewDunningRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	dunningController controllers.DunningController,
) DunningRoutes {
	return DunningRoutes{
		handler:          handler,
		logger:           logger,
		dunningController: dunningController,
	}
}