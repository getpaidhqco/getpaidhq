package services

import (
	"context"
	"encoding/json"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/values"
	"payloop/internal/infrastructure/db/postgres"
	"time"
)

type ReportService struct {
	logger           logger.Logger
	reportRepository repositories.ReportRepository
	pubsub           events.PubSub
	queueClient      events.QueueClient
	orgRepository    repositories.OrgRepository
	cdcStream        postgres.CdcStream
}

func NewReportService(
	logger logger.Logger,
	reportRepository repositories.ReportRepository,
	pubsub events.PubSub,
	queueClient events.QueueClient,
	cdcStream postgres.CdcStream,
	scheduler interfaces.Scheduler,
	orgRepository repositories.OrgRepository,
) interfaces.ReportService {
	service := ReportService{
		logger:           logger,
		reportRepository: reportRepository,
		pubsub:           pubsub,
		queueClient:      queueClient,
		cdcStream:        cdcStream,
		orgRepository:    orgRepository,
	}

	// set up the payment method expiry detection
	// 3am first of every month
	err := scheduler.ScheduleTask("0 1 * * *", service.StoreDailyMetrics)
	if err != nil {
		logger.Errorf("Failed to schedule task: %v", err)
		panic(err)
	}

	cdcStream.Start(context.Background(), service.MapCdcStream)

	_, err = pubsub.Subscribe(">", service.HandlePublishedEvent)
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}
	return service
}

func (s ReportService) GetMonthlyRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	s.logger.Debugf("Getting MRR for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetMRR(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s ReportService) GetAnnualRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	s.logger.Debugf("Getting MRR for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetMRR(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s ReportService) GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	s.logger.Debugf("Getting active subs for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetActiveSubscribers(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s ReportService) GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	s.logger.Debugf("Getting active subs for org %s between %s and %s", orgId, startDate, endDate)

	mrr, err := s.reportRepository.GetRefundTotals(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

func (s ReportService) GetCustomerChurnTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	mrr, err := s.reportRepository.GetCustomerChurnTotals(ctx, orgId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return mrr, nil
}

// HandlePublishedEvent handles the published event from the pubsub and processes it
// either via the queue or directly.  Which one to use depends on the deployment, simple
// deployments will call the service directly, while more complex deployments will use the queue.
// it will eventually end up in the reporting service HandleDataChangeEvent
func (s ReportService) HandlePublishedEvent(_ string, data []byte) {
	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	//err = s.queueClient.SendMessage(context.Background(), events.QueueMessage{
	//	Type: events.ReportingDataChange,
	//	Data: payload,
	//})
	//if err != nil {
	//	s.logger.Errorf("Failed to send message to queue: %v", err)
	//	return
	//}

	//s.ProcessDataChange(data)

}

func (s ReportService) MapCdcStream(op string, entity string, new interface{}, old interface{}) {

	event := dto.DataChangeEvent{
		Operation: common.Operation(op),
		Entity:    common.Entity(entity),
		NewObject: new,
		OldObject: old,
	}
	s.ProcessDataChange(event)
}

func (s ReportService) ProcessDataChange(event dto.DataChangeEvent) {
	s.logger.Debugf("ProcessDataChange: %s->%s", event.Operation, event.Entity)
	switch event.Entity {
	case "subscriptions":
		var subscription entities.Subscription
		payloadBytes, err := json.Marshal(event.NewObject)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &subscription)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal subscription: %v", err)
			return
		}

		err = s.reportRepository.UpsertSubscription(context.Background(), subscription)
		if err != nil {
			s.logger.Errorf("Failed to upsert subscription: %v", err)
			return
		}
	case "payments":
		var payment entities.Payment
		payloadBytes, err := json.Marshal(event.NewObject)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &payment)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal payment: %v", err)
			return
		}

		err = s.reportRepository.UpsertPayment(context.Background(), payment)
		if err != nil {
			s.logger.Errorf("Failed to upsert payment: %v", err)
			return
		}
	case common.CustomerEntity:
		var customer entities.Customer
		payloadBytes, err := json.Marshal(event.NewObject)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &customer)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal customer: %v", err)
			return
		}

		err = s.reportRepository.UpsertCustomer(context.Background(), customer)
		if err != nil {
			s.logger.Errorf("Failed to upsert customer: %v", err)
			return
		}

	case common.CustomerCohortEntity:
		var customerCohort entities.CustomerCohort
		payloadBytes, err := json.Marshal(event.NewObject)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &customerCohort)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal customer: %v", err)
			return
		}

		err = s.reportRepository.UpsertCustomerCohort(context.Background(), customerCohort)
		if err != nil {
			s.logger.Errorf("Failed to upsert customer: %v", err)
			return
		}
	}
}

func (s ReportService) StoreDailyMetrics() {
	s.logger.Debugf("Storing daily metrics")
	// get the date for today
	yesterday := time.Now().AddDate(0, 0, -1)

	err := s.reportRepository.ProcessDailyMetrics(context.Background(), yesterday)
	if err != nil {
		s.logger.Errorf("Failed to store daily metrics: %v", err)
		return
	}
	s.logger.Infof("Stored daily metrics for %s", yesterday.Format("2006-01-02"))
}
