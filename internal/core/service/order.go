package service

import (
	"context"
	"errors"
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
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
	}
}

// CreateOrder creates a new order from a session/cart or direct cart items.
func (s *OrderService) CreateOrder(ctx context.Context, input port.CreateOrderInput) (domain.CreateOrderResponse, error) {
	s.logger.Info("Creating order for cart", "session", input.SessionId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")
	var customerEntity domain.Customer
	var err error
	var orderCart domain.Cart
	currency := input.Currency

	createPspSession := true
	if input.SessionId == "" {
		// no session, so we need to have a payment method set before we can activate the order
		createPspSession = false
	}

	// check if the cart exists
	if input.SessionId != "" {
		session, err := s.sessionRepository.FindById(ctx, orgId, input.SessionId)
		if err != nil {
			s.logger.Error("Failed to find session id ", "id", input.SessionId, err.Error())
			return domain.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "session not found", nil)
		}

		existingCart, err := s.cartRepository.FindById(ctx, orgId, session.CartId)
		if err != nil {
			s.logger.Error("Failed to find cart id ", "id", input.SessionId, err.Error())
			return domain.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "cart not found", nil)
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
				return domain.CreateOrderResponse{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}

			price, err := s.priceRepository.FindById(ctx, orgId, item.PriceId)
			if err != nil {
				s.logger.Error("Failed to find price", err.Error())
				return domain.CreateOrderResponse{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}

			orderCart.Data.Items = append(orderCart.Data.Items, domain.CartLineItem{
				Id:            lib.GenerateId("ci"),
				ProductId:     product.Id,
				Price:         domain.PriceToCartItemPrice(price),
				Description:   product.Name,
				Quantity:      int64(item.Quantity),
				UnitPrice:     price.UnitPrice,
				SubTotal:      price.UnitPrice * int64(item.Quantity),
				DiscountTotal: 0,
				TaxTotal:      0,
				ShippingTotal: 0,
				Total:         price.UnitPrice * int64(item.Quantity),
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
			return domain.CreateOrderResponse{}, err
		}
	}

	// validate that the cart is not empty
	if len(orderCart.Data.Items) == 0 {
		return domain.CreateOrderResponse{}, errors.New("cart is empty")
	}

	// Archived products are retired and not sellable. Guard here so the rule holds
	// for both order paths (existing session cart and direct cart-items) — and
	// therefore for subscriptions, which are only created via the order flow. This
	// runs only at checkout; recurring renewals never call CreateOrder.
	for _, item := range orderCart.Data.Items {
		product, err := s.productRepository.FindById(ctx, orgId, item.ProductId)
		if err != nil {
			s.logger.Error("Failed to find product", err.Error())
			return domain.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "Product not found", err)
		}
		if product.IsArchived() {
			return domain.CreateOrderResponse{}, lib.NewCustomError(
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
			return domain.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
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
			return domain.CreateOrderResponse{}, err
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
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create order", err.Error())
		return domain.CreateOrderResponse{}, err
	}

	// Create a subscription for the order. A subscription is a recurring agreement;
	// its pricing method (fixed or metered) is orthogonal to its cadence — a metered
	// price is a recurring subscription billed by usage, not a rider on a fixed plan.
	//
	// Each fixed (subscription-category) item creates its own subscription, which bills
	// its flat fee plus the order's metered usage (multiline). If the order has NO fixed
	// plan, the metered usage stands on its own: one subscription anchored on the first
	// metered item bills the order's usage each cycle.
	startSubscription := func(orderItem domain.OrderItem, price domain.Price) error {
		subscription := domain.NewSubscriptionFromOrderItem(orderItem, price)
		subscription.CustomerId = customerEntity.Id
		subscription.PspId = input.PspId
		subscription.PaymentMethodId = input.PaymentMethodId
		if _, err := s.subscriptionRepository.Create(ctx, subscription); err != nil {
			return err
		}
		_ = s.pubsub.Publish(orgId, port.TopicSubscriptionCreated, subscription)
		return nil
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
			return domain.CreateOrderResponse{}, err
		}

		price, err := s.priceRepository.FindById(ctx, orgId, item.Price.Id)
		if err != nil {
			s.logger.Error("Failed to find price for order item", "price_id", item.Price.Id, "err", err.Error())
			return domain.CreateOrderResponse{}, err
		}
		lines = append(lines, orderLine{item: orderItem, price: price})
	}

	for _, anchor := range subscriptionAnchors(lines) {
		if err := startSubscription(anchor.item, anchor.price); err != nil {
			s.logger.Error("Failed to create subscription", "order_item", anchor.item.Id, "err", err.Error())
			return domain.CreateOrderResponse{}, err
		}
	}

	var pspResponse domain.InitPaymentResponse
	if createPspSession {
		s.logger.Debugf("Creating payment session for order %s", order.Id)
		gw, err := s.gatewayFactory.NewGateway(ctx, orgId, string(input.PspId))
		if err != nil {
			s.logger.Error("Failed to get gateway", "err", err.Error())
			return domain.CreateOrderResponse{}, err
		}
		// initialise the payment session with the payment processor
		pspResponse, err = gw.InitPayment(ctx, domain.InitPaymentCommand{
			OrgId:    orgId,
			Cart:     orderCart,
			Order:    order,
			Customer: customerEntity,
			Options:  input.Options,
		})
		if err != nil {
			s.logger.Error("Failed to initialise payment gateway", err.Error())
			return domain.CreateOrderResponse{}, err
		}
	}

	newOrder, _ := s.orderRepository.FindById(ctx, orgId, order.Id)

	return domain.CreateOrderResponse{
		Order: newOrder,
		Psp:   pspResponse,
	}, nil
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
//
// DB writes (order update, payment method find/create, payment+subscription
// state per sub) run inside a single transaction. Post-commit side effects
// (subscription workflow starts and the order.completed pubsub event) fire
// only after the tx commits — running them inside would orphan workflows
// and pubsub messages on rollback.
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
				Token:          input.PaymentMethod.Token,
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

		activated = make([]domain.Subscription, 0, len(subscriptions))
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
					return err
				}
			}

			itemForPrice, err := s.orderRepository.FindOrderItemById(ctx, input.OrgId, subscription.OrderItemId)
			if err != nil {
				s.logger.Error("Failed to find order item for subscription activation", "subscription_id", subscription.Id, "err", err.Error())
				return err
			}
			subPrice, err := s.priceRepository.FindById(ctx, input.OrgId, itemForPrice.PriceId)
			if err != nil {
				s.logger.Error("Failed to find price for subscription activation", "price_id", itemForPrice.PriceId, "err", err.Error())
				return err
			}
			subscription.SetActive(subPrice, payment)
			s.logger.Infof("Subscription [%s] activated. firstPaymentCharged=%t", subscription.Id, firstPaymentCharged)
			newSub, err := s.subscriptionRepository.Update(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to update subscription", "err", err.Error())
				return err
			}
			activated = append(activated, newSub)
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

// orderLine pairs a persisted order item with its resolved price.
type orderLine struct {
	item  domain.OrderItem
	price domain.Price
}

// subscriptionAnchors decides which order items start a subscription. A subscription
// is a recurring agreement; its pricing method (fixed or metered) is orthogonal to
// cadence. Every fixed (subscription-category) item anchors its own subscription —
// which bills its flat fee plus the order's metered usage. If the order has no fixed
// plan, the first metered item anchors one subscription so pure-usage orders still
// bill each cycle. One-time / free / variable items never start a subscription.
func subscriptionAnchors(lines []orderLine) []orderLine {
	var anchors []orderLine
	firstMetered := -1
	for i, l := range lines {
		if l.price.IsMetered() {
			if firstMetered == -1 {
				firstMetered = i
			}
			continue // metered usage rides the order's fixed plan, if any
		}
		if l.price.Category == domain.PriceCategorySubscription {
			anchors = append(anchors, l)
		}
	}
	if len(anchors) == 0 && firstMetered >= 0 {
		anchors = append(anchors, lines[firstMetered])
	}
	return anchors
}
