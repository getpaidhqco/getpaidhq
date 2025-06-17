package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type SettingsRoutes struct {
	logger             logger.Logger
	handler            lib.RequestHandler
	settingsController controllers.SettingsController
}

// Setup settings routes
func (s SettingsRoutes) Setup() {
	s.logger.Info("Setting up Settings routes")
	api := s.handler.Gin.Group("/api")
	{
		// Routes for settings with parent_id in the path
		api.GET("/settings/:parent_id", s.settingsController.List)
		api.GET("/settings/:parent_id/:id", s.settingsController.Get)
		api.PUT("/settings/:parent_id/:id", s.settingsController.Update)
		api.DELETE("/settings/:parent_id/:id", s.settingsController.Delete)

		// Route for creating settings (parent_id in the request body)
		api.POST("/settings/:parent_id", s.settingsController.Update)
	}
}

// NewSettingsRoutes creates new settings routes
func NewSettingsRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	settingsController controllers.SettingsController,
) SettingsRoutes {
	return SettingsRoutes{
		handler:            handler,
		logger:             logger,
		settingsController: settingsController,
	}
}
