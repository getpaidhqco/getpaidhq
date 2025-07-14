package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/security"
	"payloop/internal/lib"
	"time"
)

type SubscriptionService struct {
	sessionRepository      repositories.SessionRepository
	settingRepository      repositories.SettingRepository
	cartRepository         repositories.CartRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	paymentRepository      repositories.PaymentRepository
	orderItemRepository    repositories.OrderItemRepository
	workflowService        interfaces.WorkflowService
	gatewayFactory         factories.GatewayFactory
	tokenVault             security.TokenVault
	pubsub                 events.NotificationPublisher
	logger                 logger.Logger
	billingService         interfaces.BillingService
}

func NewSubscriptionService(
	sessionRepository repositories.SessionRepository,
	settingRepository repositories.SettingRepository,
	cartRepository repositories.CartRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	orderItemRepository repositories.OrderItemRepository,
	customerRepository repositories.CustomerRepository,
	orderRepository repositories.OrderRepository,
	paymentRepository repositories.PaymentRepository,
	tokenVault security.TokenVault,
	pubsub events.NotificationPublisher,
	gatewayFactory factories.GatewayFactory,
	logger logger.Logger,
	billingService interfaces.BillingService,
) interfaces.SubscriptionService {

	_, err := pubsub.Subscribe("subscription.workflow.>", func(topic string, data []byte) {
		logger.Infof("Received message from %s", topic)
	})
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

	return SubscriptionService{
		settingRepository:      settingRepository,
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		paymentRepository:      paymentRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		orderItemRepository:    orderItemRepository,
		subscriptionRepository: subscriptionRepository,
		tokenVault:             tokenVault,
		pubsub:                 pubsub,
		logger:                 logger,
		gatewayFactory:         gatewayFactory,
		billingService:         billingService,
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

func (s SubscriptionService) Create(ctx context.Context, orgId string, input dto.CreateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Creating new subscription", "orgId", orgId)

	// Convert application DTO to domain entity input
	// In a real implementation, we would fetch the variant and price details
	// to populate the subscription correctly
	domainInput := entities.CreateSubscriptionInput{
		OrgId:           orgId,
		PaymentMethodId: input.PaymentMethodId,
		Metadata:        input.Metadata,
		// These fields would typically be populated from the variant and price
		Amount:            0, // Would be populated from price
		Currency:          "", // Would be populated from price
		BillingInterval:   "", // Would be populated from price
		BillingIntervalQty: 0, // Would be populated from price
	}

	// If trial period is specified, convert to appropriate interval
	if input.TrialPeriodDays > 0 {
		domainInput.TrialInterval = "day"
		domainInput.TrialIntervalQty = input.TrialPeriodDays
	}

	subscription := entities.NewFromCreateInput(domainInput)
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
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
		nextCharge := subscription.RenewsAt
		if nextCharge.Before(time.Now().UTC()) {
			return entities.Subscription{}, lib.NewCustomError(lib.BadRequestError, "can't continue existing billing period, start a new period", errors.New("next billing date is in the past"))
		}
	}

	if behaviour == subscriptions.StartNewBillingPeriod {
		s.logger.Debugf(`Starting new billing period..`)
		// set the next billing date to the current date
		// add a bit of a buffer to avoid charging immediately
		nextCharge := time.Now().UTC().Add(time.Second * 20)
		subscription.BillingAnchor = nextCharge.Day()
		subscription.RenewsAt = nextCharge
		subscription.CurrentPeriodStart = nextCharge
		subscription.CurrentPeriodEnd = subscription.AddBillingInterval(nextCharge)
	}

	subscription.Status = entities.SubscriptionStatusActive
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

func (s SubscriptionService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	s.logger.Infof("Updating billing anchor for subscription %s", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return dto.UpdateBillingAnchorResult{}, err
		}
		return dto.UpdateBillingAnchorResult{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// Calculate proration details and update billing anchor
	prorationDetails := subscription.UpdateBillingAnchor(input.BillingAnchor, string(input.ProrationMode))

	// Save the updated subscription
	updatedSubscription, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return dto.UpdateBillingAnchorResult{}, err
	}

	result := dto.UpdateBillingAnchorResult{
		Subscription: updatedSubscription,
		ProrationDetails: dto.ProrationDetails{
			CreditAmount:       prorationDetails.CreditAmount,
			DaysCredited:       prorationDetails.DaysCredited,
			CurrentPeriodStart: prorationDetails.CurrentPeriodStart,
			CurrentPeriodEnd:   prorationDetails.CurrentPeriodEnd,
			OldBillingAnchor:   prorationDetails.OldBillingAnchor,
			NewBillingAnchor:   prorationDetails.NewBillingAnchor,
			NewPeriodStart:     prorationDetails.NewPeriodStart,
			NewPeriodEnd:       prorationDetails.NewPeriodEnd,
		},
	}

	return result, nil
}

func (s SubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, subscription.OrgId, subscription.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return entities.Customer{}, err
	}

	return customer, nil
}

func (s SubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error) {
	s.logger.Infof("Fetching secure payment method for subscription [%s] %s - %s",
		subscription.OrgId, subscription.Id, subscription.PaymentMethodId)

	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, subscription.OrgId, subscription.PaymentMethodId)
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return entities.SecurePaymentMethod{}, err
	}

	// Wrap in secure payment method for token access
	securePaymentMethod := entities.NewSecurePaymentMethod(paymentMethod, s.tokenVault)
	return securePaymentMethod, nil
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
		s.logger.Error("Failed to create payment", err.Error())
	}

	// update the subscription status
	lastCharge := time.Now().UTC()
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = lastCharge

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
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return entities.Subscription{}, err
	}

	// Publish the events
	if newSub.Status == entities.SubscriptionStatusExpired {
		_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionStatusExpired, newSub)
	}
	if newSub.Status == entities.SubscriptionStatusCompleted {
		_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionStatusCompleted, newSub)
	}

	_ = s.pubsub.Publish(
		subscription.OrgId,
		topic.SubscriptionPaymentChargeSuccess,
		topic.NewSubscriptionPaymentChargeSuccessEvent(subscription, payment),
	)

	return newSub, nil
}

