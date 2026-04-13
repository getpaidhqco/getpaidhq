package service

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

// SubscriptionService merges the previous SubscriptionService + SubscriptionOrchestrationService.
// Workflow orchestration is built-in: when a subscription state changes, the workflow engine is signaled.
type SubscriptionService struct {
	sessionRepository      port.SessionRepository
	settingRepository      port.SettingRepository
	cartRepository         port.CartRepository
	orderRepository        port.OrderRepository
	customerRepository     port.CustomerRepository
	subscriptionRepository port.SubscriptionRepository
	paymentRepository      port.PaymentRepository
	engine                 port.Engine
	pubsub                 port.PubSub
	logger                 port.Logger
}

func NewSubscriptionService(
	sessionRepository port.SessionRepository,
	settingRepository port.SettingRepository,
	cartRepository port.CartRepository,
	subscriptionRepository port.SubscriptionRepository,
	customerRepository port.CustomerRepository,
	orderRepository port.OrderRepository,
	paymentRepository port.PaymentRepository,
	pubsub port.PubSub,
	logger port.Logger,
	engine port.Engine,
) *SubscriptionService {

	_, err := pubsub.Subscribe("subscription.workflow.>", func(topic string, data []byte) {
		logger.Info("received message", "topic", topic)
	})
	if err != nil {
		logger.Error("failed to subscribe to topic", "error", err)
	}

	return &SubscriptionService{
		settingRepository:      settingRepository,
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		paymentRepository:      paymentRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		subscriptionRepository: subscriptionRepository,
		pubsub:                 pubsub,
		logger:                 logger,
		engine:                 engine,
	}
}

func (s *SubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	s.logger.Info("creating subscriptions for order", "orgId", orgId, "orderId", orderId)
	var subs []domain.Subscription
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("failed to find order", "error", err)
		return subs, err
	}

	orderItems, err := s.orderRepository.FindOrderItemsByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("failed to find order items", "error", err)
		return subs, err
	}

	for _, item := range orderItems {
		subscription := domain.NewSubscriptionFromOrderItem(item)
		if order.Status == domain.OrderStatusCompleted {
			subscription.Status = domain.SubscriptionStatusActive
		}

		_, err := s.subscriptionRepository.Create(ctx, subscription)
		if err != nil {
			s.logger.Error("failed to create subscription", "item", item, "error", err)
			return subs, err
		}
		subs = append(subs, subscription)
	}

	s.logger.Info("subscriptions created", "count", len(subs))
	return subs, nil
}

func (s *SubscriptionService) Create(ctx context.Context, input domain.CreateSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("creating new subscription", "orgId", input.OrgId)

	subscription := domain.NewFromCreateInput(input)
	subscription, err := s.subscriptionRepository.Create(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to create subscription", "error", err)
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCreated, subscription)
	return subscription, nil
}

func (s *SubscriptionService) Update(ctx context.Context, input domain.UpdateSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("updating subscription", "orgId", input.OrgId, "subscriptionId", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		return domain.Subscription{}, err
	}

	if input.Status != subscription.Status {
		s.logger.Info("updating status", "status", input.Status)
		subscription.Status = input.Status
	}

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.GetSubscriptionTopic(subscription.Status), newSub)
	return newSub, err
}

func (s *SubscriptionService) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Info("fetching subscription", "orgId", orgId, "subscriptionId", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		return domain.Subscription{}, err
	}
	return subscription, nil
}

// Activate a subscription and start the workflow engine.
func (s *SubscriptionService) Activate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Info("marking subscription active", "orgId", orgId, "subscriptionId", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		return domain.Subscription{}, err
	}

	subscription.Status = domain.SubscriptionStatusActive
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	err = s.engine.StartSubscriptionWorkflow(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to start workflow", "error", err)
		return domain.Subscription{}, err
	}

	return subscription, nil
}

