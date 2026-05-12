package service

import (
	"context"
	"encoding/json"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
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
	s.logger.Infof("[QueueService] queue message: [%s]", data.Type)

	payloadBytes, err := json.Marshal(data.Data)
	if err != nil {
		s.logger.Errorf("[QueueService] failed to marshal data: %v", err)
		return err
	}

	switch data.Type {
	case port.QueueIncomingWebhook:
		var payload port.PaymentWebhookPayload
		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			s.logger.Errorf("[QueueService] failed to unmarshal webhook payload: %v", err)
			return err
		}

		return s.webhookService.HandlePaymentWebhook(context.Background(), payload)
	case port.QueueReportingDataChange:
		var payload port.PubSubPayload
		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			s.logger.Errorf("[QueueService] failed to unmarshal webhook payload: %v", err)
			return err
		}

		s.logger.Debugf("[QueueService] received report event: [%s]", payload.Topic)
		//s.reportService.ProcessDataChange(payloadBytes)
		return nil
	default:
		s.logger.Errorf("[QueueService] unknown message type: [%s]", data.Type)
		return lib.NewCustomError(lib.BadRequestError, "unknown message type", nil)
	}
}
