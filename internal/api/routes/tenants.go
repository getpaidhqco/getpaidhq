package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type TenantsRoutes struct {
	logger     lib.Logger
	handler    lib.RequestHandler
	controller controllers.TenantController
}

// Setup user routes
func (s TenantsRoutes) Setup() {
	s.logger.Info("Setting up Tenants routes")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/tenants", s.controller.Create)
	}
}

// NewTenantsRoutes creates new user controller
func NewTenantsRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	controller controllers.TenantController,
) TenantsRoutes {
	return TenantsRoutes{
		handler:    handler,
		logger:     logger,
		controller: controller,
	}
}