// PauseSubscription pauses a subscription and signals the workflow engine.
func (s *SubscriptionService) PauseSubscription(ctx context.Context, input domain.PauseSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("pausing subscription", "orgId", input.OrgId, "subscriptionId", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status == domain.SubscriptionStatusPaused {
		s.logger.Info("subscription is already paused")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is paused already", nil)
	}

	subscription.Status = domain.SubscriptionStatusPaused
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	// Signal the workflow engine
	err = s.engine.UpdateSubscriptionWorkflow(ctx, "subscription.paused", subscription)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionPaused, subscription)
	return subscription, nil
}

func (s *SubscriptionService) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Subscription, int, error) {
	subs, total, err := s.subscriptionRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("failed to list subscriptions", "error", err)
		return nil, 0, err
	}
	return subs, total, nil
}

// ResumeSubscription resumes a paused subscription and signals the workflow engine.
func (s *SubscriptionService) ResumeSubscription(ctx context.Context, input domain.ResumeSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("resuming subscription", "orgId", input.OrgId, "subscriptionId", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status != domain.SubscriptionStatusPaused &&
		subscription.Status != domain.SubscriptionStatusPastDue {
		s.logger.Info("subscription is not paused")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is not paused", nil)
	}

	behaviour := domain.ContinueExistingBillingPeriod
	if input.ResumeBehavior != "" {
		behaviour = input.ResumeBehavior
	}

	if behaviour == domain.ContinueExistingBillingPeriod {
		nextCharge := subscription.CalculateNextBillingDate()
		if nextCharge.Before(time.Now().UTC()) {
			return domain.Subscription{}, lib.NewCustomError(lib.BadRequestError, "can't continue existing billing period, start a new period", errors.New("next billing date is in the past"))
		}
		subscription.RenewsAt = nextCharge
	}

	if behaviour == domain.StartNewBillingPeriod {
		s.logger.Debug("starting new billing period")
		nextCharge := time.Now().UTC().Add(time.Second * 20)
		subscription.BillingAnchor = nextCharge.Day()
		subscription.RenewsAt = nextCharge
		subscription.CurrentPeriodStart = nextCharge
		subscription.CurrentPeriodEnd = subscription.AddBillingInterval(nextCharge)
	}

	subscription.Status = domain.SubscriptionStatusActive
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	// Signal the workflow engine
	err = s.engine.UpdateSubscriptionWorkflow(ctx, port.TopicSubscriptionResumed, newSub)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(newSub.OrgId, port.TopicSubscriptionResumed, newSub)
	return newSub, nil
}

// CancelSubscription cancels a subscription. It will continue through its current billing cycle.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, input domain.CancelSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("cancelling subscription", "orgId", input.OrgId, "subscriptionId", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	if subscription.Status == domain.SubscriptionStatusCancelled {
		s.logger.Info("subscription is already cancelled")
		return subscription, lib.NewCustomError(lib.BadRequestError, "subscription is already cancelled", nil)
	}

	cancelledAt := time.Now().UTC()
	subscription.Status = domain.SubscriptionStatusCancelled
	subscription.CancelAt = subscription.RenewsAt
	subscription.CancelledAt = cancelledAt
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	// Signal the workflow engine
	s.logger.Debug("updating workflow for subscription", "subscriptionId", subscription.Id, "topic", port.TopicSubscriptionCancelled)
	err = s.engine.UpdateSubscriptionWorkflow(ctx, port.TopicSubscriptionCancelled, subscription)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCancelled, subscription)
	return subscription, nil
}

func (s *SubscriptionService) UpdateBillingAnchor(ctx context.Context, input domain.UpdateBillingAnchorInput) (domain.ProrationDetails, error) {
	s.logger.Info("updating billing anchor", "subscriptionId", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("failed to find subscription", "error", err)
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.ProrationDetails{}, err
		}
		return domain.ProrationDetails{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	prorationDetails := subscription.UpdateBillingAnchor(input.BillingAnchor, string(input.ProrationMode))

	_, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.ProrationDetails{}, err
	}

	// Publish billing anchor changed event
	sub, findErr := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if findErr == nil {
		_ = s.pubsub.Publish(sub.OrgId, port.TopicSubscriptionBillingAnchorChanged, sub)
	}

	return prorationDetails, nil
}

