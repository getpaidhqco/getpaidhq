package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type SessionRoutes struct {
	logger            lib.Logger
	handler           lib.RequestHandler
	sessionController controllers.SessionController
}

// Setup user routes
func (s SessionRoutes) Setup() {
	s.logger.Info("Setting up Session")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/sessions", s.sessionController.Create)
	}
}

// NewSessionRoutes creates new user controller
func NewSessionRoutes(
	logger lib.Logger,
	handler lib.RequestHandler,
	SessionController controllers.SessionController,
) SessionRoutes {
	return SessionRoutes{
		handler:           handler,
		logger:            logger,
		sessionController: SessionController,
	}
}
