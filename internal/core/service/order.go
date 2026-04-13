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
func (s *OrderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (domain.CreateOrderResponse, error) {
	s.logger.Info("creating order", "sessionId", input.SessionId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")
	var customerEntity domain.Customer
	var err error
	var orderCart domain.Cart
	currency := domain.Currency(input.Currency)

	createPspSession := true
	if input.SessionId == "" {
		// no session, so we need to have a payment method set before we can activate the order
		createPspSession = false
	}

	// check if the cart exists
	if input.SessionId != "" {
		session, err := s.sessionRepository.FindById(ctx, orgId, input.SessionId)
		if err != nil {
			s.logger.Error("failed to find session", "sessionId", input.SessionId, "error", err)
			return domain.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "session not found", nil)
		}

		existingCart, err := s.cartRepository.FindById(ctx, orgId, session.CartId)
		if err != nil {
			s.logger.Error("failed to find cart", "cartId", session.CartId, "error", err)
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
				s.logger.Error("failed to find product", "error", err)
				return domain.CreateOrderResponse{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}

			price, err := s.priceRepository.FindById(ctx, orgId, item.PriceId)
			if err != nil {
				s.logger.Error("failed to find price", "error", err)
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
			s.logger.Error("failed to create cart", "error", err)
			return domain.CreateOrderResponse{}, err
		}
	}

	// validate that the cart is not empty
	if len(orderCart.Data.Items) == 0 {
		return domain.CreateOrderResponse{}, errors.New("cart is empty")
	}

	// Get or create the customer
	if input.Customer.Id != "" {
		customerEntity, err = s.customerRepository.FindById(ctx, orgId, input.Customer.Id)
		if err != nil {
			s.logger.Error("failed to find customer", "error", err)
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
			var derr lib.DatabaseError
			if errors.As(err, &derr) && derr.Code == lib.UniqueKeyViolation {
				customerEntity, err = s.customerRepository.Update(ctx, domain.Customer{
					OrgId:     orgId,
					Id:        lib.GenerateId("customer"),
					FirstName: input.Customer.FirstName,
					LastName:  input.Customer.LastName,
					Phone:     input.Customer.Phone,
					Email:     input.Customer.Email,
				})
				if err != nil {
					s.logger.Error("failed to update customer", "error", err)
					return domain.CreateOrderResponse{}, err
				}
			} else {
				s.logger.Error("failed to create customer", "error", err)
				return domain.CreateOrderResponse{}, err
			}
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
		s.logger.Error("failed to create order", "error", err)
		return domain.CreateOrderResponse{}, err
	}

	// Go through the list of items in the cart and create the order items for each item
	for _, item := range orderCart.Data.Items {
		orderItem, err := s.orderRepository.CreateOrderItem(ctx, domain.OrderItem{
			OrgId:         orgId,
			Id:            lib.GenerateId("order_item"),
			OrderId:       orderId,
			ProductId:     item.ProductId,
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
			s.logger.Error("failed to create order item", "item", item, "error", err)
			return domain.CreateOrderResponse{}, err
		}

		if orderItem.Price.Category == domain.PriceCategorySubscription {
			subscription := domain.NewSubscriptionFromOrderItem(orderItem)
			subscription.CustomerId = customerEntity.Id
			subscription.PspId = input.PspId
			subscription.PaymentMethodId = input.PaymentMethodId

			_, err := s.subscriptionRepository.Create(ctx, subscription)
			if err != nil {
				s.logger.Error("failed to create subscription", "item", item, "error", err)
				return domain.CreateOrderResponse{}, err
			}
			_ = s.pubsub.Publish(orgId, port.TopicSubscriptionCreated, subscription)
		}
	}

	var pspResponse domain.InitPaymentResponse
	if createPspSession {
		s.logger.Debug("creating payment session for order", "orderId", order.Id)
		gw, err := s.gatewayFactory.NewGateway(ctx, orgId, string(input.PspId))
		if err != nil {
			s.logger.Error("failed to get gateway", "error", err)
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
			s.logger.Error("failed to initialise payment gateway", "error", err)
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
		s.logger.Error("failed to list orders", "error", err)
		return nil, 0, err
	}
	return orders, total, nil
}

func (s *OrderService) ListOrderSubscriptions(ctx context.Context, orgId string, id string) ([]domain.Subscription, error) {
	s.logger.Info("listing subscriptions for order", "orgId", orgId, "orderId", id)

	_, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("order not found", "error", err)
		return nil, errors.New("order not found")
	}

	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to retrieve subscriptions", "error", err)
		return nil, err
	}

	return subscriptions, nil
}

// CompleteOrder marks a pending order as completed and activates subscriptions.
// No payment is involved - subscriptions start charging using the specified payment methods.
func (s *OrderService) CompleteOrder(ctx context.Context, input domain.CompleteOrderInput) (domain.Order, error) {
	s.logger.Info("completing order", "orgId", input.OrgId, "orderId", input.Id)

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
		s.logger.Error("failed to update order", "error", err)
		return domain.Order{}, err
	}

	var paymentMethod domain.PaymentMethod
	if input.PaymentMethodId != "" {
		paymentMethod, err = s.customerRepository.FindPaymentMethodById(ctx, order.OrgId, input.PaymentMethodId)
		if err != nil {
			s.logger.Error("failed to find payment method", "error", err)
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
			s.logger.Debug("payment method expiry", "expireAt", expireAt)
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
			s.logger.Error("failed to create payment method", "error", err)
			return domain.Order{}, err
		}
		s.logger.Debug("created payment method for order", "paymentMethodId", paymentMethod.Id, "orderId", order.Id)
	}

	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Info("no subscriptions to process", "error", err)
	}

	for _, subscription := range subscriptions {
		s.logger.Debug("setting subscription to active", "subscriptionId", subscription.Id)

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
				Currency:       domain.Currency(input.Payment.Currency),
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
				s.logger.Error("failed to create payment", "error", err)
				return domain.Order{}, err
			}
		}

		subscription.SetActive(payment)
		s.logger.Info("subscription activated", "subscriptionId", subscription.Id, "firstPaymentCharged", firstPaymentCharged)
		newSub, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("failed to update subscription", "error", err)
			return domain.Order{}, err
		}

		s.logger.Debug("starting subscription workflow")
		err = s.engine.StartSubscriptionWorkflow(ctx, newSub)
		if err != nil {
			s.logger.Error("failed to start workflow", "error", err)
			return domain.Order{}, err
		}
	}

	_ = s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order)
	return order, nil
}

// CompleteCheckoutSession marks a pending order as completed via a payment webhook.
// This handles the PSP-triggered flow (Paystack/Checkout.com webhook -> order completion).
func (s *OrderService) CompleteCheckoutSession(ctx context.Context, input domain.CompleteCheckoutSessionInput) (domain.Order, error) {
	s.logger.Info("completing order via checkout session", "orderId", input.OrderId)
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
		s.logger.Error("failed to update order", "error", err)
		return domain.Order{}, err
	}

	// The payment context comes from the webhook
	paymentCtx := input.PaymentContext

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
		s.logger.Error("failed to create payment method", "error", err)
		return domain.Order{}, err
	}
	s.logger.Info("created payment method for order", "paymentMethodId", paymentMethod.Id, "orderId", order.Id)

	var subscriptionId string
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("no subscriptions found", "error", err)
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
			s.logger.Error("failed to update subscription status", "error", err)
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
			s.logger.Error("failed to create payment", "error", err)
		}
	}

	_ = s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order)
	return order, nil
}
