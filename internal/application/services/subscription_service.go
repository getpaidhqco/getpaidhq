package services

import (
	"context"
	"errors"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
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
	workflowService        interfaces.WorkflowService
	gatewayFactory         factories.GatewayFactory
	pubsub                 events.PubSub
	logger                 logger.Logger
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
	gatewayFactory factories.GatewayFactory,
	logger logger.Logger,
) interfaces.SubscriptionService {

	_, err := pubsub.Subscribe("subscription.workflow.>", func(topic string, data []byte) {
		logger.Infof("Received message from %s", topic)
	})
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

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
		gatewayFactory:         gatewayFactory,
	}
}

func (s SubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	s.logger.Info("CreateSubscriptionsForOrder", "orgId", orgId, "orderId", orderId)
	var subs []entities.Subscription
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order", err.Error())
		return subs, err
	}

	orderItems, err := s.orderItemRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order items", err.Error())
		return subs, err
	}

	for _, item := range orderItems {
		subscription := entities.NewSubscriptionFromOrderItem(item)
		if order.Status == entities.OrderStatusCompleted {
			subscription.Status = entities.SubscriptionStatusActive
		}

		_, err := s.subscriptionRepository.Create(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to create subscription", "item", item, err.Error())
			return subs, err
		}
		subs = append(subs, subscription)
	}

	s.logger.Info("Subscriptions created", "count", len(subs))
	return subs, nil
}

func (s SubscriptionService) Create(ctx context.Context, input entities.CreateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Creating new subscription", "orgId", input.OrgId)

	subscription := entities.NewFromCreateInput(input)
	subscription, err := s.subscriptionRepository.Create(ctx, subscription)

	if err != nil {
		s.logger.Error("Failed create subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionCreated, subscription)

	return subscription, nil
}

func (s SubscriptionService) Update(ctx context.Context, input subscriptions.UpdateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Updating subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	if input.Status != subscription.Status {
		s.logger.Infof("Updating status %s", input.Status)
		subscription.Status = input.Status
	}

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, topic.GetSubscriptionTopic(subscription.Status), newSub)
	return newSub, err
}

func (s SubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Info("Fetching", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s SubscriptionService) Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
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

func (s SubscriptionService) PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Pausing subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status == entities.SubscriptionStatusPaused {
		s.logger.Info("Subscription is already paused")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is paused already", nil)
	}

	subscription.Status = entities.SubscriptionStatusPaused
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s SubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	subs, total, err := s.subscriptionRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list subscriptions", err.Error())
		return nil, 0, err
	}

	return subs, total, nil
}

func (s SubscriptionService) ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Resuming subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status != entities.SubscriptionStatusPaused &&
		subscription.Status != entities.SubscriptionStatusPastDue {
		s.logger.Info("Subscription is not paused")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is not paused", nil)
	}

	behaviour := subscriptions.ContinueExistingBillingPeriod
	if input.ResumeBehavior != "" {
		behaviour = input.ResumeBehavior
	}

	if behaviour == subscriptions.ContinueExistingBillingPeriod {
		nextCharge := subscription.CalculateNextBillingDate()
		if nextCharge.Before(time.Now().UTC()) {
			return entities.Subscription{}, lib.NewCustomError(lib.BadRequestError, "next billing date is in the past", errors.New("next billing date is in the past"))
		}
		// set the next billing date to the next billing date
		subscription.RenewsAt = nextCharge
	}

	if behaviour == subscriptions.StartNewBillingPeriod {
		// set the next billing date to the current date
		// add a bit of a buffer to avoid charging immediately
		nextCharge := time.Now().UTC().Add(time.Second * 20)
		subscription.BillingAnchor = nextCharge.Day()
		subscription.RenewsAt = nextCharge
	}
	subscription.Status = entities.SubscriptionStatusActive

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	return newSub, nil
}

// CancelSubscription
// A canceled subscription will continue through its current billing cycle. At the end of the current billing cycle the subscription will expire and the customer will no longer be billed.
// Canceled subscriptions can be reactivated until the end of the billing cycle
func (s SubscriptionService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Cancelling subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status == entities.SubscriptionStatusCancelled {
		s.logger.Info("Subscription is already cancelled")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is already cancelled", nil)
	}

	// set the subscription status to cancelled
	// set the cancelAt date to the next billing date
	cancelledAt := time.Now().UTC()
	subscription.Status = entities.SubscriptionStatusCancelled
	subscription.CancelAt = subscription.RenewsAt
	subscription.CancelledAt = cancelledAt
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s SubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, subscription.OrgId, subscription.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return entities.Customer{}, err
	}

	return customer, nil
}

