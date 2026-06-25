package service

import (
	"context"
	"errors"
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"

	"golang.org/x/sync/errgroup"
)

// OrderService owns engine-aware order operations: creating orders,
// completing orders from the HTTP flow (which starts subscription workflows),
// and read-side queries.
//
// The webhook-driven CompleteCheckoutSession path lives on
// OrderWorkflowService — that path is invoked from a workflow step, so
// it cannot depend on the engine that dispatches it. Splitting the two
// avoids a construction-time cycle.
type OrderService struct {
	tx                      port.TxManager
	engine                  port.Engine
	sessionRepository       port.SessionRepository
	cartRepository          port.CartRepository
	priceRepository         port.PriceRepository
	orderRepository         port.OrderRepository
	customerRepository      port.CustomerRepository
	subscriptionRepository  port.SubscriptionRepository
	paymentMethodRepository port.PaymentMethodRepository
	paymentRepository       port.PaymentRepository
	productRepository       port.ProductRepository
	gatewayFactory          port.GatewayFactory
	pubsub                  port.PubSub
	logger                  port.Logger
	coupons                 *CouponService
	invoiceService          *InvoiceService
}

func NewOrderService(
	tx port.TxManager,
	engine port.Engine,
	sessionRepository port.SessionRepository,
	priceRepository port.PriceRepository,
	cartRepository port.CartRepository,
	orderRepository port.OrderRepository,
	customerRepository port.CustomerRepository,
	subscriptionRepository port.SubscriptionRepository,
	paymentRepository port.PaymentRepository,
	paymentMethodRepository port.PaymentMethodRepository,
	productRepository port.ProductRepository,
	gatewayFactory port.GatewayFactory,
	pubsub port.PubSub,
	logger port.Logger,
	coupons *CouponService,
	invoiceService *InvoiceService,
) *OrderService {
	return &OrderService{
		tx:                      tx,
		engine:                  engine,
		customerRepository:      customerRepository,
		paymentMethodRepository: paymentMethodRepository,
		priceRepository:         priceRepository,
		sessionRepository:       sessionRepository,
		cartRepository:          cartRepository,
		subscriptionRepository:  subscriptionRepository,
		orderRepository:         orderRepository,
		productRepository:       productRepository,
		gatewayFactory:          gatewayFactory,
		logger:                  logger,
		paymentRepository:       paymentRepository,
		pubsub:                  pubsub,
		coupons:                 coupons,
		invoiceService:          invoiceService,
	}
}

