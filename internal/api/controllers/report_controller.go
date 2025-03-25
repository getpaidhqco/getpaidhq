package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"time"
)

type ReportController struct {
	reportService interfaces.ReportService
	logger        logger.Logger
}

func NewReportController(
	reportService interfaces.ReportService,
	logger logger.Logger,
) ReportController {

	return ReportController{
		reportService: reportService,
		logger:        logger,
	}
}

func (s ReportController) GetMRR(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	startTime, err := time.Parse(time.RFC3339, c.Query("start_date"))
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.RFC3339, c.Query("end_date"))
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	mrr, err := s.reportService.GetMonthlyRecurringRevenue(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, mrr)
}

func (s ReportController) GetARR(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	startTime, err := time.Parse(time.RFC3339, c.Query("start_date"))
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.RFC3339, c.Query("end_date"))
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetAnnualRecurringRevenue(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}
