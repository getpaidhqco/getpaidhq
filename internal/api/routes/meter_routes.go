package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type MeterRoutes struct {
	logger         logger.Logger
	handler        lib.RequestHandler
	meterController controllers.MeterController
}

// Setup meter routes
func (m MeterRoutes) Setup() {
	m.logger.Info("Setting up meter routes")
	api := m.handler.Gin.Group("/api")
	{
		// Meter endpoints
		api.POST("/meters", m.meterController.Create)
		api.PUT("/meters/:id", m.meterController.Update)
		api.GET("/meters", m.meterController.List)
		api.GET("/meters/:id", m.meterController.Get)
		api.GET("/meters/slug/:slug", m.meterController.GetBySlug)
		api.DELETE("/meters/:id", m.meterController.Delete)
	}
}

// NewMeterRoutes creates new meter routes
func NewMeterRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	meterController controllers.MeterController,
) MeterRoutes {
	return MeterRoutes{
		logger:         logger,
		handler:        handler,
		meterController: meterController,
	}
}