// HandleSubscriptionChargeFailure logs the failed payment and updates the subscription status to past_due.
func (s SubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	s.logger.Info("Charge failure happened",
		"orgId", input.Subscription.OrgId,
		"id", input.Subscription.Id,
		"reason", input.ChargeResult.ErrorReason)

	subscription := input.Subscription
	charge := input.ChargeResult

	s.logger.Infof("Subscription [%s] charge failed with reason [%s][%s][chargeResult status = %s][]",
		subscription.Id,
		charge.ErrorCode,
		charge.ErrorReason,
		charge.Status)
	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
	}

	// store the failed payment
	payment := entities.Payment{
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
		s.logger.Error("Failed to create payment", err.Error())
	}

	s.logger.Debug("Created payment for subscription")

	// set the subscription status to past_due
	subscription.Status = entities.SubscriptionStatusPastDue
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return entities.Subscription{}, err
	}

	// Publish the events
	_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionPaymentChargeFailed, map[string]interface{}{
		"subscription":  subscription,
		"charge_result": charge,
	})
	// Publish the events
	_ = s.pubsub.Publish(subscription.OrgId, topic.PaymentFailed, payment)
	_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionStatusPastDue, newSub)

	return newSub, nil
}

func (s SubscriptionService) GetRetryPolicy(ctx context.Context, orgId string) subscriptions.RetryPolicy {
	defaultPolicy := subscriptions.RetryPolicy{
		RetryAttempts: 3,
		RetryInterval: subscriptions.RetryIntervalDay,
		RetryPeriod:   10,
		FailureAction: subscriptions.FailureActionCancel,
	}
	setting, err := s.GetOrgSubscriptionSettings(ctx, orgId)
	if err != nil {
		s.logger.Infof(`Retry policy not set, using default policy`)
		return defaultPolicy
	}

	return setting.RetryPolicy
}

