package services

import (
	"context"
	"encoding/json"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/interfaces/webhooks"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type QueueService struct {
	logger      logger.Logger
	queueClient events.QueueClient

	webhookService webhooks.WebhookService
}

func NewQueueService(
	logger logger.Logger,
	queueClient events.QueueClient,
	webhookService webhooks.WebhookService,
) interfaces.QueueService {
	service := QueueService{
		logger:         logger,
		queueClient:    queueClient,
		webhookService: webhookService,
	}

	queueClient.Start(service.HandleQueueMessage)
	return service
}

func (s QueueService) HandleQueueMessage(data events.QueueMessage) error {
	s.logger.Infof("[QueueService] queue message: [%s]", data.Type)

	switch data.Type {
	case events.IncomingWebhook:
		var payload webhooks.PaymentWebhookPayload
		payloadBytes, err := json.Marshal(data.Data)
		if err != nil {
			s.logger.Errorf("[QueueService] failed to marshal data: %v", err)
			return err
		}

		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			s.logger.Errorf("[QueueService] failed to unmarshal webhook payload: %v", err)
			return err
		}

		return s.webhookService.HandlePaymentWebhook(context.Background(), payload)
	default:
		s.logger.Errorf("[QueueService] unknown message type: [%s]", data.Type)
		return lib.NewCustomError(lib.BadRequestError, "unknown message type", nil)
	}
}
