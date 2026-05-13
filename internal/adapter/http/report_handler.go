package handler

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type ReportHandler struct {
	reportService *service.ReportService
	logger        port.Logger
}

func NewReportHandler(reportService *service.ReportService, logger port.Logger) *ReportHandler {
	return &ReportHandler{reportService: reportService, logger: logger}
}

func (s *ReportHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/reports", option.Tags("Reports"))
	dateRange := option.Group(
		option.Query("start_date", "Inclusive start date (YYYY-MM-DD)"),
		option.Query("end_date", "Inclusive end date (YYYY-MM-DD)"),
	)
	fuego.Get(g, "/revenue/mrr", s.GetMRR, option.Summary("Monthly recurring revenue"), dateRange)
	fuego.Get(g, "/revenue/arr", s.GetARR, option.Summary("Annual recurring revenue"), dateRange)
	fuego.Get(g, "/active-subscribers", s.GetSubscribers, option.Summary("Active subscribers"), dateRange)
	fuego.Get(g, "/refunds", s.GetRefundTotals, option.Summary("Refund totals"), dateRange)
	fuego.Get(g, "/churn/totals", s.GetCustomerChurnTotals, option.Summary("Customer churn totals"), dateRange)
	fuego.Get(g, "/churn/rates", s.GetCustomerChurnRates, option.Summary("Customer churn rates"), dateRange)
}

func parseDateRange[B, P any](c fuego.Context[B, P]) (time.Time, time.Time, error) {
	startTime, err := time.Parse(time.DateOnly, c.QueryParam("start_date"))
	if err != nil {
		return time.Time{}, time.Time{}, NewApiError(lib.BadRequestError, "Invalid start_date (expected YYYY-MM-DD)", err.Error())
	}
	endTime, err := time.Parse(time.DateOnly, c.QueryParam("end_date"))
	if err != nil {
		return time.Time{}, time.Time{}, NewApiError(lib.BadRequestError, "Invalid end_date (expected YYYY-MM-DD)", err.Error())
	}
	startTime = startTime.Truncate(24 * time.Hour)
	endTime = endTime.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Nanosecond)
	return startTime, endTime, nil
}

func (s *ReportHandler) GetMRR(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetMonthlyRecurringRevenue(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}

func (s *ReportHandler) GetARR(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetAnnualRecurringRevenue(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}

func (s *ReportHandler) GetSubscribers(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetActiveSubscribers(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}

func (s *ReportHandler) GetRefundTotals(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetRefundTotals(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}

func (s *ReportHandler) GetCustomerChurnTotals(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetCustomerChurnTotals(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}

func (s *ReportHandler) GetCustomerChurnRates(c fuego.ContextNoBody) (any, error) {
	authUser := AuthUserFrom(c)
	start, end, err := parseDateRange(c)
	if err != nil {
		return nil, err
	}
	out, err := s.reportService.GetCustomerChurnRates(c.Context(), authUser.OrgId, start, end)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return out, nil
}
