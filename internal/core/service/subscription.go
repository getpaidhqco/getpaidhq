package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
)

// SubscriptionService is the narrow subscription service. It owns all
// subscription operations that do NOT signal the workflow engine: CRUD,
// charge-result handling (called from activities), and the DB-side of the
// lifecycle transitions (Pause/Resume/Cancel without signaling).
//
// The engine-aware operations live on SubscriptionOrchestrationService,
// which wraps this service. Splitting the two avoids a construction-time
// cycle between services, the engine, and the activities that the engine
// dispatches into the services.
type SubscriptionService struct {
	sessionRepository      port.SessionRepository
	settingRepository      port.SettingRepository
	cartRepository         port.CartRepository
	orderRepository        port.OrderRepository
	customerRepository     port.CustomerRepository
	subscriptionRepository port.SubscriptionRepository
	paymentRepository      port.PaymentRepository
	gatewayFactory         port.GatewayFactory
	pubsub                 port.PubSub
	errorReporter          lib.ErrorReporter
	logger                 port.Logger
	// tx wraps lifecycle state transitions in a SQL transaction so the
	// FindByIdForUpdate-then-Update sequence is atomic and SELECT FOR
	// UPDATE actually holds. May be nil in tests that don't drive a
	// transition path; transitionInTx falls back to running the
	// closure without a tx in that case.
	tx port.TxManager
}

// NewSubscriptionService wires the narrow subscription service. The
// startup subscription is now safety-wrapped (panic recovery) and
// errors surface to the caller — see SubscriptionEventBridge for the
// reasoning.
func NewSubscriptionService(
	sessionRepository port.SessionRepository,
	settingRepository port.SettingRepository,
	cartRepository port.CartRepository,
	subscriptionRepository port.SubscriptionRepository,
	customerRepository port.CustomerRepository,
	orderRepository port.OrderRepository,
	paymentRepository port.PaymentRepository,
	gatewayFactory port.GatewayFactory,
	pubsub port.PubSub,
	errorReporter lib.ErrorReporter,
	logger port.Logger,
	tx port.TxManager,
) (*SubscriptionService, error) {

	handler := safePubSubHandler(logger, "SubscriptionService.workflow-tap", func(topic string, data []byte) {
		logger.Infof("Received message from %s", topic)
	})
	if _, err := pubsub.Subscribe("subscription.workflow.>", handler); err != nil {
		return nil, err
	}

	return &SubscriptionService{
		settingRepository:      settingRepository,
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		paymentRepository:      paymentRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		subscriptionRepository: subscriptionRepository,
		gatewayFactory:         gatewayFactory,
		pubsub:                 pubsub,
		errorReporter:          errorReporter,
		logger:                 logger,
		tx:                     tx,
	}, nil
}

// transitionInTx runs `fn` inside a SQL transaction when one is
// available (production wiring). In tests that don't wire a
// TxManager we just call fn directly — the underlying fake repos
// don't honor isolation anyway, so the call-site semantics are
// preserved.
func (s *SubscriptionService) transitionInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.RunInTx(ctx, fn)
}

func (s *SubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	s.logger.Info("CreateSubscriptionsForOrder", "orgId", orgId, "orderId", orderId)
	var subs []domain.Subscription
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order", err.Error())
		return subs, err
	}

	orderItems, err := s.orderRepository.FindOrderItemsByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order items", err.Error())
		return subs, err
	}

	for _, item := range orderItems {
		subscription := domain.NewSubscriptionFromOrderItem(item)
		if order.Status == domain.OrderStatusCompleted {
			subscription.Status = domain.SubscriptionStatusActive
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

func (s *SubscriptionService) Create(ctx context.Context, input domain.CreateSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("Creating new subscription", "orgId", input.OrgId)

	subscription := domain.NewFromCreateInput(input)
	subscription, err := s.subscriptionRepository.Create(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed create subscriptions", err.Error())
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCreated, subscription)
	return subscription, nil
}

func (s *SubscriptionService) Update(ctx context.Context, input domain.UpdateSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("Updating subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return domain.Subscription{}, err
	}

	if input.Status != subscription.Status {
		s.logger.Infof("Updating status %s", input.Status)
		subscription.Status = input.Status
	}

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.GetSubscriptionTopic(subscription.Status), newSub)
	return newSub, err
}

func (s *SubscriptionService) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Info("Fetching", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return domain.Subscription{}, err
	}
	return subscription, nil
}

