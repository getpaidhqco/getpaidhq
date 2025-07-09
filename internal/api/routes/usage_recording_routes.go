package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type UsageRecordingRoutes struct {
	logger                   logger.Logger
	handler                  lib.RequestHandler
	usageRecordingController controllers.UsageRecordingController
}

// Setup usage recording routes
func (u UsageRecordingRoutes) Setup() {
	u.logger.Info("Setting up usage recording routes")
	api := u.handler.Gin.Group("/api")
	{
		// CloudEvents usage endpoint (new standard)
		api.POST("/usage-events", u.usageRecordingController.RecordUsage)

		// Legacy usage recording endpoints
		api.POST("/usage-records", u.usageRecordingController.RecordUsage)
		api.GET("/usage-records", u.usageRecordingController.ListUsageRecords)
  api.GET("/usage-records/:id", u.usageRecordingController.GetUsageEvent)
  api.DELETE("/usage-records/:id", u.usageRecordingController.DeleteUsageEvent)

		// Usage summary endpoints
		api.GET("/subscriptions/:id/usage", u.usageRecordingController.GetSubscriptionUsage)
	}
}

// NewUsageRecordingRoutes creates new usage recording routes
func NewUsageRecordingRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	usageRecordingController controllers.UsageRecordingController,
) UsageRecordingRoutes {
	return UsageRecordingRoutes{
		logger:                   logger,
		handler:                  handler,
		usageRecordingController: usageRecordingController,
	}
}
