package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type HealthRoutes struct {
	logger           logger.Logger
	handler          lib.RequestHandler
	healthController controllers.HealthController
}

// Setup user routes
func (s HealthRoutes) Setup() {
	s.logger.Info("Setting up Health")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/health", s.healthController.Healthcheck)
	}
}

func NewHealthRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	healthController controllers.HealthController,
) HealthRoutes {
	return HealthRoutes{
		handler:          handler,
		logger:           logger,
		healthController: healthController,
	}
}
