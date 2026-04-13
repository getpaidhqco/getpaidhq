package service

import (
	"context"
	"errors"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

// OrderService merges the previous OrderService + OrderWorkflowService.
// CompleteCheckoutSession is now a method on this unified service.
type OrderService struct {
	engine                 port.Engine
	sessionRepository      port.SessionRepository
	cartRepository         port.CartRepository
	priceRepository        port.PriceRepository
	orderRepository        port.OrderRepository
	customerRepository     port.CustomerRepository
	subscriptionRepository port.SubscriptionRepository
	paymentMethodRepository port.PaymentMethodRepository
	paymentRepository      port.PaymentRepository
	pubsub                 port.PubSub
	logger                 port.Logger
}

func NewOrderService(
	engine port.Engine,
	sessionRepository port.SessionRepository,
	priceRepository port.PriceRepository,
	cartRepository port.CartRepository,
	orderRepository port.OrderRepository,
	customerRepository port.CustomerRepository,
	subscriptionRepository port.SubscriptionRepository,
	paymentRepository port.PaymentRepository,
	paymentMethodRepository port.PaymentMethodRepository,
	pubsub port.PubSub,
	logger port.Logger,
) *OrderService {
	return &OrderService{
		engine:                  engine,
		customerRepository:      customerRepository,
		paymentMethodRepository: paymentMethodRepository,
		priceRepository:         priceRepository,
		sessionRepository:       sessionRepository,
		cartRepository:          cartRepository,
		subscriptionRepository:  subscriptionRepository,
		orderRepository:         orderRepository,
		logger:                  logger,
		paymentRepository:       paymentRepository,
		pubsub:                  pubsub,
	}
}

func (s *OrderService) FindById(ctx context.Context, orgId string, id string) (domain.Order, error) {
	order, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}

func (s *OrderService) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Order, int, error) {
	orders, total, err := s.orderRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list orders", err.Error())
		return nil, 0, err
	}
	return orders, total, nil
}

func (s *OrderService) ListOrderSubscriptions(ctx context.Context, orgId string, id string) ([]domain.Subscription, error) {
	s.logger.Info("Listing subscriptions for order", "orgId", orgId, "id", id)

	_, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Order not found", "err", err.Error())
		return nil, errors.New("order not found")
	}

	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to retrieve subscriptions", err.Error())
		return nil, err
	}

	return subscriptions, nil
}

// CompleteOrder marks a pending order as completed and activates subscriptions.
// No payment is involved - subscriptions start charging using the specified payment methods.
func (s *OrderService) CompleteOrder(ctx context.Context, input domain.CompleteOrderInput) (domain.Order, error) {
	s.logger.Infof("Completing order [%s][%s]", input.OrgId, input.Id)

	order, err := s.orderRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		return domain.Order{}, errors.New("order not found")
	}

	if order.Status != domain.OrderStatusPending {
		return domain.Order{}, lib.NewCustomError(lib.BadRequestError, "Order is not pending", nil)
	}
	if input.PaymentMethodId == "" && input.PaymentMethod.Token == "" {
		return domain.Order{}, lib.NewCustomError(lib.BadRequestError, "You need to provide payment method or payment method ID", nil)
	}

	order.Status = domain.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	order.SetMetadata(input.Metadata)

	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return domain.Order{}, err
	}

	var paymentMethod domain.PaymentMethod
	if input.PaymentMethodId != "" {
		paymentMethod, err = s.customerRepository.FindPaymentMethodById(ctx, order.OrgId, input.PaymentMethodId)
		if err != nil {
			s.logger.Error("Failed to find payment method", err.Error())
			return domain.Order{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
		}
	}

	if input.PaymentMethod.Token != "" {
		var expireAt time.Time
		if input.PaymentMethod.Details != nil {
			details, err := domain.ParsePaymentMethodDetails(input.PaymentMethod.Type, input.PaymentMethod.Details)
			if err != nil {
				return domain.Order{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
			}
			expireAt = details.GetExpiryDate()
			s.logger.Debugf("This payment method expires at: %v", expireAt)
		}

		paymentMethod, err = s.paymentMethodRepository.Create(ctx, domain.PaymentMethod{
			OrgId:          order.OrgId,
			Id:             lib.GenerateId("pm"),
			Psp:            input.PaymentMethod.Psp,
			Status:         domain.PaymentMethodStatusActive,
			ExpireAt:       expireAt,
			Metadata:       input.PaymentMethod.Metadata,
			Name:           input.PaymentMethod.Name,
			CustomerId:     order.CustomerId,
			BillingAddress: domain.Address{},
			Type:           input.PaymentMethod.Type,
			Token:          input.PaymentMethod.Token,
			Details:        input.PaymentMethod.Details,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		})
		if err != nil {
			s.logger.Error("Failed to create payment method", err.Error())
			return domain.Order{}, err
		}
		s.logger.Debugf(`Created payment method [%s] for order [%s]`, paymentMethod.Id, order.Id)
	}

	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Info("no subscriptions to process", err.Error())
	}

	for _, subscription := range subscriptions {
		s.logger.Debugf("Setting subscription [%s] to active", subscription.Id)

		subscription.PaymentMethodId = paymentMethod.Id
		subscription.SetMetadata(input.Metadata)

		firstPaymentCharged := input.Payment.Amount > 0
		var payment domain.Payment
		if firstPaymentCharged {
			payment = domain.Payment{
				OrgId:          input.OrgId,
				Id:             lib.GenerateId("pmt"),
				Psp:            subscription.PspId,
				Recurring:      true,
				PspId:          input.Payment.PspId,
				Reference:      input.Payment.Reference,
				OrderId:        input.Id,
				SubscriptionId: subscription.Id,
				Status:         domain.PaymentStatusSucceeded,
				Currency:       input.Payment.Currency,
				Amount:         input.Payment.Amount,
				PspFee:         0,
				PlatformFee:    0,
				NetAmount:      input.Payment.Amount,
				Metadata:       input.Payment.Metadata,
				CompletedAt:    input.Payment.CompletedAt,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}
			payment, err = s.paymentRepository.Create(ctx, payment)
			if err != nil {
				s.logger.Error("Failed to create payment", err.Error())
				return domain.Order{}, err
			}
		}

		subscription.SetActive(payment)
		s.logger.Infof("Subscription [%s] activated. firstPaymentCharged=%t", subscription.Id, firstPaymentCharged)
		newSub, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription", "err", err.Error())
			return domain.Order{}, err
		}

		s.logger.Debugf("Starting subscription workflow")
		err = s.engine.StartSubscriptionWorkflow(ctx, newSub)
		if err != nil {
			s.logger.Errorf("Failed to start workflow %v", err.Error())
			return domain.Order{}, err
		}
	}

	_ = s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order)
	return order, nil
}