// Activate marks the subscription active in the database. Wrapped in
// a transaction with SELECT FOR UPDATE on the subscription row so that
// concurrent activate/pause/cancel transitions cannot read the same
// initial state and overwrite each other.
func (s *SubscriptionService) Activate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Info("Marking subscription active", "orgId", orgId, "id", id)

	var subscription domain.Subscription
	err := s.transitionInTx(ctx, func(ctx context.Context) error {
		sub, err := s.subscriptionRepository.FindByIdForUpdate(ctx, orgId, id)
		if err != nil {
			return err
		}
		sub.Status = domain.SubscriptionStatusActive
		sub, err = s.subscriptionRepository.Update(ctx, sub)
		if err != nil {
			return err
		}
		subscription = sub
		return nil
	})
	if err != nil {
		s.logger.Error("Activate failed", "err", err.Error())
		return domain.Subscription{}, err
	}

	return subscription, nil
}

// PauseSubscription updates the subscription state in the database.
// The orchestration wrapper additionally signals the workflow engine.
// The read-then-write is atomic via SELECT FOR UPDATE — see Activate.
func (s *SubscriptionService) PauseSubscription(ctx context.Context, input domain.PauseSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("Pausing subscription", "orgId", input.OrgId, "id", input.Id)

	var subscription domain.Subscription
	txErr := s.transitionInTx(ctx, func(ctx context.Context) error {
		sub, err := s.subscriptionRepository.FindByIdForUpdate(ctx, input.OrgId, input.Id)
		if err != nil {
			if _, ok := errors.AsType[lib.CustomError](err); ok {
				return err
			}
			return lib.NewCustomError(lib.InternalError, "", err)
		}
		if sub.Status == domain.SubscriptionStatusPaused {
			subscription = sub
			return lib.NewCustomError(lib.BadRequestError, "subscription is paused already", nil)
		}
		sub.Status = domain.SubscriptionStatusPaused
		sub, err = s.subscriptionRepository.Update(ctx, sub)
		if err != nil {
			return err
		}
		subscription = sub
		return nil
	})
	if txErr != nil {
		s.logger.Error("PauseSubscription failed", "err", txErr.Error())
		// Pre-existing-paused returns the loaded subscription with the
		// BadRequestError so handlers can render a useful body — keep
		// that contract.
		if _, ok := errors.AsType[lib.CustomError](txErr); ok && subscription.Id != "" {
			return subscription, txErr
		}
		return domain.Subscription{}, txErr
	}

	return subscription, nil
}

func (s *SubscriptionService) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Subscription, int, error) {
	subs, total, err := s.subscriptionRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list subscriptions", err.Error())
		return nil, 0, err
	}
	return subs, total, nil
}

// ResumeSubscription updates the subscription state in the database.
// The orchestration wrapper additionally signals the workflow engine.
// The read-then-write is atomic via SELECT FOR UPDATE — see Activate.
func (s *SubscriptionService) ResumeSubscription(ctx context.Context, input domain.ResumeSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("Resuming subscription", "orgId", input.OrgId, "id", input.Id)

	var newSub domain.Subscription
	txErr := s.transitionInTx(ctx, func(ctx context.Context) error {
		subscription, err := s.subscriptionRepository.FindByIdForUpdate(ctx, input.OrgId, input.Id)
		if err != nil {
			if _, ok := errors.AsType[lib.CustomError](err); ok {
				return err
			}
			return lib.NewCustomError(lib.InternalError, "", err)
		}

		if subscription.Status != domain.SubscriptionStatusPaused &&
			subscription.Status != domain.SubscriptionStatusPastDue {
			newSub = subscription
			return lib.NewCustomError(lib.BadRequestError, "subscription is not paused", nil)
		}

		behaviour := domain.ContinueExistingBillingPeriod
		if input.ResumeBehavior != "" {
			behaviour = input.ResumeBehavior
		}

		if behaviour == domain.ContinueExistingBillingPeriod {
			nextCharge := subscription.CalculateNextBillingDate()
			if nextCharge.Before(time.Now().UTC()) {
				return lib.NewCustomError(lib.BadRequestError, "can't continue existing billing period, start a new period", errors.New("next billing date is in the past"))
			}
			subscription.RenewsAt = nextCharge
		}

		if behaviour == domain.StartNewBillingPeriod {
			nextCharge := time.Now().UTC().Add(time.Second * 20)
			subscription.BillingAnchor = nextCharge.Day()
			subscription.RenewsAt = nextCharge
			subscription.CurrentPeriodStart = nextCharge
			subscription.CurrentPeriodEnd = subscription.AddBillingInterval(nextCharge)
		}

		subscription.Status = domain.SubscriptionStatusActive
		updated, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			return err
		}
		newSub = updated
		return nil
	})
	if txErr != nil {
		s.logger.Error("ResumeSubscription failed", "err", txErr.Error())
		if _, ok := errors.AsType[lib.CustomError](txErr); ok && newSub.Id != "" {
			return newSub, txErr
		}
		return domain.Subscription{}, txErr
	}

	return newSub, nil
}

