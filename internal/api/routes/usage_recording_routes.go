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
		// Usage recording endpoints
		api.POST("/usage-records", u.usageRecordingController.RecordUsage)
		api.POST("/usage-records/batch", u.usageRecordingController.BatchRecordUsage)
		api.GET("/usage-records", u.usageRecordingController.ListUsageRecords)
		api.GET("/usage-records/:id", u.usageRecordingController.GetUsageRecord)
		api.DELETE("/usage-records/:id", u.usageRecordingController.DeleteUsageRecord)

		// Usage summary endpoints
		api.GET("/subscription-items/:id/usage-summary", u.usageRecordingController.GetUsageSummary)
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