// ChangeSubscriptionPlan changes a subscription's plan to a different variant/price
func (s SubscriptionService) ChangeSubscriptionPlan(ctx context.Context, input subscriptions.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error) {
	s.logger.Info("Changing subscription plan", "orgId", input.OrgId, "id", input.Id)

	// 1. Validate subscription state and eligibility
	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return nil, nil, err
	}

	// Check if subscription is in a valid state for plan change
	if !isValidStateForPlanChange(subscription.Status) {
		return nil, nil, lib.NewCustomError(lib.BadRequestError, "subscription is not in a valid state for plan change", nil)
	}

	// Check if subscription has any usage-based items
	// For now, we only support fixed subscriptions
	for _, item := range subscription.Items {
		if item.HasUsage {
			return nil, nil, lib.NewCustomError(lib.BadRequestError, "changing plans for usage-based subscriptions is not supported yet", nil)
		}
	}

	// If no items exist, return an error
	if len(subscription.Items) == 0 {
		return nil, nil, lib.NewCustomError(lib.BadRequestError, "subscription has no items", nil)
	}

	// Store the original values for the plan change record
	// Use the first subscription item for backward compatibility
	item := subscription.Items[0]
	fromProductId := item.ProductId
	fromVariantId := item.VariantId
	fromPriceId := item.PriceId
	fromAmount := item.Amount * int64(item.Quantity)

	// 4. Calculate proration (if applicable)
	var prorationAmount int64 = 0
	effectiveDate := time.Now().UTC()

	if input.EffectiveDate == "next_billing_cycle" {
		effectiveDate = subscription.RenewsAt
	} else if input.ProrationMode != "none" {
		// Calculate proration based on the current billing period
		prorationDetails := subscription.CalculateProrationDetails(
			input.ProrationMode,
			time.Now().UTC(),
			0, 0, time.Time{}, time.Time{},
		)
		prorationAmount = int64(prorationDetails.CreditAmount)
	}

	// 5. Update subscription item fields with the new plan details
	for i := range subscription.Items {
		if subscription.Items[i].Id == item.Id {
			subscription.Items[i].VariantId = input.NewVariantId
			subscription.Items[i].PriceId = input.NewPriceId
			subscription.Items[i].UpdatedAt = time.Now().UTC()
			break
		}
	}

	// Update subscription's updated timestamp
	subscription.UpdatedAt = time.Now().UTC()

	// Update the subscription in the database
	updatedSubscription, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return nil, nil, err
	}

	// Get the updated item from the updated subscription
	var updatedItem entities.SubscriptionItem
	if len(updatedSubscription.Items) > 0 {
		updatedItem = updatedSubscription.Items[0]
	} else {
		updatedItem = item
	}

	// 6. Create SubscriptionPlanChange record
	toAmount := updatedItem.Amount * int64(updatedItem.Quantity)
	changeType := "switch"
	if toAmount > fromAmount {
		changeType = "upgrade"
	} else if toAmount < fromAmount {
		changeType = "downgrade"
	}

	planChange := entities.SubscriptionPlanChange{
		Id:              lib.GenerateId("spc"),
		OrgId:           input.OrgId,
		SubscriptionId:  input.Id,
		FromProductId:   fromProductId,
		FromVariantId:   fromVariantId,
		FromPriceId:     fromPriceId,
		FromAmount:      fromAmount,
		ToProductId:     updatedItem.ProductId,
		ToVariantId:     updatedItem.VariantId,
		ToPriceId:       updatedItem.PriceId,
		ToAmount:        toAmount,
		ChangeType:      changeType,
		EffectiveDate:   effectiveDate,
		ProrationMode:   input.ProrationMode,
		ProrationAmount: prorationAmount,
		Reason:          input.Reason,
		InitiatedBy:     "customer", // This could be parameterized
		Metadata:        make(map[string]string),
		CreatedAt:       time.Now().UTC(),
	}

	createdPlanChange, err := s.subscriptionRepository.CreatePlanChange(ctx, planChange)
	if err != nil {
		s.logger.Error("Failed to create plan change record", err.Error())
		// Continue anyway, as the subscription has already been updated
	}

	// 7. Emit subscription.plan_changed event
	event := topic.NewSubscriptionPlanChangedEvent(updatedSubscription, createdPlanChange)
	_ = s.pubsub.Publish(subscription.OrgId, topic.SubscriptionPlanChanged, event)

	return &updatedSubscription, &createdPlanChange, nil
}

// isValidStateForPlanChange checks if the subscription is in a valid state for plan change
func isValidStateForPlanChange(status entities.SubscriptionStatus) bool {
	return status == entities.SubscriptionStatusActive ||
		status == entities.SubscriptionStatusTrial ||
		status == entities.SubscriptionStatusPastDue
}

func (s SubscriptionService) GetOrgSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	subscription, err := s.settingRepository.FindById(ctx, orgId, orgId, "subscriptions")
	if err != nil {
		s.logger.Warn(`failed to find org subscription settings`, "parentId", orgId)
		return settings.Subscription{}, err
	}

	var subscriptionSettings settings.Subscription
	err = json.Unmarshal([]byte(subscription.Value), &subscriptionSettings)
	if err != nil {
		s.logger.Warn(`invalid subscription settings format`, "parentId", orgId, "id", "subscriptions")
		return settings.Subscription{}, errors.New("invalid subscription settings format")
	}

	return subscriptionSettings, nil
}

