package service

import (
	"context"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
)

// PaymentService owns payment lifecycle operations triggered by the workflow
// engine (refunds today; other PSP-driven payment events later). Keeping this
// off SubscriptionService lets each engine adapter call a single thin service
// instead of reimplementing the body of each activity.
type PaymentService struct {
	paymentRepository port.PaymentRepository
	logger            port.Logger
}

func NewPaymentService(paymentRepository port.PaymentRepository, logger port.Logger) *PaymentService {
	return &PaymentService{
		paymentRepository: paymentRepository,
		logger:            logger,
	}
}

// GetById returns one payment.
func (s *PaymentService) GetById(ctx context.Context, orgId, id string) (domain.Payment, error) {
	return s.paymentRepository.FindById(ctx, orgId, id)
}

// List returns the org's payments, newest first.
func (s *PaymentService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Payment, int, error) {
	return s.paymentRepository.List(ctx, orgId, p)
}

// ListBySubscription returns a subscription's payments.
func (s *PaymentService) ListBySubscription(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.Payment, int, error) {
	return s.paymentRepository.FindBySubscriptionId(ctx, orgId, subscriptionId, p)
}

// ProcessRefund flips the matching payment to refunded and records a refund row.
func (s *PaymentService) ProcessRefund(ctx context.Context, paymentContext domain.PaymentWebhookContext) (domain.Payment, error) {
	s.logger.Info("ProcessRefund", "orgId", paymentContext.OrgId, "orderId", paymentContext.OrderId)

	payment, err := s.paymentRepository.FindByPspId(ctx, paymentContext.OrgId, paymentContext.Payment.PspId)
	if err != nil {
		s.logger.Error("error finding payment", "orgId", paymentContext.OrgId, "pspId", paymentContext.Payment.PspId, "err", err.Error())
		return domain.Payment{}, err
	}

	payment.Status = domain.PaymentStatusRefunded
	newPayment, err := s.paymentRepository.Update(ctx, payment)
	if err != nil {
		s.logger.Error("error updating payment", "orgId", paymentContext.OrgId, "paymentId", payment.Id, "err", err.Error())
		return domain.Payment{}, err
	}

	now := time.Now().UTC()
	_, err = s.paymentRepository.CreateRefund(ctx, domain.Refund{
		OrgId:      paymentContext.OrgId,
		Id:         lib.GenerateId("refund"),
		PaymentId:  payment.Id,
		Amount:     paymentContext.Payment.Amount,
		Currency:   paymentContext.Payment.Currency,
		RefundedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		s.logger.Error("error creating refund", "orgId", paymentContext.OrgId, "paymentId", payment.Id, "err", err.Error())
		return domain.Payment{}, err
	}

	return newPayment, nil
}