func (s SubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.PaymentMethod, error) {
	s.logger.Infof("Fetching payment method for subscription [%s] %s - %s",
		subscription.OrgId, subscription.Id, subscription.PaymentMethodId)

	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, subscription.OrgId, subscription.PaymentMethodId)
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return entities.PaymentMethod{}, err
	}

	return paymentMethod, nil
}

func (s SubscriptionService) FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error) {
	s.logger.Info("Fetching payment method for subscription", "orgId", pk.OrgId, "id", pk.Id)

	payments, total, err := s.paymentRepository.FindBySubscriptionId(ctx, pk.OrgId, pk.Id, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return nil, 0, err
	}

	return payments, total, nil
}

func (s SubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	s.logger.Info("Recording subscription payment and updating subscription")
	subscription := input.Subscription
	charge := input.ChargeResult

	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
	}

	payment := entities.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		PspId:          charge.PspId,
		Reference:      charge.Reference,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Status:         charge.Status,
		Currency:       charge.Currency,
		Amount:         charge.Amount,
		PspFee:         0,
		PlatformFee:    0,
		NetAmount:      subscription.Amount,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payment.SetMetadata(subscription.Metadata)

	payment, err := s.paymentRepository.Create(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to create payment", err.Error())
	}

	// update the subscription status
	lastCharge := time.Now().UTC()
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = lastCharge
	subscription.Retries = 0
	subscription.NextRetryAt = time.Time{}

	if subscription.Cycles != 0 && subscription.CyclesProcessed >= subscription.Cycles {
		// this is the last charge for a subscription
		subscription.Status = entities.SubscriptionStatusCompleted
		subscription.EndsAt = lastCharge
		subscription.RenewsAt = time.Time{}
		subscription.CurrentPeriodEnd = time.Time{}
		subscription.CurrentPeriodStart = time.Time{}
	} else {
		// this is a normal recurring charge that needs to move to the new billing cycle
		subscription.Status = entities.SubscriptionStatusActive
		nextCharge := subscription.CalculateNextBillingDate()
		subscription.RenewsAt = nextCharge
		subscription.CurrentPeriodStart = subscription.CurrentPeriodEnd
		subscription.CurrentPeriodEnd = nextCharge
	}

	s.logger.Infof("[%s][%s] subscription charged, updating with new values [%s]",
		subscription.OrgId,
		subscription.Id,
		subscription.Status)

	// Update the subscription in the database
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	// Publish the events
	if newSub.Status == entities.SubscriptionStatusExpired {
		_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionStatusExpired, newSub)
	}

	_ = s.pubsub.Publish(
		subscription.OrgId,
		topic.SubscriptionPaymentChargeSuccess,
		topic.NewSubscriptionPaymentChargeSuccessEvent(subscription, payment),
	)

	return newSub, nil
}

func (s SubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	s.logger.Info("Charge failure happened", "orgId", input.Subscription.OrgId, "id", input.Subscription.Id)

	subscription := input.Subscription
	charge := input.ChargeResult

	s.logger.Infof("Subscription [%s] charge failed with reason [%s][%s][chargeResult status = %s]",
		subscription.Id, charge.ErrorCode, charge.ErrorReason, charge.Status)
	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
	}

	if subscription.Retries < 3 {
		// update the subscription status
		subscription.Status = entities.SubscriptionStatusRetry
		nextCharge := subscription.CalculateNextBillingDate()
		subscription.RenewsAt = nextCharge
		subscription.NextRetryAt = nextCharge
		subscription.Retries++
	} else {
		subscription.Status = entities.SubscriptionStatusPastDue
		subscription.Retries = 0
		subscription.NextRetryAt = time.Time{}

		_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionStatusPastDue, subscription)
	}

	s.logger.Infof("[%s][%s] nextCharge=[%s]", subscription.OrgId, subscription.Id, subscription.RenewsAt)
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionPaymentChargeFailed, map[string]interface{}{
		"subscription":  subscription,
		"charge_result": charge,
	})

	return newSub, nil
}
