package service

import (
	"context"
	"errors"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"getpaidhq/internal/lib/ids"
	"time"
)

// OrderWorkflowService handles webhook-driven order completion. It does NOT
// hold the workflow engine: this method is invoked from a workflow step, and
// the step is registered with the very engine that dispatches it — so depending
// on the engine here would create a construction-time cycle.
//
// HTTP-driven order completion (which DOES start subscription workflows) lives
// on OrderService.
type OrderWorkflowService struct {
	orderRepository         port.OrderRepository
	customerRepository      port.CustomerRepository
	subscriptionRepository  port.SubscriptionRepository
	paymentMethodRepository port.PaymentMethodRepository
	paymentRepository       port.PaymentRepository
	priceRepository         port.PriceRepository
	tx                      port.TxManager
	pubsub                  port.PubSub
	logger                  port.Logger
	invoiceService          OrderInvoicing
	coupons                 OrderCoupons
}

func NewOrderWorkflowService(
	orderRepository port.OrderRepository,
	customerRepository port.CustomerRepository,
	subscriptionRepository port.SubscriptionRepository,
	paymentMethodRepository port.PaymentMethodRepository,
	paymentRepository port.PaymentRepository,
	priceRepository port.PriceRepository,
	tx port.TxManager,
	pubsub port.PubSub,
	logger port.Logger,
	invoiceService OrderInvoicing,
	coupons OrderCoupons,
) *OrderWorkflowService {
	return &OrderWorkflowService{
		orderRepository:         orderRepository,
		customerRepository:      customerRepository,
		subscriptionRepository:  subscriptionRepository,
		paymentMethodRepository: paymentMethodRepository,
		paymentRepository:       paymentRepository,
		priceRepository:         priceRepository,
		tx:                      tx,
		pubsub:                  pubsub,
		logger:                  logger,
		invoiceService:          invoiceService,
		coupons:                 coupons,
	}
}

