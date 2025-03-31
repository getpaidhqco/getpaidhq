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

type PspRoutes struct {
	logger        logger.Logger
	handler       lib.RequestHandler
	pspController controllers.PspController
	authz         authz.Authz
}

// Setup user routes
func (s PspRoutes) Setup() {
	s.logger.Info("Setting up Psp")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/gateways", s.checkAuthz(authz.CreatePaymentServiceProvider), s.pspController.Create)
	}
}

func (s PspRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
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

// NewPspRoutes creates new user controller
func NewPspRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	pspController controllers.PspController,
	authz authz.Authz,
) PspRoutes {
	return PspRoutes{
		handler:       handler,
		logger:        logger,
		authz:         authz,
		pspController: pspController,
	}
}
