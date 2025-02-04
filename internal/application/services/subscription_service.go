package services

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type SubscriptionService struct {
	sessionRepository      repositories.SessionRepository
	cartRepository         repositories.CartRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	paymentRepository      repositories.PaymentRepository
	paymentGateway         payment_providers.Gateway
	logger                 lib.Logger
}

func NewSubscriptionService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	customerRepository repositories.CustomerRepository,
	orderRepository repositories.OrderRepository,
	paymentRepository repositories.PaymentRepository,
	paymentGateway payment_providers.Gateway,
	logger lib.Logger,
) SubscriptionService {
	return SubscriptionService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
		paymentGateway:         paymentGateway,
	}
}

func (s *SubscriptionService) Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Info("Marking subscription active", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	subscription.Status = entities.SubscriptionStatusActive
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s *SubscriptionService) StoreSubscriptionPayment(ctx context.Context, input subscriptions.StoreSubscriptionPaymentInput) (entities.Subscription, error) {
	s.logger.Info("Recording subscription payment and updating subscription")
	subscription := input.Subscription
	charge := input.ChargeResult

	matadata := make(map[string]string)
	matadata["psp_id"] = charge.PspId

	payment := entities.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Status:         charge.Status,
		Currency:       charge.Currency,
		Amount:         charge.Amount,
		PspFee:         0,
		PlatformFee:    0,
		NetAmount:      subscription.Amount,
		Metadata:       matadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payment, err := s.paymentRepository.Create(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to create payment", err.Error())
	}

	// update the subscription status
	lastCharge := time.Now().UTC()
	subscription.Status = entities.SubscriptionStatusActive
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = &lastCharge

	nextCharge := subscription.NextBillingDate()
	subscription.RenewsAt = &nextCharge

	s.logger.Info("Subscription charged, updating with new values",
		"id", subscription.Id,
		"NextCharge", nextCharge,
		"cycles", subscription.CyclesProcessed,
		"totalRevenue", subscription.TotalRevenue)
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)

	return newSub, nil
}