// CancelSubscription cancels a subscription. It will continue through its current billing cycle.
// The orchestration wrapper additionally signals the workflow engine.
// The read-then-write is atomic via SELECT FOR UPDATE — see Activate.
func (s *SubscriptionService) CancelSubscription(ctx context.Context, input domain.CancelSubscriptionInput) (domain.Subscription, error) {
	s.logger.Info("Cancelling subscription", "orgId", input.OrgId, "id", input.Id)

	var subscription domain.Subscription
	txErr := s.transitionInTx(ctx, func(ctx context.Context) error {
		sub, err := s.subscriptionRepository.FindByIdForUpdate(ctx, input.OrgId, input.Id)
		if err != nil {
			if _, ok := errors.AsType[lib.CustomError](err); ok {
				return err
			}
			return lib.NewCustomError(lib.InternalError, "", err)
		}

		if sub.Status == domain.SubscriptionStatusCancelled {
			subscription = sub
			return lib.NewCustomError(lib.BadRequestError, "subscription is already cancelled", nil)
		}

		cancelledAt := time.Now().UTC()
		sub.Status = domain.SubscriptionStatusCancelled
		sub.CancelAt = sub.RenewsAt
		sub.CancelledAt = cancelledAt
		updated, err := s.subscriptionRepository.Update(ctx, sub)
		if err != nil {
			return err
		}
		subscription = updated
		return nil
	})
	if txErr != nil {
		s.logger.Error("CancelSubscription failed", "err", txErr.Error())
		if _, ok := errors.AsType[lib.CustomError](txErr); ok && subscription.Id != "" {
			return subscription, txErr
		}
		return domain.Subscription{}, txErr
	}

	return subscription, nil
}

func (s *SubscriptionService) UpdateBillingAnchor(ctx context.Context, input domain.UpdateBillingAnchorInput) (domain.ProrationDetails, error) {
	s.logger.Infof("Updating billing anchor for subscription %s", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		if _, ok := errors.AsType[lib.CustomError](err); ok {
			return domain.ProrationDetails{}, err
		}
		return domain.ProrationDetails{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	prorationDetails := subscription.UpdateBillingAnchor(input.BillingAnchor, string(input.ProrationMode))

	_, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return domain.ProrationDetails{}, err
	}

	sub, findErr := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if findErr == nil {
		_ = s.pubsub.Publish(sub.OrgId, port.TopicSubscriptionBillingAnchorChanged, sub)
	}

	return prorationDetails, nil
}

func (s *SubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription domain.Subscription) (domain.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, subscription.OrgId, subscription.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return domain.Customer{}, err
	}
	return customer, nil
}

func (s *SubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription domain.Subscription) (domain.PaymentMethod, error) {
	s.logger.Infof("Fetching payment method for subscription [%s] %s - %s",
		subscription.OrgId, subscription.Id, subscription.PaymentMethodId)

	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, subscription.OrgId, subscription.PaymentMethodId)
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return domain.PaymentMethod{}, err
	}
	return paymentMethod, nil
}

