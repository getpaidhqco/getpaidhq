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

type DiscountRoutes struct {
	logger             logger.Logger
	handler            lib.RequestHandler
	discountController controllers.DiscountController
	authz              authz.Authz
}

// Setup discount routes
func (d DiscountRoutes) Setup() {
	d.logger.Info("Setting up Discount routes")
	api := d.handler.Gin.Group("/api")
	{
		// Discount CRUD routes only
		api.GET("/discounts", d.checkAuthz(authz.ListDiscounts), d.discountController.ListDiscounts)
		api.POST("/discounts", d.checkAuthz(authz.CreateDiscount), d.discountController.CreateDiscount)
		api.GET("/discounts/:id", d.checkAuthz(authz.GetDiscount), d.discountController.GetDiscount)
		api.PUT("/discounts/:id", d.checkAuthz(authz.UpdateDiscount), d.discountController.UpdateDiscount)
		api.DELETE("/discounts/:id", d.checkAuthz(authz.DeleteDiscount), d.discountController.DeleteDiscount)
	}
}

func (d DiscountRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(authn.User)
		allowed := d.authz.Enforce(authUser, action, "")
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

// NewDiscountRoutes creates new discount routes
func NewDiscountRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	discountController controllers.DiscountController,
	authz authz.Authz,
) DiscountRoutes {
	return DiscountRoutes{
		handler:            handler,
		logger:             logger,
		authz:              authz,
		discountController: discountController,
	}
}