// CompleteCheckoutSession marks a pending order as completed via a payment webhook.
// This handles the PSP-triggered flow (Paystack/Checkout.com webhook -> order completion).
func (s *OrderWorkflowService) CompleteCheckoutSession(ctx context.Context, input port.CompleteCheckoutSessionInput) (domain.Order, error) {
	s.logger.Info("Completing order via checkout session", "order_id", input.OrderId)
	orgId := input.OrgId
	orderId := input.OrderId

	paymentCtx := input.PaymentContext
	var order domain.Order
	var shouldPublish bool

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		var err error
		order, err = s.orderRepository.FindByIdForUpdate(ctx, orgId, orderId)
		if err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return errors.Join(errors.New("order not found"), port.ErrNotFound)
			}
			return err
		}

		if order.Status == domain.OrderStatusCompleted {
			return nil
		}
		if order.Status != domain.OrderStatusPending {
			return lib.NewCustomError(lib.ConflictError, "Order is not pending", nil)
		}

		customer, err := s.customerRepository.FindById(ctx, orgId, order.CustomerId)
		if err != nil {
			s.logger.Error("Failed to find customer for order", "customer_id", order.CustomerId, "err", err.Error())
			return err
		}

		// Details keeps the PSP's display data but NOT the token — Details is
		// echoed in API responses and events, and the token already lives in the
		// dedicated (redacting) Token field.
		details := paymentCtx.PaymentMethod
		details.Token = ""
		paymentMethod, err := s.paymentMethodRepository.Create(ctx, domain.PaymentMethod{
			OrgId:          orgId,
			Id:             ids.Generate("payment_method"),
			Psp:            string(paymentCtx.Psp),
			Status:         domain.PaymentMethodStatusActive,
			Token:          domain.Secret(paymentCtx.PaymentMethod.Token),
			Name:           "Default",
			CustomerId:     order.CustomerId,
			BillingAddress: customer.BillingAddress,
			Type:           domain.PaymentMethodType(paymentCtx.PaymentMethod.Type),
			Details:        details,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		})
		if err != nil {
			s.logger.Error("Failed to create payment method", err.Error())
			return err
		}
		s.logger.Infof("Created payment method %s for order %s", paymentMethod.Id, order.Id)

		order.Status = domain.OrderStatusCompleted
		order.UpdatedAt = time.Now()
		order, err = s.orderRepository.Update(ctx, order)
		if err != nil {
			s.logger.Error("Failed to update order", err.Error())
			return err
		}

		var subscriptionId string
		subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
		if err != nil {
			s.logger.Error("error finding subscriptions", err.Error())
			return err
		}

		recurringPayment := len(subscriptions) > 0 && paymentCtx.Payment.Amount > 0
		for _, subscription := range subscriptions {
			// The subscription carries its own cadence + trial (derived from its lines),
			// so activation needs no price. Revenue is the recurring fixed base for the
			// first cycle (the subscription stores no amount, ADR 0002).
			charged := paymentCtx.Payment.Amount > 0 && subscription.StartDate.Sub(time.Now().UTC()) < 0
			if charged {
				fixedBase, err := fixedBaseAmount(ctx, s.orderRepository, s.priceRepository, orgId, subscription.Id)
				if err != nil {
					s.logger.Error("Failed to resolve subscription base", "subscription_id", subscription.Id, "err", err.Error())
					return err
				}
				subscriptionId = subscription.Id
				subscription.SetActivationDates()
				subscription.Status = domain.SubscriptionStatusActive
				subscription.LastCharge = subscription.StartDate
				subscription.TotalRevenue = fixedBase
				subscription.CyclesProcessed = 1
			} else {
				subscription.SetActivationDates()
				subscription.Status = domain.SubscriptionStatusTrial
			}
			subscription.PaymentMethodId = paymentMethod.Id

			_, err = s.subscriptionRepository.Update(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to update subscription status", err.Error())
				return err
			}
		}

		// Convert the order's coupon reservation (if any) into an active Discount,
		// for ALL orders — subscription and pure one-time alike. A subscription
		// order anchors the discount on its (single) activated sub; a one-time
		// order owns the discount via OrderId only. No-op when the order has no
		// reservation. Consumed BEFORE BuildForOrder so the committed Discount is
		// visible to BuildForOrder's ActiveForOrder.
		consume := ConsumeInput{
			OrgId:      orgId,
			OrderId:    orderId,
			StartCycle: 0,
		}
		if subscriptionId != "" {
			consume.SubscriptionId = subscriptionId
		}
		if _, err := s.coupons.Consume(ctx, consume); err != nil {
			return err
		}

		// Build the ONE combined cycle-0 invoice for the order (sub first-period
		// line(s) + every one-time line, with the order discount applied). Returns
		// the already-built invoice when upfront_invoice opened it at create time
		// (idempotent on the order). An order with no billable items returns
		// port.ErrNotFound → no invoice, no payment-link, no settlement.
		var invoiceId string
		inv, berr := s.invoiceService.BuildForOrder(ctx, order)
		if berr != nil {
			if !errors.Is(berr, port.ErrNotFound) {
				s.logger.Error("Failed to build order invoice", berr.Error())
				return berr
			}
			// nothing to invoice — leave invoiceId empty
		} else {
			invoiceId = inv.Id
		}

		if paymentCtx.Payment.Amount > 0 {
			payment := domain.Payment{
				OrgId:          orgId,
				Id:             ids.Generate("pmt"),
				Psp:            paymentCtx.Psp,
				PspId:          paymentCtx.Payment.PspId,
				Reference:      paymentCtx.Payment.Reference,
				OrderId:        orderId,
				SubscriptionId: subscriptionId,
				InvoiceId:      invoiceId,
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
				return err
			}

			// The combined invoice is paid by this first charge — open then settle it
			// (no-op when there was nothing to invoice; idempotent from open when
			// upfront_invoice already opened it).
			if err := s.invoiceService.SettleOrderInvoice(ctx, orgId, invoiceId); err != nil {
				return err
			}
		}

		shouldPublish = true
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}

	if shouldPublish {
		if err := s.pubsub.Publish(ctx, order.OrgId, port.TopicOrderCompleted, order); err != nil {
			s.logger.Errorf("Failed to publish %s for order %s: %v", port.TopicOrderCompleted, order.Id, err)
		}
	}

	return order, nil
}