// ProcessSubscriptionCharge handles the complete subscription charging process
// including billing calculation, payment processing, and result handling
func (s SubscriptionService) ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	// Get latest subscription data
	currentSubscription, err := s.subscriptionRepository.FindById(ctx, subscription.OrgId, subscription.Id)
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Skip charging if subscription is not active
	if !currentSubscription.IsRunning() {
		s.logger.Warn("Skipping charge for non-active subscription", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "status", currentSubscription.Status)
		return payments.ChargeResult{
			Status:      payments.PaymentStatusFailed,
			ErrorReason: "Subscription is not active",
		}, nil
	}

	// Calculate billing amount
	billingCalculation, err := s.billingService.CalculateBillingAmount(ctx, currentSubscription)
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to calculate billing amount: %w", err)
	}

	// Skip charging if amount is zero
	if billingCalculation.TotalAmount <= 0 {
		return payments.ChargeResult{
			Status:      payments.PaymentStatusSucceeded,
			Amount:      0,
			Currency:    billingCalculation.Currency,
			ProcessedAt: time.Now(),
		}, nil
	}

	// Get payment gateway
	gateway, err := s.gatewayFactory.NewGateway(ctx, currentSubscription.OrgId, string(currentSubscription.PspId))
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to get payment gateway: %w", err)
	}

	// Get customer and payment method
	customer, err := s.GetSubscriptionCustomer(ctx, currentSubscription)
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to get customer: %w", err)
	}

	paymentMethod, err := s.GetSubscriptionPaymentMethod(ctx, currentSubscription)
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to get payment method: %w", err)
	}

	decryptedToken, err := paymentMethod.GetToken(ctx)
	if err != nil {
		return payments.ChargeResult{}, fmt.Errorf("failed to decrypt payment token: %w", err)
	}

	// Process payment
	chargeResult := gateway.ChargePayment(ctx, payment_providers.ChargePaymentCommand{
		OrgId:          currentSubscription.OrgId,
		OrderId:        currentSubscription.OrderId,
		SubscriptionId: currentSubscription.Id,
		Amount:         billingCalculation.TotalAmount,
		Currency:       billingCalculation.Currency,
		PaymentMethod: payment_providers.PaymentMethod{
			PspId:       paymentMethod.Id,
			Name:        paymentMethod.Name,
			Type:        string(paymentMethod.Type),
			IsRecurring: true,
			Token:       decryptedToken,
		},
		Customer: customer,
	})

	// Handle gateway errors
	if chargeResult.Status == payment_providers.GatewayError {
		s.logger.Error("Gateway error while charging subscription",
			"orgId", currentSubscription.OrgId,
			"subscriptionId", currentSubscription.Id,
			"error", chargeResult.ErrorReason,
			"psp", string(currentSubscription.PspId),
		)
		return payments.ChargeResult{}, errors.New("gateway error: " + chargeResult.ErrorReason)
	}

	// Convert to domain charge result
	rawData, _ := json.Marshal(chargeResult.PspResponse)

	var status payments.PaymentStatus
	var completedAt time.Time

	switch chargeResult.Status {
	case payment_providers.ChargePaymentStatusSuccess:
		status = payments.PaymentStatusSucceeded
		completedAt = time.Now()
	case payment_providers.ChargePaymentStatusPending:
		status = payments.PaymentStatusPending
	case payment_providers.ChargePaymentStatusError:
		status = payments.PaymentStatusFailed
	}

	domainChargeResult := payments.ChargeResult{
		Psp:         chargeResult.Psp,
		Amount:      chargeResult.AmountCharged,
		Status:      status,
		Currency:    currentSubscription.Currency,
		ErrorReason: chargeResult.ErrorReason,
		ErrorCode:   chargeResult.ErrorCode,
		PspId:       chargeResult.PspId,
		Reference:   chargeResult.Reference,
		ProcessedAt: completedAt,
		RawData:     string(rawData),
	}

	// Handle charge result (success or failure)
	if domainChargeResult.Status == payments.PaymentStatusSucceeded {
		_, err = s.HandleSubscriptionChargeSuccess(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: currentSubscription,
			ChargeResult: domainChargeResult,
		})
	} else {
		_, err = s.HandleSubscriptionChargeFailure(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: currentSubscription,
			ChargeResult: domainChargeResult,
		})
	}

	if err != nil {
		s.logger.Error("Failed to handle charge result",
			"orgId", currentSubscription.OrgId,
			"subscriptionId", currentSubscription.Id,
			"status", domainChargeResult.Status,
			"error", err.Error(),
		)
		return domainChargeResult, fmt.Errorf("failed to handle charge result: %w", err)
	}

	return domainChargeResult, nil
}