// CreateOrder creates a new order from a session/cart or direct cart items.
func (s *OrderService) CreateOrder(ctx context.Context, input port.CreateOrderInput) (port.CreateOrderResult, error) {
	s.logger.Info("Creating order for cart", "session", input.SessionId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")
	var customerEntity domain.Customer
	var err error
	var orderCart domain.Cart
	currency := input.Currency

	// check if the cart exists
	if input.SessionId != "" {
		session, err := s.sessionRepository.FindById(ctx, orgId, input.SessionId)
		if err != nil {
			s.logger.Error("Failed to find session id ", "id", input.SessionId, err.Error())
			return port.CreateOrderResult{}, lib.NewCustomError(lib.NotFoundError, "session not found", nil)
		}

		existingCart, err := s.cartRepository.FindById(ctx, orgId, session.CartId)
		if err != nil {
			s.logger.Error("Failed to find cart id ", "id", input.SessionId, err.Error())
			return port.CreateOrderResult{}, lib.NewCustomError(lib.NotFoundError, "cart not found", nil)
		}
		orderCart = existingCart
		currency = orderCart.Data.Currency
	} else {
		// Create a cart from the items in the input
		orderCart = domain.Cart{
			OrgId: orgId,
			Id:    lib.GenerateId("cart"),
			Data: domain.CartData{
				Currency: currency,
			},
		}

		for _, item := range input.CartItems {
			product, err := s.productRepository.FindById(ctx, orgId, item.ProductId)
			if err != nil {
				s.logger.Error("Failed to find product", err.Error())
				return port.CreateOrderResult{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}

			price, err := s.priceRepository.FindById(ctx, orgId, item.PriceId)
			if err != nil {
				s.logger.Error("Failed to find price", err.Error())
				return port.CreateOrderResult{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}

			orderCart.Data.Items = append(orderCart.Data.Items, domain.CartLineItem{
				Id:            lib.GenerateId("ci"),
				ProductId:     product.Id,
				Price:         domain.PriceToCartItemPrice(price),
				Description:   product.Name,
				Quantity:      int64(item.Quantity),
				UnitPrice:     price.UnitPrice,
				UnitCount:     int64(price.UnitCount),
				SubTotal:      domain.FixedLineAmount(price.UnitPrice, int64(price.UnitCount), int64(item.Quantity)),
				DiscountTotal: 0,
				TaxTotal:      0,
				ShippingTotal: 0,
				Total:         domain.FixedLineAmount(price.UnitPrice, int64(price.UnitCount), int64(item.Quantity)),
			})
		}
		orderCart.Calculate()

		_, err := s.cartRepository.Create(ctx, domain.Cart{
			OrgId: input.OrgId,
			Id:    orderCart.Id,
			Data:  orderCart.Data,
			Total: orderCart.Total,
		})
		if err != nil {
			s.logger.Error("failed to create cart", err)
			return port.CreateOrderResult{}, err
		}
	}

	// validate that the cart is not empty
	if len(orderCart.Data.Items) == 0 {
		return port.CreateOrderResult{}, errors.New("cart is empty")
	}

	// Archived products are retired and not sellable. Guard here so the rule holds
	// for both order paths (existing session cart and direct cart-items) — and
	// therefore for subscriptions, which are only created via the order flow. This
	// runs only at checkout; recurring renewals never call CreateOrder.
	for _, item := range orderCart.Data.Items {
		product, err := s.productRepository.FindById(ctx, orgId, item.ProductId)
		if err != nil {
			s.logger.Error("Failed to find product", err.Error())
			return port.CreateOrderResult{}, lib.NewCustomError(lib.NotFoundError, "Product not found", err)
		}
		if product.IsArchived() {
			return port.CreateOrderResult{}, lib.NewCustomError(
				lib.ConflictError,
				fmt.Sprintf("Product %s is archived and cannot be sold", product.Id),
				nil,
			)
		}
	}

	// Get or create the customer
	if input.Customer.Id != "" {
		customerEntity, err = s.customerRepository.FindById(ctx, orgId, input.Customer.Id)
		if err != nil {
			s.logger.Error("Failed to find customer", err.Error())
			return port.CreateOrderResult{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
		}
	} else {
		customerEntity, err = s.customerRepository.Create(ctx, domain.Customer{
			OrgId:     orgId,
			Id:        lib.GenerateId("customer"),
			FirstName: input.Customer.FirstName,
			LastName:  input.Customer.LastName,
			Phone:     input.Customer.Phone,
			Email:     input.Customer.Email,
		})
		if err != nil {
			s.logger.Error("Failed to create customer", err.Error())
			return port.CreateOrderResult{}, err
		}
	}

	ref := time.Now().Format("20060102150405")
	order, err := s.orderRepository.Create(ctx, domain.Order{
		OrgId:      orgId,
		Id:         orderId,
		Reference:  ref,
		CustomerId: customerEntity.Id,
		Status:     domain.OrderStatusPending,
		SessionId:  input.SessionId,
		CartId:     orderCart.Id,
		Currency:   currency,
		Total:      orderCart.Total,
		Metadata:   input.Metadata,
		Config:     input.Config,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create order", err.Error())
		return port.CreateOrderResult{}, err
	}

	var lines []orderLine
	for _, item := range orderCart.Data.Items {
		orderItem, err := s.orderRepository.CreateOrderItem(ctx, domain.OrderItem{
			OrgId:         orgId,
			Id:            lib.GenerateId("order_item"),
			OrderId:       orderId,
			ProductId:     item.ProductId,
			VariantId:     item.Price.VariantId,
			PriceId:       item.Price.Id,
			Description:   item.Description,
			Quantity:      int(item.Quantity),
			TaxTotal:      item.TaxTotal,
			DiscountTotal: item.DiscountTotal,
			Subtotal:      item.SubTotal,
			Total:         item.Total,
			Metadata:      nil,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		})
		if err != nil {
			s.logger.Error("Failed to create order item", "item", item, "err", err.Error())
			return port.CreateOrderResult{}, err
		}

		price, err := s.priceRepository.FindById(ctx, orgId, item.Price.Id)
		if err != nil {
			s.logger.Error("Failed to find price for order item", "price_id", item.Price.Id, "err", err.Error())
			return port.CreateOrderResult{}, err
		}
		lines = append(lines, orderLine{item: orderItem, price: price})
	}

	// Group the order's recurring lines by cadence; each group becomes one
	// subscription that owns (bills) all its lines. The subscription stores no
	// charge amount (ADR 0002); the per-cycle total is computed onto the invoice.
	for _, group := range groupIntoSubscriptions(lines) {
		prices := make([]domain.Price, len(group))
		for i, l := range group {
			prices[i] = l.price
		}
		sub := domain.NewSubscriptionFromLines(orgId, orderId, customerEntity.Id, prices)
		sub.PspId = input.PspId
		sub.PaymentMethodId = input.PaymentMethodId
		created, err := s.subscriptionRepository.Create(ctx, sub)
		if err != nil {
			s.logger.Error("Failed to create subscription", "err", err.Error())
			return port.CreateOrderResult{}, err
		}
		for _, l := range group {
			item := l.item
			item.SubscriptionId = created.Id
			if _, err := s.orderRepository.UpdateOrderItem(ctx, item); err != nil {
				s.logger.Error("Failed to link order item to subscription", "order_item", item.Id, "err", err.Error())
				return port.CreateOrderResult{}, err
			}
		}
		_ = s.pubsub.Publish(orgId, port.TopicSubscriptionCreated, created)
	}

	// Reserve the coupon's capacity for this order. A refusal (exhausted code,
	// minimum not met, etc.) is returned as a typed ApiError and fails the order.
	if input.CouponCode != "" && s.coupons != nil {
		if _, err := s.coupons.Reserve(ctx, ReserveInput{
			OrgId:      orgId,
			Code:       input.CouponCode,
			CustomerId: customerEntity.Id,
			OrderId:    orderId,
			Currency:   currency,
			Amount:     order.Total, // cart subtotal for MinimumAmount
		}); err != nil {
			return port.CreateOrderResult{}, err
		}
	}

	newOrder, _ := s.orderRepository.FindById(ctx, orgId, order.Id)

	result := port.CreateOrderResult{Order: newOrder}

	// When the order opts into upfront invoicing, build the combined cycle-0
	// invoice now and open it. The order's items and subscriptions are already
	// committed above, so BuildForOrder (which queries them by order id) sees a
	// complete order. BuildForOrder is idempotent on the order and opens its own
	// tx; an empty order (nothing to invoice) returns port.ErrNotFound, which we
	// treat as "no invoice".
	if input.Config.UpfrontInvoice && s.invoiceService != nil {
		inv, err := s.invoiceService.BuildForOrder(ctx, newOrder)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return port.CreateOrderResult{}, err
		}
		if err == nil {
			opened, oerr := s.invoiceService.MarkOpen(ctx, newOrder.OrgId, inv.Id)
			if oerr != nil {
				return port.CreateOrderResult{}, oerr
			}
			result.Invoice = &opened
		}
	}

	return result, nil
}

// InitOrderPayment initialises (or returns) the PSP payment session for an
// existing pending order. Idempotent on the stored session: a repeat call, or a
// retry after a gateway failure, returns the same session — never a duplicate.
// pspId selects the gateway (the order does not persist it); it is ignored when
// a session already exists.
func (s *OrderService) InitOrderPayment(ctx context.Context, orgId, orderId, pspId string, opts map[string]any) (port.InitPaymentResponse, error) {
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find order", "id", orderId, "err", err.Error())
		return port.InitPaymentResponse{}, lib.NewCustomError(lib.NotFoundError, "order not found", err)
	}

	if order.Status != domain.OrderStatusPending {
		return port.InitPaymentResponse{}, lib.NewCustomError(lib.ConflictError, "order is not pending", nil)
	}

	// Already initialised — return the stored session without touching the gateway.
	if order.PaymentSession != nil {
		return port.InitPaymentResponse{PspResponse: order.PaymentSession}, nil
	}

	var (
		cart     domain.Cart
		customer domain.Customer
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var ferr error
		cart, ferr = s.cartRepository.FindById(gctx, orgId, order.CartId)
		if ferr != nil {
			s.logger.Error("Failed to find cart", "id", order.CartId, "err", ferr.Error())
			return lib.NewCustomError(lib.NotFoundError, "cart not found", ferr)
		}
		return nil
	})
	g.Go(func() error {
		var ferr error
		customer, ferr = s.customerRepository.FindById(gctx, orgId, order.CustomerId)
		if ferr != nil {
			s.logger.Error("Failed to find customer", "id", order.CustomerId, "err", ferr.Error())
			return lib.NewCustomError(lib.NotFoundError, "customer not found", ferr)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return port.InitPaymentResponse{}, err
	}

	gw, err := s.gatewayFactory.NewGateway(ctx, orgId, pspId)
	if err != nil {
		s.logger.Error("Failed to get gateway", "err", err.Error())
		return port.InitPaymentResponse{}, err
	}

	resp, err := gw.InitPayment(ctx, port.InitPaymentInput{
		OrgId:    orgId,
		Cart:     cart,
		Order:    order,
		Customer: customer,
		Options:  toStringMap(opts),
	})
	if err != nil {
		s.logger.Error("Failed to initialise payment gateway", err.Error())
		return port.InitPaymentResponse{}, err
	}

	if err := s.orderRepository.SetPaymentSession(ctx, orgId, orderId, resp.PspResponse); err != nil {
		s.logger.Error("Failed to persist payment session", "id", orderId, "err", err.Error())
		return port.InitPaymentResponse{}, err
	}

	return resp, nil
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

// CompleteOrder marks a pending order completed: it activates every
// subscription on the order, consumes the order's coupon reservation (if any)
// into an order-owned Discount, builds the ONE combined cycle-0 invoice
// (subscription first-period line(s) + every one-time line), and — when the
// caller charged a first payment — persists a single order Payment linked to
// that invoice and settles the invoice to paid.
//
// All DB writes (order update, payment method find/create, subscription
// activation, coupon consume, invoice build+settle, payment) run inside a single
// transaction; the nested RunInTx of Consume/BuildForOrder join it (ambient-tx
// reuse). Post-commit side effects (subscription workflow starts and the
// order.completed pubsub event) fire only after the tx commits — running them
// inside would orphan workflows and pubsub messages on rollback.
func (s *OrderService) CompleteOrder(ctx context.Context, input port.CompleteOrderInput) (domain.Order, error) {
	s.logger.Infof("Completing order [%s][%s]", input.OrgId, input.Id)

	var order domain.Order
	var activated []domain.Subscription

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		var err error
		order, err = s.orderRepository.FindById(ctx, input.OrgId, input.Id)
		if err != nil {
			return errors.New("order not found")
		}

		if order.Status != domain.OrderStatusPending {
			return lib.NewCustomError(lib.BadRequestError, "Order is not pending", nil)
		}
		if input.PaymentMethodId == "" && input.PaymentMethod.Token == "" {
			return lib.NewCustomError(lib.BadRequestError, "You need to provide payment method or payment method ID", nil)
		}

		order.Status = domain.OrderStatusCompleted
		order.UpdatedAt = time.Now()
		order.SetMetadata(input.Metadata)

		if _, err = s.orderRepository.Update(ctx, order); err != nil {
			s.logger.Error("Failed to update order", err.Error())
			return err
		}

		var paymentMethod domain.PaymentMethod
		if input.PaymentMethodId != "" {
			paymentMethod, err = s.customerRepository.FindPaymentMethodById(ctx, order.OrgId, input.PaymentMethodId)
			if err != nil {
				s.logger.Error("Failed to find payment method", err.Error())
				return lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
			}
		}

		if input.PaymentMethod.Token != "" {
			var expireAt time.Time
			if input.PaymentMethod.Details != nil {
				details, err := domain.ParsePaymentMethodDetails(input.PaymentMethod.Type, input.PaymentMethod.Details)
				if err != nil {
					return lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
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
				Token:          domain.Secret(input.PaymentMethod.Token),
				Details:        input.PaymentMethod.Details,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			})
			if err != nil {
				s.logger.Error("Failed to create payment method", err.Error())
				return err
			}
			s.logger.Debugf(`Created payment method [%s] for order [%s]`, paymentMethod.Id, order.Id)
		}

		subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, input.OrgId, input.Id)
		if err != nil {
			s.logger.Info("no subscriptions to process", err.Error())
		}

		// A first charge happened iff the caller passed a positive amount. We build
		// the Payment value in memory now (no InvoiceId yet) so SetActive can use
		// its Amount/CompletedAt to advance each subscription's first period. The
		// single order Payment is persisted later, after BuildForOrder, linked to
		// the combined invoice.
		firstPaymentCharged := input.Payment.Amount > 0
		var payment domain.Payment
		if firstPaymentCharged {
			payment = domain.Payment{
				OrgId:       input.OrgId,
				Recurring:   len(subscriptions) > 0,
				Status:      domain.PaymentStatusSucceeded,
				Currency:    input.Payment.Currency,
				Amount:      input.Payment.Amount,
				NetAmount:   input.Payment.Amount,
				CompletedAt: input.Payment.CompletedAt,
			}
		}

		activated = make([]domain.Subscription, 0, len(subscriptions))
		for _, subscription := range subscriptions {
			s.logger.Debugf("Setting subscription [%s] to active", subscription.Id)

			subscription.PaymentMethodId = paymentMethod.Id
			subscription.SetMetadata(input.Metadata)

			// The subscription carries its own cadence + trial (derived from its
			// lines at construction), so activation needs no price lookup. SetActive
			// reads payment.Amount/CompletedAt to advance the first period.
			subscription.SetActive(payment)
			s.logger.Infof("Subscription [%s] activated. firstPaymentCharged=%t", subscription.Id, firstPaymentCharged)
			newSub, err := s.subscriptionRepository.Update(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to update subscription", "err", err.Error())
				return err
			}
			activated = append(activated, newSub)
		}

		// Convert the order's coupon reservation (if any) into an active Discount,
		// for ALL orders — subscription and pure one-time alike. A subscription
		// order anchors the discount on its (single) sub; a one-time order owns the
		// discount via OrderId only. No-op when the order has no reservation, so
		// coupon-less orders are unaffected. Consumed BEFORE BuildForOrder so the
		// committed Discount is visible to BuildForOrder's ActiveForOrder.
		if s.coupons != nil {
			consume := ConsumeInput{
				OrgId:      input.OrgId,
				OrderId:    order.Id,
				StartCycle: 0,
			}
			if len(activated) > 0 {
				consume.SubscriptionId = activated[0].Id
			}
			if _, err := s.coupons.Consume(ctx, consume); err != nil {
				return err
			}
		}

		// Build the one combined cycle-0 invoice for the order (sub first-period
		// line(s) + every one-time line, with the order discount applied). Returns
		// the already-built invoice when upfront_invoice opened it at create time.
		// An order with no billable items returns port.ErrNotFound → no invoice,
		// no payment-link, no settlement. Guarded so unit-test harnesses that pass
		// a nil invoiceService behave as before.
		if s.invoiceService == nil {
			return nil
		}
		inv, err := s.invoiceService.BuildForOrder(ctx, order)
		if err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return nil // nothing to invoice
			}
			s.logger.Error("Failed to build order invoice", err.Error())
			return err
		}

		if firstPaymentCharged {
			// Persist ONE Payment for the order, linked to the combined invoice.
			// A single-subscription order links the payment to that sub; a
			// multi-sub or pure one-time order leaves SubscriptionId empty (the
			// combined invoice itself carries no single SubscriptionId in those
			// cases).
			payment.Id = lib.GenerateId("pmt")
			payment.PspId = input.Payment.PspId
			payment.Reference = input.Payment.Reference
			payment.Metadata = input.Payment.Metadata
			payment.OrderId = order.Id
			payment.InvoiceId = inv.Id
			payment.CreatedAt = time.Now().UTC()
			payment.UpdatedAt = time.Now().UTC()
			if len(activated) == 1 {
				payment.SubscriptionId = activated[0].Id
				payment.Psp = activated[0].PspId
			}
			if _, err := s.paymentRepository.Create(ctx, payment); err != nil {
				s.logger.Error("Failed to create payment", err.Error())
				return err
			}

			// Open then settle the invoice — it is paid by this first charge.
			if err := s.invoiceService.SettleOrderInvoice(ctx, order.OrgId, inv.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}

	// Post-commit side effects. Failures here are logged, not returned —
	// the order is already committed, so a non-2xx response would mislead
	// the caller into thinking the action didn't happen.
	for _, sub := range activated {
		if startErr := s.engine.StartSubscriptionWorkflow(ctx, sub); startErr != nil {
			s.logger.Errorf("Failed to start subscription workflow for %s: %v", sub.Id, startErr)
		}
	}
	if pubErr := s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order); pubErr != nil {
		s.logger.Errorf("Failed to publish %s for order %s: %v", port.TopicOrderCompleted, order.Id, pubErr)
	}
	return order, nil
}

// GetDetails composes an OrderDetails read model: order + customer + items
// (each item paired with its Price).
func (s *OrderService) GetDetails(ctx context.Context, orgId, id string) (OrderDetails, error) {
	order, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		return OrderDetails{}, err
	}
	customer, err := s.customerRepository.FindById(ctx, orgId, order.CustomerId)
	if err != nil {
		return OrderDetails{}, err
	}
	items, err := s.orderRepository.FindOrderItemsByOrderId(ctx, orgId, order.Id)
	if err != nil {
		return OrderDetails{}, err
	}
	priceIds := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, it := range items {
		if !seen[it.PriceId] {
			seen[it.PriceId] = true
			priceIds = append(priceIds, it.PriceId)
		}
	}
	prices, err := s.priceRepository.FindByIds(ctx, orgId, priceIds)
	if err != nil {
		return OrderDetails{}, err
	}
	priceById := make(map[string]domain.Price, len(prices))
	for _, p := range prices {
		priceById[p.Id] = p
	}
	itemDetails := make([]OrderItemDetails, len(items))
	for i, it := range items {
		itemDetails[i] = OrderItemDetails{Item: it, Price: priceById[it.PriceId]}
	}
	return OrderDetails{Order: order, Customer: customer, Items: itemDetails}, nil
}

// ListDetails returns orders with their composed details. Customers and prices
// are batch-loaded to avoid N+1.
func (s *OrderService) ListDetails(ctx context.Context, orgId string, pagination domain.Pagination) ([]OrderDetails, int, error) {
	orders, total, err := s.orderRepository.Find(ctx, orgId, pagination)
	if err != nil {
		return nil, 0, err
	}
	if len(orders) == 0 {
		return []OrderDetails{}, total, nil
	}
	customerIds := make([]string, 0, len(orders))
	cSeen := make(map[string]bool)
	for _, o := range orders {
		if !cSeen[o.CustomerId] {
			cSeen[o.CustomerId] = true
			customerIds = append(customerIds, o.CustomerId)
		}
	}
	customers, err := s.customerRepository.FindByIds(ctx, orgId, customerIds)
	if err != nil {
		return nil, 0, err
	}
	customerById := make(map[string]domain.Customer, len(customers))
	for _, c := range customers {
		customerById[c.Id] = c
	}
	out := make([]OrderDetails, len(orders))
	for i, o := range orders {
		items, err := s.orderRepository.FindOrderItemsByOrderId(ctx, orgId, o.Id)
		if err != nil {
			return nil, 0, err
		}
		priceIds := make([]string, 0, len(items))
		pSeen := make(map[string]bool)
		for _, it := range items {
			if !pSeen[it.PriceId] {
				pSeen[it.PriceId] = true
				priceIds = append(priceIds, it.PriceId)
			}
		}
		prices, err := s.priceRepository.FindByIds(ctx, orgId, priceIds)
		if err != nil {
			return nil, 0, err
		}
		priceById := make(map[string]domain.Price, len(prices))
		for _, p := range prices {
			priceById[p.Id] = p
		}
		itemDetails := make([]OrderItemDetails, len(items))
		for j, it := range items {
			itemDetails[j] = OrderItemDetails{Item: it, Price: priceById[it.PriceId]}
		}
		out[i] = OrderDetails{Order: o, Customer: customerById[o.CustomerId], Items: itemDetails}
	}
	return out, total, nil
}

// toStringMap flattens the InitOrderPayment opts (map[string]any) into the
// map[string]string the gateway InitPayment input expects. Non-string values
// are rendered with fmt; a nil map yields nil.
func toStringMap(in map[string]any) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// orderLine pairs a persisted order item with its resolved price.
type orderLine struct {
	item  domain.OrderItem
	price domain.Price
}

// fixedBaseAmount sums a subscription's recurring flat fee from its own lines —
// the fixed (non-metered) unit prices × quantity. Metered lines contribute zero
// (ADR 0002). Used for proration credit and first-cycle revenue; the subscription
// stores no amount, so this is derived on demand from the current line prices.
func fixedBaseAmount(ctx context.Context, orderRepo port.OrderRepository, priceRepo port.PriceRepository, orgId, subscriptionId string) (int64, error) {
	items, err := orderRepo.FindOrderItemsBySubscriptionId(ctx, orgId, subscriptionId)
	if err != nil {
		return 0, err
	}
	var fixedBase int64
	for _, it := range items {
		p, perr := priceRepo.FindById(ctx, orgId, it.PriceId)
		if perr != nil {
			return 0, perr
		}
		if !p.IsMetered() {
			q := int64(it.Quantity)
			if q <= 0 {
				q = 1
			}
			fixedBase += domain.FixedLineAmount(p.UnitPrice, int64(p.UnitCount), q)
		}
	}
	return fixedBase, nil
}

// groupIntoSubscriptions partitions an order's recurring lines (any price with a
// real billing interval — fixed-subscription or metered) into one group per
// billing cadence. Each group becomes one subscription that bills all its lines
// together (flat + metered, on the same interval). One-time / free / no-interval
// lines are not grouped — they are charged once and start no subscription.
func groupIntoSubscriptions(lines []orderLine) [][]orderLine {
	type cadence struct {
		interval domain.BillingInterval
		qty      int
	}
	var order []cadence
	byCadence := map[cadence][]orderLine{}
	for _, l := range lines {
		if !l.price.IsRecurring() {
			continue // one-time / free line — no subscription
		}
		// Group by the line's *effective* cadence: a metered line is capped at
		// monthly, so an annual base + usage on one order yields an annual base
		// subscription and a separate monthly usage subscription.
		interval, qty := l.price.SubscriptionCadence()
		c := cadence{interval, qty}
		if _, ok := byCadence[c]; !ok {
			order = append(order, c)
		}
		byCadence[c] = append(byCadence[c], l)
	}
	groups := make([][]orderLine, 0, len(order))
	for _, c := range order {
		groups = append(groups, byCadence[c])
	}
	return groups
}
