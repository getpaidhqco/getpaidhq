package service

import (
	"context"
	"encoding/json"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type QueueService struct {
	logger         port.Logger
	queueClient    port.QueueClient
	reportService  *ReportService
	webhookService *WebhookService
}

func NewQueueService(
	logger port.Logger,
	queueClient port.QueueClient,
	webhookService *WebhookService,
	reportService *ReportService,
) *QueueService {
	service := &QueueService{
		logger:         logger,
		queueClient:    queueClient,
		webhookService: webhookService,
		reportService:  reportService,
	}

	queueClient.Start(service.HandleQueueMessage)
	return service
}

func (s *QueueService) HandleQueueMessage(data port.QueueMessage) error {
	s.logger.Info("queue message received", "type", data.Type)

	payloadBytes, err := json.Marshal(data.Data)
	if err != nil {
		s.logger.Error("failed to marshal data", "error", err)
		return err
	}

	switch data.Type {
	case port.QueueIncomingWebhook:
		var payload port.PaymentWebhookPayload
		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			s.logger.Error("failed to unmarshal webhook payload", "error", err)
			return err
		}

		return s.webhookService.HandlePaymentWebhook(context.Background(), payload)
	case port.QueueReportingDataChange:
		var payload port.PubSubPayload
		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			s.logger.Error("failed to unmarshal webhook payload", "error", err)
			return err
		}

		s.logger.Debug("received report event", "topic", payload.Topic)
		//s.reportService.ProcessDataChange(payloadBytes)
		return nil
	default:
		s.logger.Error("unknown message type", "type", data.Type)
		return lib.NewCustomError(lib.BadRequestError, "unknown message type", nil)
	}
}
