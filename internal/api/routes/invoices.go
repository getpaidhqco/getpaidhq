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

type InvoiceRoutes struct {
	logger            logger.Logger
	handler           lib.RequestHandler
	invoiceController controllers.InvoiceController
	authz             authz.Authz
}

// Setup invoice routes
func (s InvoiceRoutes) Setup() {
	s.logger.Info("Setting up Invoice routes")
	api := s.handler.Gin.Group("/api")
	{
		// Invoice CRUD operations
		api.GET("/invoices", s.invoiceController.List)
		api.GET("/invoices/:id", s.invoiceController.Get)
		api.POST("/invoices", s.checkAuthz(authz.CreateInvoice), s.invoiceController.Create)
		api.PUT("/invoices/:id", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.Update)
		api.POST("/invoices/:id/actions", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.PerformAction)

		// Invoice line items
		api.GET("/invoices/:id/line-items", s.invoiceController.ListLineItems)
		api.POST("/invoices/:id/line-items", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.AddLineItem)
		api.PUT("/invoices/:id/line-items/:lineItemId", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.UpdateLineItem)
		api.DELETE("/invoices/:id/line-items/:lineItemId", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.DeleteLineItem)

		// Invoice history
		api.GET("/invoices/:id/history", s.invoiceController.ListHistory)

		// Invoice PDF generation
		api.POST("/invoices/:id/pdf", s.checkAuthz(authz.UpdateInvoice), s.invoiceController.GeneratePDF)

		// Customer invoices
		api.GET("/customers/:id/invoices", s.invoiceController.ListByCustomer)
	}
}

func (s InvoiceRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
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

func NewInvoiceRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	authz authz.Authz,
	invoiceController controllers.InvoiceController,
) InvoiceRoutes {
	return InvoiceRoutes{
		handler:           handler,
		logger:            logger,
		authz:             authz,
		invoiceController: invoiceController,
	}
}
