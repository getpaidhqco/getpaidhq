package services

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type PaymentService struct {
	paymentRepository repositories.PaymentRepository
	gatewayFactory    factories.GatewayFactory
	pubsub            events.PubSub
	logger            logger.Logger
}

func NewPaymentService(
	paymentRepository repositories.PaymentRepository,
	pubsub events.PubSub,
	gatewayFactory factories.GatewayFactory,
	logger logger.Logger,
) interfaces.PaymentService {
	return PaymentService{
		paymentRepository: paymentRepository,
		pubsub:            pubsub,
		gatewayFactory:    gatewayFactory,
		logger:            logger,
	}
}

func (s PaymentService) FindById(ctx context.Context, orgId string, id string) (entities.Payment, error) {
	s.logger.Info("Fetching payment", "orgId", orgId, "id", id)

	payment, err := s.paymentRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find payment", err.Error())
		return entities.Payment{}, err
	}

	return payment, nil
}

func (s PaymentService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Payment, int, error) {
	s.logger.Info("Listing payments", "orgId", orgId)

	// Convert the request pagination to domain pagination
	domainPagination := entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortBy:        pagination.SortBy,
		SortDirection: pagination.SortDirection,
	}

	// Call the repository to get the payments
	payments, total, err := s.paymentRepository.List(ctx, orgId, domainPagination)
	if err != nil {
		s.logger.Error("Failed to list payments", err.Error())
		return nil, 0, err
	}

	return payments, total, nil
}

// Refund processes a refund for a payment. It does the validation and creates a refund request.
// The rest of the processing is handled by the webhook handler and RefundProcessed workflow.
func (s PaymentService) Refund(ctx context.Context, orgId string, paymentId string, input request.RefundPaymentRequest) (entities.Refund, error) {
	s.logger.Info("Refunding payment", "orgId", orgId, "paymentId", paymentId)

	// Find the payment
	payment, err := s.paymentRepository.FindById(ctx, orgId, paymentId)
	if err != nil {
		s.logger.Error("Failed to find payment", err.Error())
		return entities.Refund{}, err
	}

	// Check if the payment can be refunded
	if payment.Status != payments.PaymentStatusSucceeded &&
		payment.Status != payments.PaymentStatusPartialRefund {
		s.logger.Error("Payment cannot be refunded", "status", payment.Status)
		return entities.Refund{}, lib.NewCustomError(lib.BadRequestError, "Payment cannot be refunded", nil)
	}

	// Create the refund
	refund := entities.Refund{
		OrgId:     orgId,
		Id:        lib.GenerateId("ref"),
		PaymentId: paymentId,
		Amount:    input.Amount,
		Currency:  payment.Currency,
		Reason:    input.Reason,
		Status:    entities.RefundStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	gw, err := s.gatewayFactory.NewGateway(ctx, orgId, payment.PspId)
	if err != nil {
		s.logger.Error("Failed to create gateway", err.Error())
		return entities.Refund{}, err
	}
	// Process the refund through the payment gateway
	rsp, err := gw.RefundPayment(ctx, payment_providers.RefundPaymentCommand{
		PaymentId: payment.PspId,
		Currency:  common.Currency(payment.Currency),
		Amount:    refund.Amount,
		Reason:    refund.Reason,
	})
	if err != nil {
		s.logger.Error("Failed to process refund through gateway", err.Error())
		return entities.Refund{}, err
	}
	s.logger.Info("Refund processed through gateway", "rsp", rsp)

	// Save the refund
	refund, err = s.paymentRepository.CreateRefund(ctx, refund)
	if err != nil {
		s.logger.Error("Failed to create refund", err.Error())
		return entities.Refund{}, err
	}

	// Publish events
	_ = s.pubsub.Publish(orgId, topic.RefundCreated, refund)

	return refund, nil
}
