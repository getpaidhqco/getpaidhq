package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type AccountsRoutes struct {
	logger     lib.Logger
	handler    lib.RequestHandler
	controller controllers.AccountController
}

// Setup user routes
func (s AccountsRoutes) Setup() {
	s.logger.Info("Setting up Tenants routes")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/tenants", s.controller.Create)
	}
}

// NewAccountsRoutes creates new user controller
func NewAccountsRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	controller controllers.AccountController,
) AccountsRoutes {
	return AccountsRoutes{
		handler:    handler,
		logger:     logger,
		controller: controller,
	}
}