// CompleteCheckoutSession marks a pending order as completed via a payment webhook.
// This handles the PSP-triggered flow (Paystack/Checkout.com webhook → order completion).
func (s *OrderService) CompleteCheckoutSession(ctx context.Context, input domain.CompleteCheckoutSessionInput) (domain.Order, error) {
	s.logger.Info("Completing order via checkout session", "order_id", input.OrderId)
	orgId := input.OrgId
	orderId := input.OrderId

	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		return domain.Order{}, errors.New("order not found")
	}

	order.Status = domain.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return domain.Order{}, err
	}

	// The payment context comes from the webhook - extract what we need
	paymentCtx, ok := input.PaymentContext.(port.PaymentWebhookContext)
	if !ok {
		// Try parsing from interface
		var parseErr error
		paymentCtx, parseErr = port.ParsePaymentWebhookContext(input.PaymentContext)
		if parseErr != nil {
			s.logger.Error("Failed to parse payment context", parseErr.Error())
			return domain.Order{}, parseErr
		}
	}

	paymentMethod, err := s.paymentMethodRepository.Create(ctx, domain.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("payment_method"),
		Psp:            string(paymentCtx.Psp),
		Token:          paymentCtx.PaymentMethod.Token,
		Name:           "Default",
		CustomerId:     order.CustomerId,
		BillingAddress: order.Customer.BillingAddress,
		Type:           domain.PaymentMethodType(paymentCtx.PaymentMethod.Type),
		Details:        paymentCtx.PaymentMethod,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create payment method", err.Error())
		return domain.Order{}, err
	}
	s.logger.Infof("Created payment method %s for order %s", paymentMethod.Id, order.Id)

	var subscriptionId string
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("no subscriptions", err.Error())
	}

	recurringPayment := len(subscriptions) > 0 && paymentCtx.Payment.Amount > 0
	for _, subscription := range subscriptions {
		charged := paymentCtx.Payment.Amount > 0 && subscription.StartDate.Sub(time.Now().UTC()) < 0
		if charged {
			subscriptionId = subscription.Id
			subscription.SetActivationDates()
			subscription.Status = domain.SubscriptionStatusActive
			subscription.LastCharge = subscription.StartDate
			subscription.TotalRevenue = subscription.Amount
			subscription.CyclesProcessed = 1
		} else {
			subscription.SetActivationDates()
			subscription.Status = domain.SubscriptionStatusTrial
		}
		subscription.PaymentMethodId = paymentMethod.Id

		_, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription status", err.Error())
			return domain.Order{}, err
		}
	}

	if paymentCtx.Payment.Amount > 0 {
		payment := domain.Payment{
			OrgId:          orgId,
			Id:             lib.GenerateId("pmt"),
			Psp:            paymentCtx.Psp,
			PspId:          paymentCtx.Payment.PspId,
			Reference:      paymentCtx.Payment.Reference,
			OrderId:        orderId,
			SubscriptionId: subscriptionId,
			Status:         domain.PaymentStatusSucceeded,
			Recurring:      recurringPayment,
			Currency:       paymentCtx.Payment.Currency,
			Amount:         paymentCtx.Payment.Amount,
			PspFee:         0,
			PlatformFee:    0,
			NetAmount:      paymentCtx.Payment.Amount,
			Metadata:       nil,
			CompletedAt:    paymentCtx.Payment.PaidAt,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		_, err := s.paymentRepository.Create(ctx, payment)
		if err != nil {
			s.logger.Error("Failed to create payment", err.Error())
		}
	}

	_ = s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order)
	return order, nil
}
