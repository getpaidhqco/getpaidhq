package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type PublicPaymentRoutes struct {
	logger                  logger.Logger
	handler                 lib.RequestHandler
	publicPaymentController controllers.PublicPaymentController
}

// Setup payment routes
func (r PublicPaymentRoutes) Setup() {
	r.logger.Info("Setting up public payment routes")
	pay := r.handler.Gin.Group("/api/pay")
	{
		// Public payment endpoints - no authentication middleware
		pay.GET("/:slug", r.publicPaymentController.GetPaymentDetails)
		pay.POST("/:slug/create-order", r.publicPaymentController.CreateOrder)
		pay.GET("/:slug/order/:orderId/status", r.publicPaymentController.GetOrderStatus)
	}
}

// NewPublicPaymentRoutes creates new public payment routes
func NewPublicPaymentRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	publicPaymentController controllers.PublicPaymentController,
) PublicPaymentRoutes {
	return PublicPaymentRoutes{
		handler:                 handler,
		logger:                  logger,
		publicPaymentController: publicPaymentController,
	}
}