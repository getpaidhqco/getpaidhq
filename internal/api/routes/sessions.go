package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type SessionRoutes struct {
	logger            logger.Logger
	handler           lib.RequestHandler
	sessionController controllers.SessionController
	authz             authz.Authz
}

// Setup user routes
func (s SessionRoutes) Setup() {
	s.logger.Info("Setting up Session")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/sessions", s.checkAuthz(authz.CreateSession), s.sessionController.Create)
	}
}

func (s SessionRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(authn.User)
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// NewSessionRoutes creates new user controller
func NewSessionRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	sessionController controllers.SessionController,
	authz authz.Authz,
) SessionRoutes {
	return SessionRoutes{
		handler:           handler,
		logger:            logger,
		authz:             authz,
		sessionController: sessionController,
	}
}
