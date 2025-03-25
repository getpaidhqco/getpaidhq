package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type ReportRoutes struct {
	logger           logger.Logger
	handler          lib.RequestHandler
	reportController controllers.ReportController
}

// Setup user routes
func (s ReportRoutes) Setup() {
	s.logger.Info("Setting up Report")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/reports/revenue/mrr", s.reportController.GetMRR)
		api.GET("/reports/revenue/arr", s.reportController.GetARR)
	}
}

// NewReportRoutes creates new user controller
func NewReportRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	reportController controllers.ReportController,
) ReportRoutes {
	return ReportRoutes{
		handler:          handler,
		logger:           logger,
		reportController: reportController,
	}
}
