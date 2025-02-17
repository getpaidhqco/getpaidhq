package subscriptions

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
	workflowService        interfaces.WorkflowService
	paymentGateway         payment_providers.Gateway
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
	paymentGateway payment_providers.Gateway,
	logger logger.Logger,
) interfaces.SubscriptionActivityService {

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
		paymentGateway:         paymentGateway,
	}
}

func (s SubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
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

func (s SubscriptionService) Create(ctx context.Context, input entities.CreateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Creating new subscription", "orgId", input.OrgId)

	subscription := entities.NewFromCreateInput(input)
	subscription, err := s.subscriptionRepository.Create(ctx, subscription)

	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
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

	_ = s.pubsub.Publish(subscription.OrgId, entities.GetTopicFromStatus(subscription.Status), newSub)
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

func (s SubscriptionService) Pause(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
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

	subscription.Status = entities.SubscriptionStatusPaused
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	// publi
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionPaused, subscription)

	return subscription, nil
}

func (s SubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, error) {
	subs, err := s.subscriptionRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list subscriptions", err.Error())
		return nil, err
	}

	return subs, nil
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
		subscription.RenewsAt = &nextCharge
	}

	if behaviour == subscriptions.StartNewBillingPeriod {
		// set the next billing date to the current date
		// add a bit of a buffer to avoid charging immediately
		nextCharge := time.Now().UTC().Add(time.Second * 20)
		subscription.BillingAnchor = nextCharge.Day()
		subscription.RenewsAt = &nextCharge
	}
	subscription.Status = entities.SubscriptionStatusActive

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	// Publish the resume event
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionActivated, newSub)
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionResumed, newSub)

	return subscription, nil
}

// CancelSubscription
// A canceled subscription will continue through its current billing cycle. At the end of the current billing cycle the subscription will expire and the customer will no longer be billed.
// Canceled subscriptions can be reactivated until the end of the billing cycle
func (s SubscriptionService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {
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

	if subscription.Status == entities.SubscriptionStatusCancelled {
		s.logger.Info("Subscription is already cancelled")
		return subscription, nil
	}

	// set the subscription status to cancelled
	// set the cancelAt date to the next billing date
	cancelledAt := time.Now().UTC()
	subscription.Status = entities.SubscriptionStatusCancelled
	subscription.CancelAt = subscription.RenewsAt
	subscription.CancelledAt = &cancelledAt
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	// publi
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionCancelled, subscription)

	return subscription, nil
}

//func (s SubscriptionService) ProcessSubscriptionCharge(ctx context.Context, input subscriptions.ProcessSubscriptionChargeInput) (payments.ChargeResult, error) {
//
//	subscription := input.Subscription
//	s.logger.Info("Processing subscription charge", "orgId", subscription.OrgId, "id", subscription.Id)
//
//	chargeResult, err := s.paymentGateway.ChargePayment(ctx, subscription)
//
//	return newSub, nil
//}

func (s SubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, subscription.OrgId, subscription.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return entities.Customer{}, err
	}

	return customer, nil
}

func (s SubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.PaymentMethod, error) {
	s.logger.Info("Fetching payment method for subscription", "orgId", subscription.OrgId, "subscriptionId", subscription.Id)

	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, subscription.OrgId, *subscription.PaymentMethodId)
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return entities.PaymentMethod{}, err
	}

	return paymentMethod, nil
}

func (s SubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
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
	payment, err := s.paymentRepository.Create(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to create payment", err.Error())
	}

	// update the subscription status
	lastCharge := time.Now().UTC()
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = &lastCharge
	subscription.Retries = 0
	subscription.NextRetryAt = nil

	if subscription.Cycles != 0 && subscription.CyclesProcessed >= subscription.Cycles {
		subscription.Status = entities.SubscriptionStatusCompleted
		subscription.EndsAt = &lastCharge
		subscription.RenewsAt = nil
	} else {
		subscription.Status = entities.SubscriptionStatusActive
		nextCharge := subscription.CalculateNextBillingDate()
		subscription.RenewsAt = &nextCharge
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

	_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionPaymentChargeSuccess, payment)

	return newSub, nil
}

func (s SubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	s.logger.Info("Charge failure happened", "orgId", input.Subscription.OrgId, "id", input.Subscription.Id)

	subscription := input.Subscription
	charge := input.ChargeResult

	s.logger.Infof("Subscription [%s] charge failed with reason [%s][%s]", subscription.Id, charge.ErrorCode, charge.ErrorReason)
	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
	}

	if subscription.Retries < 3 {
		// update the subscription status
		subscription.Status = entities.SubscriptionStatusRetry
		nextCharge := subscription.CalculateNextBillingDate()
		subscription.RenewsAt = &nextCharge
		subscription.NextRetryAt = &nextCharge
		subscription.Retries++
	} else {
		subscription.Status = entities.SubscriptionStatusPastDue
		subscription.Retries = 0
		subscription.NextRetryAt = nil

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
