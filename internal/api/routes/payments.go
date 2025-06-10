package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type PaymentRoutes struct {
	logger            logger.Logger
	handler           lib.RequestHandler
	paymentController controllers.PaymentController
	authz             authz.Authz
}

// Setup payment routes
func (s PaymentRoutes) Setup() {
	s.logger.Info("Setting up Payment routes")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/payments", s.paymentController.List)
		api.GET("/payments/:id", s.paymentController.Get)
		api.POST("/payments/:id/refund", s.paymentController.Refund)
	}
}

// NewPaymentRoutes creates new payment routes
func NewPaymentRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	paymentController controllers.PaymentController,
	authz authz.Authz,
) PaymentRoutes {
	return PaymentRoutes{
		handler:           handler,
		logger:            logger,
		paymentController: paymentController,
		authz:             authz,
	}
}