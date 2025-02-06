package services

import (
	"context"
	"payloop/internal/application/lib/events"
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
	orderItemRepository    repositories.OrderItemRepository
	paymentGateway         payment_providers.Gateway
	pubsub                 events.PubSub
	logger                 lib.Logger
}

func NewSubscriptionService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	orderItemRepository repositories.OrderItemRepository,
	customerRepository repositories.CustomerRepository,
	orderRepository repositories.OrderRepository,
	paymentRepository repositories.PaymentRepository,
	pubsub events.PubSub,
	paymentGateway payment_providers.Gateway,
	logger lib.Logger,
) SubscriptionService {
	return SubscriptionService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		paymentRepository:      paymentRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		orderItemRepository:    orderItemRepository,
		subscriptionRepository: subscriptionRepository,
		pubsub:                 pubsub,
		logger:                 logger,
		paymentGateway:         paymentGateway,
	}
}

func (s *SubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	s.logger.Info("CreateSubscriptionsForOrder", "orgId", orgId, "orderId", orderId)
	var subscriptions []entities.Subscription
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order", err.Error())
		return subscriptions, err
	}

	orderItems, err := s.orderItemRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order items", err.Error())
		return subscriptions, err
	}

	for _, item := range orderItems {
		subscription := entities.NewSubscriptionFromOrderItem(item)
		if order.Status == entities.OrderStatusCompleted {
			subscription.Status = entities.SubscriptionStatusActive
		}

		_, err := s.subscriptionRepository.Create(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to create subscription", "item", item, err.Error())
			return subscriptions, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	s.logger.Info("Subscriptions created", "count", len(subscriptions))
	return subscriptions, nil
}

func (s *SubscriptionService) Create(ctx context.Context, input subscriptions.CreateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Creating new subscription", "orgId", input.OrgId)

	subscription := subscriptions.NewFromCreateInput(input)
	subscription, err := s.subscriptionRepository.Create(ctx, subscription)

	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.PublishJSON(events.TopicSubscriptionCreated, subscription)

	return subscription, nil
}

func (s *SubscriptionService) Update(ctx context.Context, input subscriptions.UpdateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Marking subscription active", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s *SubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Info("Fetching", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
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

func (s *SubscriptionService) Pause(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Info("Pausing subscriptino", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	subscription.Status = entities.SubscriptionStatusPaused
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	// publi
	_ = s.pubsub.PublishJSON(events.TopicSubscriptionPaused, subscription)

	return subscription, nil
}

func (s *SubscriptionService) StoreSubscriptionPayment(ctx context.Context, input subscriptions.StoreSubscriptionPaymentInput) (entities.Subscription, error) {
	s.logger.Info("Recording subscription payment and updating subscription")
	subscription := input.Subscription
	charge := input.ChargeResult

	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
	}

	matadata := make(map[string]string)
	matadata["psp_id"] = charge.PspId

	payment := entities.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		PspId:          charge.PspId,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,

		Status:      charge.Status,
		Currency:    charge.Currency,
		Amount:      charge.Amount,
		PspFee:      0,
		PlatformFee: 0,
		NetAmount:   subscription.Amount,
		Metadata:    matadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	_, err := s.paymentRepository.Create(ctx, payment)
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
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	return newSub, nil
}
