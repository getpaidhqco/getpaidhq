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

type PaymentMethodRoutes struct {
	logger             logger.Logger
	handler            lib.RequestHandler
	customerController controllers.CustomerController
	authz              authz.Authz
}

// Setup user routes
func (s PaymentMethodRoutes) Setup() {
	s.logger.Info("Setting up PaymentMethod routes")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/payment-methods/:id", s.customerController.GetCustomerPaymentMethod)
	}
}

func (s PaymentMethodRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
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

func NewPaymentMethodRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	authz authz.Authz,
	customerController controllers.CustomerController,
) PaymentMethodRoutes {
	return PaymentMethodRoutes{
		handler:            handler,
		logger:             logger,
		authz:              authz,
		customerController: customerController,
	}
}
