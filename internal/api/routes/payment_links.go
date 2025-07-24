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

type PaymentLinkRoutes struct {
	logger                 logger.Logger
	handler                lib.RequestHandler
	paymentLinkController  controllers.PaymentLinkController
	authz                  authz.Authz
}

// Setup payment link routes
func (p PaymentLinkRoutes) Setup() {
	p.logger.Info("Setting up Payment Link routes")
	api := p.handler.Gin.Group("/api")
	{
		// Payment Link routes
		api.GET("/payment-links", p.checkAuthz(authz.ListPaymentLinks), p.paymentLinkController.ListPaymentLinks)
		api.POST("/payment-links", p.checkAuthz(authz.CreatePaymentLink), p.paymentLinkController.CreatePaymentLink)

		// Payment Link Usage routes
		api.POST("/payment-links/usage", p.checkAuthz(authz.RecordPaymentLinkUsage), p.paymentLinkController.RecordPaymentLinkUsage)
		api.GET("/payment-links/usage/:id", p.checkAuthz(authz.GetPaymentLinkUsage), p.paymentLinkController.GetPaymentLinkUsage)

		// Routes with slug parameter
		api.GET("/payment-links/slug/:slug", p.checkAuthz(authz.GetPaymentLink), p.paymentLinkController.GetPaymentLinkBySlug)

		// Routes with ID parameter - specific endpoints first
		api.GET("/payment-links/:id/usage", p.checkAuthz(authz.ListPaymentLinkUsages), p.paymentLinkController.ListPaymentLinkUsages)
		api.GET("/payment-links/:id", p.checkAuthz(authz.GetPaymentLink), p.paymentLinkController.GetPaymentLink)
		api.PUT("/payment-links/:id", p.checkAuthz(authz.UpdatePaymentLink), p.paymentLinkController.UpdatePaymentLink)
		api.DELETE("/payment-links/:id", p.checkAuthz(authz.DeletePaymentLink), p.paymentLinkController.DeletePaymentLink)
	}
}

func (p PaymentLinkRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(authn.User)
		allowed := p.authz.Enforce(authUser, action, "")
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

// NewPaymentLinkRoutes creates new payment link routes
func NewPaymentLinkRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	paymentLinkController controllers.PaymentLinkController,
	authz authz.Authz,
) PaymentLinkRoutes {
	return PaymentLinkRoutes{
		handler:                handler,
		logger:                 logger,
		authz:                  authz,
		paymentLinkController:  paymentLinkController,
	}
}