func (s *SubscriptionService) FindSubscriptionPayments(ctx context.Context, pk domain.EntityKey, pagination domain.Pagination) ([]domain.Payment, int, error) {
	s.logger.Info("Fetching payment method for subscription", "orgId", pk.OrgId, "id", pk.Id)

	payments, total, err := s.paymentRepository.FindBySubscriptionId(ctx, pk.OrgId, pk.Id, pagination)
	if err != nil {
		s.logger.Error("Failed to find payment method", err.Error())
		return nil, 0, err
	}
	return payments, total, nil
}

func (s *SubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input domain.SubscriptionChargeInput) (domain.Subscription, error) {
	s.logger.Info("Recording subscription payment and updating subscription")
	subscription := input.Subscription
	charge := input.ChargeResult

	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
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
		s.logger.Error("Failed to create payment", err.Error())
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

	s.logger.Infof("[%s][%s] subscription charged, updating with new values [%s]",
		subscription.OrgId, subscription.Id, subscription.Status)

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
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
	s.logger.Info("Charge failure happened",
		"orgId", input.Subscription.OrgId,
		"id", input.Subscription.Id,
		"reason", input.ChargeResult.ErrorReason)

	subscription := input.Subscription
	charge := input.ChargeResult

	s.logger.Infof("Subscription [%s] charge failed with reason [%s][%s][chargeResult status = %s][retries=%d]",
		subscription.Id, charge.ErrorCode, charge.ErrorReason, charge.Status, subscription.Retries)
	if subscription.Id == "" {
		s.logger.Error("Subscription is empty")
		panic("Subscription is empty")
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
		s.logger.Error("Failed to create payment", err.Error())
	}

	s.logger.Debug("Created payment for subscription")

	retryPolicy := s.GetRetryPolicy(ctx, subscription.OrgId)
	s.logger.Debug("Retry policy",
		"attempts", retryPolicy.RetryAttempts,
		"interval", retryPolicy.RetryInterval,
		"qty", retryPolicy.RetryPeriod,
		"action", retryPolicy.FailureAction,
	)

	nextRetryDate := retryPolicy.GetNextCharge(subscription)
	if nextRetryDate.IsZero() {
		s.logger.Debugf("Subscription [%s] has no more retries left", subscription.Id)
		if retryPolicy.FailureAction == domain.FailureActionMarkUnpaid {
			s.logger.Debugf("Marking as unpaid..")
			subscription.Status = domain.SubscriptionStatusUnpaid
		}
		if retryPolicy.FailureAction == domain.FailureActionCancel {
			s.logger.Debugf("Cancelling..")
			subscription.SetCancelled()
		}
	} else {
		s.logger.Debugf("Subscription [%s] next retry date [%s]", subscription.Id, nextRetryDate)
		subscription.Status = domain.SubscriptionStatusPastDue
		subscription.NextRetryAt = nextRetryDate
		subscription.Retries++
	}

	s.logger.Infof("[%s][%s] nextCharge=[%s]", subscription.OrgId, subscription.Id, subscription.GetNextChargeDate())
	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return domain.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionPaymentChargeFailed, map[string]any{
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

// ChargeForBillingPeriod runs one charge attempt against the gateway for the
// subscription's current billing cycle and returns a normalized ChargeResult.
// A non-nil error means the gateway itself failed (e.g. rate limit) and the
// caller should retry; engine adapters translate that into engine-specific
// retryable errors.
func (s *SubscriptionService) ChargeForBillingPeriod(ctx context.Context, currentSub domain.Subscription) (domain.ChargeResult, error) {
	s.logger.Infof("ChargeForBillingPeriod [%s] amount=%d", currentSub.Id, currentSub.Amount)

	subscription, err := s.subscriptionRepository.FindById(ctx, currentSub.OrgId, currentSub.Id)
	if err != nil {
		s.logger.Error("Failed to find subscription", "error", err.Error())
		return domain.ChargeResult{}, err
	}

	gw, err := s.gatewayFactory.NewGateway(ctx, subscription.OrgId, string(subscription.PspId))
	if err != nil {
		s.logger.Error("Failed to get gateway", "err", err.Error())
		return domain.ChargeResult{}, err
	}

	customer, err := s.GetSubscriptionCustomer(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to get customer", "error", err.Error())
		return domain.ChargeResult{}, err
	}

	paymentMethod, err := s.GetSubscriptionPaymentMethod(ctx, subscription)
	if err != nil {
		s.logger.Error("failed to get paymentMethod", "error", err.Error())
		return domain.ChargeResult{}, err
	}

	chargeResult := gw.ChargePayment(ctx, domain.ChargePaymentCommand{
		OrgId:          subscription.OrgId,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Amount:         subscription.Amount,
		Currency:       subscription.Currency,
		PaymentMethod: domain.GatewayPaymentMethod{
			PspId:       paymentMethod.Id,
			Name:        paymentMethod.Name,
			Type:        string(paymentMethod.Type),
			IsRecurring: true,
			Token:       paymentMethod.Token,
		},
		Customer: customer,
	})

	if chargeResult.Status == domain.GatewayError {
		s.logger.Error("Gateway error, charge should be retried", "error", chargeResult.ErrorReason)
		s.errorReporter.ReportError(ctx, errors.New("gateway error while charging subscription"), map[string]any{
			"org_id":          subscription.OrgId,
			"error":           chargeResult.ErrorReason,
			"psp":             string(subscription.PspId),
			"subscription_id": subscription.Id,
		})
		return domain.ChargeResult{}, fmt.Errorf("gateway error: %s", chargeResult.ErrorReason)
	}

	rawData, err := json.Marshal(chargeResult.PspResponse)
	if err != nil {
		s.logger.Error("failed to marshal charge result", "error", err.Error())
	}

	var status domain.PaymentStatus
	var completedAt time.Time
	switch chargeResult.Status {
	case domain.ChargePaymentStatusSuccess:
		status = domain.PaymentStatusSucceeded
		completedAt = time.Now()
	case domain.ChargePaymentStatusPending:
		status = domain.PaymentStatusPending
	case domain.ChargePaymentStatusError:
		status = domain.PaymentStatusFailed
	}

	return domain.ChargeResult{
		Psp:         chargeResult.Psp,
		Amount:      chargeResult.AmountCharged,
		Status:      status,
		Currency:    subscription.Currency,
		ErrorReason: chargeResult.ErrorReason,
		ErrorCode:   chargeResult.ErrorCode,
		PspId:       chargeResult.PspId,
		Reference:   chargeResult.Reference,
		ProcessedAt: completedAt,
		RawData:     string(rawData),
	}, nil
}

// SendRenewalReminder publishes a renewal reminder event for the subscription
// after re-reading it from the repository to ensure fresh state.
func (s *SubscriptionService) SendRenewalReminder(ctx context.Context, orgId string, id string) error {
	s.logger.Info("SendRenewalReminder", "orgId", orgId, "id", id)
	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscription", "error", err.Error())
		return err
	}

	if err := s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionRenewalReminder, subscription); err != nil {
		s.logger.Error("Failed to publish reminder event", "error", err.Error())
		return err
	}
	return nil
}

// MarkAsError transitions the subscription into the error state, recording the
// causing error onto its metadata.
func (s *SubscriptionService) MarkAsError(ctx context.Context, subscription domain.Subscription, cause error) error {
	s.logger.Info("MarkAsError", "orgId", subscription.OrgId, "id", subscription.Id, "err", cause.Error())

	subscription.Status = domain.SubscriptionStatusError
	if subscription.Metadata == nil {
		subscription.Metadata = map[string]string{}
	}
	subscription.Metadata["error"] = cause.Error()

	if _, err := s.subscriptionRepository.Update(ctx, subscription); err != nil {
		s.logger.Error("Failed to update subscription", "error", err.Error())
		return err
	}
	return nil
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
		s.logger.Infof(`Retry policy not set, using default policy`)
		return defaultPolicy
	}

	var retryPolicy domain.RetryPolicy
	err = json.Unmarshal([]byte(setting.Value), &retryPolicy)
	if err != nil {
		s.logger.Error("Failed to unmarshal retry policy", "error", err)
		return defaultPolicy
	}
	return retryPolicy
}
