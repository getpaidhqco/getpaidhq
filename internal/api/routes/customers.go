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

type CustomerRoutes struct {
	logger             logger.Logger
	handler            lib.RequestHandler
	customerController controllers.CustomerController
	authz              authz.Authz
}

// Setup user routes
func (s CustomerRoutes) Setup() {
	s.logger.Info("Setting up Customer routes")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/customers", s.customerController.List)
		api.GET("/customers/:id", s.customerController.Get)
		api.POST("/customers", s.customerController.Create)
		api.POST("/customers/:id/payment-methods", s.checkAuthz(authz.CreatePaymentMethod), s.customerController.CreateCustomerPaymentMethod)
		api.PUT("/customers/:id/payment-methods/:pmid", s.checkAuthz(authz.CreatePaymentMethod), s.customerController.UpdateCustomerPaymentMethod)
	}
}

func (s CustomerRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
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

func NewCustomerRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	authz authz.Authz,
	customerController controllers.CustomerController,
) CustomerRoutes {
	return CustomerRoutes{
		handler:            handler,
		logger:             logger,
		authz:              authz,
		customerController: customerController,
	}
}
