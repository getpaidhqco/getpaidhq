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

	cdcStream postgres.CdcStream
}

func NewReportService(
	logger logger.Logger,
	reportRepository repositories.ReportRepository,
	pubsub events.PubSub,
	queueClient events.QueueClient,
	cdcStream postgres.CdcStream,
) interfaces.ReportService {
	service := ReportService{
		logger:           logger,
		reportRepository: reportRepository,
		pubsub:           pubsub,
		queueClient:      queueClient,
	}

	cdcStream.Start(context.Background(), service.MapCdcStream)

	_, err := pubsub.Subscribe(">", service.HandlePublishedEvent)
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

	mrr, err := s.reportRepository.GetMRR(ctx, orgId, startDate, endDate)
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
			s.logger.Errorf("Failed to unmarshal subscription: %v", err)
			return
		}

		err = s.reportRepository.UpsertPayment(context.Background(), payment)
		if err != nil {
			s.logger.Errorf("Failed to upsert subscription: %v", err)
			return
		}
	}
}
