package service

import (
	"context"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"
)

type ReportService struct {
	logger           port.Logger
	reportRepository port.ReportRepository
	orgRepository    port.OrgRepository
}

func NewReportService(
	logger port.Logger,
	reportRepository port.ReportRepository,
	scheduler port.Scheduler,
	orgRepository port.OrgRepository,
) *ReportService {
	service := &ReportService{
		logger:           logger,
		reportRepository: reportRepository,
		orgRepository:    orgRepository,
	}

	err := scheduler.ScheduleTask("0 1 * * *", service.StoreDailyMetrics)
	if err != nil {
		logger.Errorf("Failed to schedule task: %v", err)
		panic(err)
	}

	return service
}

func (s *ReportService) GetMonthlyRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	s.logger.Debugf("Getting MRR for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetMRR(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) GetAnnualRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	s.logger.Debugf("Getting MRR for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetMRR(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	s.logger.Debugf("Getting active subs for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetActiveSubscribers(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	s.logger.Debugf("Getting active subs for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetRefundTotals(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) GetCustomerChurnTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	mrr, err := s.reportRepository.GetCustomerChurnTotals(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) GetCustomerChurnRates(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	mrr, err := s.reportRepository.GetCustomerChurnRates(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s *ReportService) StoreDailyMetrics() {
	s.logger.Debugf("Storing daily metrics")
	yesterday := time.Now().AddDate(0, 0, -1)

	err := s.reportRepository.ProcessDailyMetrics(context.Background(), yesterday)
	if err != nil {
		s.logger.Errorf("Failed to store daily metrics: %v", err)
		return
	}
	s.logger.Infof("Stored daily metrics for %s", yesterday.Format("2006-01-02"))
}
