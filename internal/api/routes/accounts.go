package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type OrgsRoutes struct {
	logger     lib.Logger
	handler    lib.RequestHandler
	controller controllers.OrgController
}

// Setup user routes
func (s OrgsRoutes) Setup() {
	s.logger.Info("Setting up Tenants routes")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/tenants", s.controller.Create)
	}
}

// NewOrgsRoutes creates new user controller
func NewOrgsRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	controller controllers.OrgController,
) OrgsRoutes {
	return OrgsRoutes{
		handler:    handler,
		logger:     logger,
		controller: controller,
	}
}
