package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// ReportHandler handles HTTP requests for reports.
type ReportHandler struct {
	reportService *service.ReportService
	logger        port.Logger
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(reportService *service.ReportService, logger port.Logger) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		logger:        logger,
	}
}

// RegisterRoutes registers report routes on the given router group.
func (s *ReportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/reports/revenue/mrr", s.GetMRR)
	rg.GET("/reports/revenue/arr", s.GetARR)
	rg.GET("/reports/active-subscribers", s.GetSubscribers)
	rg.GET("/reports/refunds", s.GetRefundTotals)
	rg.GET("/reports/churn/totals", s.GetCustomerChurnTotals)
	rg.GET("/reports/churn/rates", s.GetCustomerChurnRates)
}

func (s *ReportHandler) GetMRR(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)

	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	mrr, err := s.reportService.GetMonthlyRecurringRevenue(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, mrr)
}

func (s *ReportHandler) GetARR(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetAnnualRecurringRevenue(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}

func (s *ReportHandler) GetSubscribers(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetActiveSubscribers(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}

func (s *ReportHandler) GetRefundTotals(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetRefundTotals(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}

func (s *ReportHandler) GetCustomerChurnTotals(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetCustomerChurnTotals(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}

func (s *ReportHandler) GetCustomerChurnRates(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	startTime, err := time.Parse(time.DateOnly, c.Query("start_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	endTime, err := time.Parse(time.DateOnly, c.Query("end_date"))
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)

	arr, err := s.reportService.GetCustomerChurnRates(c.Request.Context(), authUser.OrgId, startTime, endTime)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, arr)
}