// UpdateWorkflowState refreshes the workflow state from the database. Used for debugging and error recovery.
func (s *SubscriptionService) UpdateWorkflowState(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Info("updating workflow state", "orgId", orgId, "subscriptionId", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		return domain.Subscription{}, lib.NewCustomError(lib.NotFoundError, "Not found", err)
	}

	err = s.engine.UpdateSubscriptionWorkflow(ctx, "refresh-state", subscription)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, err.Error(), err)
	}

	return subscription, nil
}

func (s *SubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription domain.Subscription) (domain.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, subscription.OrgId, subscription.CustomerId)
	if err != nil {
		s.logger.Error("failed to find customer", "error", err)
		return domain.Customer{}, err
	}
	return customer, nil
}

func (s *SubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription domain.Subscription) (domain.PaymentMethod, error) {
	s.logger.Info("fetching payment method for subscription", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "paymentMethodId", subscription.PaymentMethodId)

	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, subscription.OrgId, subscription.PaymentMethodId)
	if err != nil {
		s.logger.Error("failed to find payment method", "error", err)
		return domain.PaymentMethod{}, err
	}
	return paymentMethod, nil
}

func (s *SubscriptionService) FindSubscriptionPayments(ctx context.Context, pk domain.EntityKey, pagination domain.Pagination) ([]domain.Payment, int, error) {
	s.logger.Info("fetching payments for subscription", "orgId", pk.OrgId, "subscriptionId", pk.Id)

	payments, total, err := s.paymentRepository.FindBySubscriptionId(ctx, pk.OrgId, pk.Id, pagination)
	if err != nil {
		s.logger.Error("failed to find payments", "error", err)
		return nil, 0, err
	}
	return payments, total, nil
}

func (s *SubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input domain.SubscriptionChargeInput) (domain.Subscription, error) {
	s.logger.Info("recording subscription payment and updating subscription")
	subscription := input.Subscription
	charge := input.ChargeResult

	if subscription.Id == "" {
		return domain.Subscription{}, errors.New("subscription is empty")
	}

	payment := domain.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		Psp:            charge.Psp,
		PspId:          charge.PspId,
		Reference:      charge.Reference,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Status:         charge.Status,
		Recurring:      true,
		Currency:       charge.Currency,
		Amount:         charge.Amount,
		PspFee:         0,
		PlatformFee:    0,
		NetAmount:      subscription.Amount,
		Metadata:       nil,
		CompletedAt:    input.ChargeResult.ProcessedAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payment.SetMetadata(subscription.Metadata)

	payment, err := s.paymentRepository.Create(ctx, payment)
	if err != nil {
		s.logger.Error("failed to create payment", "error", err)
	}

	lastCharge := time.Now().UTC()
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = lastCharge
	subscription.Retries = 0
	subscription.NextRetryAt = time.Time{}

	if subscription.Cycles != 0 && subscription.CyclesProcessed >= subscription.Cycles {
		subscription.Status = domain.SubscriptionStatusCompleted
		subscription.EndsAt = lastCharge
		subscription.RenewsAt = time.Time{}
		subscription.CurrentPeriodEnd = time.Time{}
		subscription.CurrentPeriodStart = time.Time{}
	} else {
		subscription.Status = domain.SubscriptionStatusActive
		nextCharge := subscription.CalculateNextBillingDate()
		subscription.RenewsAt = nextCharge
		subscription.CurrentPeriodStart = subscription.CurrentPeriodEnd
		subscription.CurrentPeriodEnd = nextCharge
	}

	s.logger.Info("subscription charged", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "status", subscription.Status)

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	if newSub.Status == domain.SubscriptionStatusExpired {
		_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionExpired, newSub)
	}
	if newSub.Status == domain.SubscriptionStatusCompleted {
		_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCompleted, newSub)
	}

	_ = s.pubsub.Publish(
		subscription.OrgId,
		port.TopicSubscriptionPaymentChargeSuccess,
		port.NewSubscriptionPaymentChargeSuccessEvent(subscription, payment),
	)

	return newSub, nil
}

func (s *SubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input domain.SubscriptionChargeInput) (domain.Subscription, error) {
	s.logger.Info("charge failure happened",
		"orgId", input.Subscription.OrgId,
		"subscriptionId", input.Subscription.Id,
		"reason", input.ChargeResult.ErrorReason)

	subscription := input.Subscription
	charge := input.ChargeResult

	s.logger.Info("subscription charge failed",
		"subscriptionId", subscription.Id,
		"errorCode", charge.ErrorCode,
		"errorReason", charge.ErrorReason,
		"chargeStatus", charge.Status,
		"retries", subscription.Retries)
	if subscription.Id == "" {
		return domain.Subscription{}, errors.New("subscription is empty")
	}

	payment := domain.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		Psp:            charge.Psp,
		PspId:          charge.PspId,
		Reference:      charge.Reference,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Status:         charge.Status,
		Recurring:      true,
		Currency:       charge.Currency,
		Amount:         charge.Amount,
		PspFee:         0,
		PlatformFee:    0,
		NetAmount:      subscription.Amount,
		Metadata:       nil,
		CompletedAt:    input.ChargeResult.ProcessedAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payment.SetMetadata(subscription.Metadata)

	payment, err := s.paymentRepository.Create(ctx, payment)
	if err != nil {
		s.logger.Error("failed to create payment", "error", err)
	}

	s.logger.Debug("created payment for subscription")

	retryPolicy := s.GetRetryPolicy(ctx, subscription.OrgId)
	s.logger.Debug("retry policy",
		"attempts", retryPolicy.RetryAttempts,
		"interval", retryPolicy.RetryInterval,
		"qty", retryPolicy.RetryPeriod,
		"action", retryPolicy.FailureAction,
	)

	nextRetryDate := retryPolicy.GetNextCharge(subscription)
	if nextRetryDate.IsZero() {
		s.logger.Debug("subscription has no more retries left", "subscriptionId", subscription.Id)
		if retryPolicy.FailureAction == domain.FailureActionMarkUnpaid {
			s.logger.Debug("marking as unpaid")
			subscription.Status = domain.SubscriptionStatusUnpaid
		}
		if retryPolicy.FailureAction == domain.FailureActionCancel {
			s.logger.Debug("cancelling")
			subscription.SetCancelled()
		}
	} else {
		s.logger.Debug("subscription next retry scheduled", "subscriptionId", subscription.Id, "nextRetryDate", nextRetryDate)
		subscription.Status = domain.SubscriptionStatusPastDue
		subscription.NextRetryAt = nextRetryDate
		subscription.Retries++
	}

	s.logger.Info("updating subscription after charge failure", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "nextChargeDate", subscription.GetNextChargeDate())
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to update subscription", "error", err)
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionPaymentChargeFailed, map[string]interface{}{
		"subscription":  subscription,
		"charge_result": charge,
	})
	_ = s.pubsub.Publish(subscription.OrgId, port.TopicPaymentCreated, payment)

	switch newSub.Status {
	case domain.SubscriptionStatusCancelled:
		_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCancelled, newSub)
	case domain.SubscriptionStatusUnpaid:
		_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionUnpaid, newSub)
	case domain.SubscriptionStatusExpired:
		_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionExpired, newSub)
	case domain.SubscriptionStatusPastDue:
		if subscription.Retries == 1 {
			_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionPastDue, newSub)
		}
	}

	return newSub, nil
}

func (s *SubscriptionService) GetRetryPolicy(ctx context.Context, orgId string) domain.RetryPolicy {
	defaultPolicy := domain.RetryPolicy{
		RetryAttempts: 3,
		RetryInterval: domain.RetryIntervalMinute,
		RetryPeriod:   4,
		FailureAction: domain.FailureActionCancel,
	}
	setting, err := s.settingRepository.FindById(ctx, orgId, "subscriptions", "retry_policy")
	if err != nil || setting.Value == "" {
		s.logger.Info("retry policy not set, using default policy")
		return defaultPolicy
	}

	var retryPolicy domain.RetryPolicy
	err = json.Unmarshal([]byte(setting.Value), &retryPolicy)
	if err != nil {
		s.logger.Error("failed to unmarshal retry policy", "error", err)
		return defaultPolicy
	}
	return retryPolicy
}
