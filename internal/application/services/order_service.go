package services

import (
	"context"
	"errors"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/cart"
	"payloop/internal/lib"
	"time"
)

type OrderService struct {
	workflowEngine         interfaces.Engine
	sessionRepository      repositories.SessionRepository
	cartRepository         repositories.CartRepository
	priceRepository        repositories.PriceRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	orderItemRepository    repositories.OrderItemRepository
	paymentRepository      repositories.PaymentRepository
	gatewayFactory         factories.GatewayFactory
	pubsub                 events.PubSub
	cartFactory            factories.CartFactory
	db                     lib.Database
	logger                 logger.Logger
}

func NewOrderService(
	workflowEngine interfaces.Engine,
	sessionRepository repositories.SessionRepository,
	priceRepository repositories.PriceRepository,
	cartRepository repositories.CartRepository,
	orderRepository repositories.OrderRepository,
	customerRepository repositories.CustomerRepository,
	orderItemRepository repositories.OrderItemRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	paymentRepository repositories.PaymentRepository,
	gatewayFactory factories.GatewayFactory,
	cartFactory factories.CartFactory,
	pubsub events.PubSub,
	db lib.Database,
	logger logger.Logger,
) interfaces.OrderService {
	return OrderService{
		workflowEngine:         workflowEngine,
		customerRepository:     customerRepository,
		priceRepository:        priceRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		logger:                 logger,
		gatewayFactory:         gatewayFactory,
		cartFactory:            cartFactory,
		paymentRepository:      paymentRepository,
		pubsub:                 pubsub,
		db:                     db,

		orderItemRepository: orderItemRepository,
	}
}

func (s OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderInput) (orders.CreateOrderResponse, error) {
	s.logger.Info("Creating order for cart", "cart", input.SessionId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")
	var customerEntity entities.Customer
	var err error
	var orderCart cart.Cart
	var currency = input.Currency

	var createPspSession = true
	if input.SessionId == "" {
		// no session, so we need to have a payment method set before we can activate the order
		createPspSession = false
	}

	// check if the cart exists
	if input.SessionId != "" {
		session, err := s.sessionRepository.FindById(ctx, orgId, input.SessionId)
		if err != nil {
			s.logger.Error("Failed to find session id ", "id", input.SessionId, err.Error())
			return orders.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "session not found", nil)
		}

		existingCart, err := s.cartRepository.FindById(ctx, orgId, session.CartId)
		if err != nil {
			s.logger.Error("Failed to find cart id ", "id", input.SessionId, err.Error())
			return orders.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "cart not found", nil)
		}
		orderCart = s.cartFactory.NewFromEntity(existingCart)
		currency = orderCart.Currency
	} else {

		// Create a cart from the items in the input
		orderCart = s.cartFactory.NewCart(orgId, common.Currency(currency))

		for _, item := range input.CartItems {
			_, err = orderCart.AddItem(ctx, cart.AddItemInput{
				ProductId: item.ProductId,
				PriceId:   item.PriceId,
				Quantity:  item.Quantity,
			})
			if err != nil {
				s.logger.Error("Failed to add item to cart", err.Error())
				return orders.CreateOrderResponse{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}
		}

		_, err := s.cartRepository.Create(ctx, entities.Cart{
			OrgId:    input.OrgId,
			Id:       orderCart.Id,
			Data:     orderCart.CartData,
			Metadata: nil,
		})
		if err != nil {
			s.logger.Error(`failed to create cart`, err)
			return orders.CreateOrderResponse{}, err
		}

	}

	// validate that the cart is not empty
	if len(orderCart.Items) == 0 {
		return orders.CreateOrderResponse{}, errors.New("cart is empty")
	}

	// Get or create the customer
	if input.Customer.Id != "" {
		customerEntity, err = s.customerRepository.FindById(ctx, orgId, input.Customer.Id)
		if err != nil {
			s.logger.Error("Failed to find customer", err.Error())
			return orders.CreateOrderResponse{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
		}

	} else {
		customerEntity, err = s.customerRepository.Create(ctx, entities.Customer{
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
				customerEntity, err = s.customerRepository.Update(ctx, entities.Customer{
					OrgId:     orgId,
					Id:        lib.GenerateId("customer"),
					FirstName: input.Customer.FirstName,
					LastName:  input.Customer.LastName,
					Phone:     input.Customer.Phone,
					Email:     input.Customer.Email,
				})
				if err != nil {
					s.logger.Error("Failed to update customer", err.Error())
					return orders.CreateOrderResponse{}, err
				}
			} else {
				s.logger.Error("Failed to create customer", err.Error())
				return orders.CreateOrderResponse{}, err
			}
		}
	}

	ref := time.Now().Format("20060102150405")
	order, err := s.orderRepository.Create(ctx, entities.Order{
		OrgId:      orgId,
		Id:         orderId,
		Reference:  ref,
		CustomerId: customerEntity.Id,
		Status:     entities.OrderStatusPending,
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
		return orders.CreateOrderResponse{}, err
	}

	// Go through the list of items in the cart and create the order items for each item
	for _, item := range orderCart.Items {
		orderItem, err := s.orderItemRepository.Create(ctx, entities.OrderItem{
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
			s.logger.Error("Failed to create order item", "item", item, "err", err.Error())
			return orders.CreateOrderResponse{}, err
		}

		if orderItem.Price.Category == prices.PriceCategorySubscription {
			subscription := entities.NewSubscriptionFromOrderItem(orderItem)
			subscription.CustomerId = customerEntity.Id
			subscription.PspId = input.PspId
			subscription.PaymentMethodId = input.PaymentMethodId

			_, err := s.subscriptionRepository.Create(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to create subscription", "item", item, err.Error())
				return orders.CreateOrderResponse{}, err
			}
		}
	}

	var pspResponse payment_providers.InitPaymentResponse
	if createPspSession {
		s.logger.Debugf("Creating payment session for order %s", order.Id)
		gw, err := s.gatewayFactory.NewGateway(ctx, orgId, input.PspId)
		if err != nil {
			s.logger.Error("Failed to get gateway", err.Error())
			return orders.CreateOrderResponse{}, err
		}
		// initialise the payment session with the payment processor
		pspResponse, err = gw.InitPayment(ctx, payment_providers.InitPaymentCommand{
			OrgId:    orgId,
			Cart:     orderCart,
			Order:    order,
			Customer: customerEntity,
			Options:  input.Options,
		})
		if err != nil {
			s.logger.Error("Failed to initialise payment gateway", err.Error())
			return orders.CreateOrderResponse{}, err
		}
	}

	newOrder, _ := s.orderRepository.FindById(ctx, orgId, order.Id)

	return orders.CreateOrderResponse{
		Order: newOrder,
		Psp:   pspResponse,
	}, nil
}

// CompleteOrder marks a pending order as completed and activates the subscriptions. There is no payment involved, so the
// subscriptions will start charging as needed using the payment methods specified in the create process.
func (s OrderService) CompleteOrder(ctx context.Context, input orders.CompleteOrderInput) (entities.Order, error) {
	s.logger.Info("Completing order [%s][%s]", input.OrgId, input.Id)

	order, err := s.orderRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		return entities.Order{}, errors.New("order not found")
	}

	// TODO validation
	if order.Status != entities.OrderStatusPending {
		return entities.Order{}, errors.New("order is not pending")
	}
	if input.PaymentMethodId == "" && input.PaymentMethod.Token == "" {
		return entities.Order{}, errors.New("payment method not provided")
	}

	// update the order status
	order.Status = entities.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	order.SetMetadata(input.Metadata)

	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return entities.Order{}, err
	}

	var paymentMethod entities.PaymentMethod
	if input.PaymentMethodId != "" {
		paymentMethod, err = s.customerRepository.FindPaymentMethodById(ctx, order.OrgId, input.PaymentMethodId)
		if err != nil {
			s.logger.Error("Failed to find payment method", err.Error())
			return entities.Order{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
		}
	}

	// create the payment method
	if input.PaymentMethod.Token != "" {
		// create the payment method
		paymentMethod, err = s.customerRepository.CreatePaymentMethod(ctx, entities.PaymentMethod{
			OrgId:          order.OrgId,
			Id:             lib.GenerateId("pm"),
			Psp:            input.PaymentMethod.Psp,
			Name:           input.PaymentMethod.Name,
			CustomerId:     order.CustomerId,
			IsDefault:      input.PaymentMethod.IsDefault,
			BillingAddress: entities.Address{},
			Type:           input.PaymentMethod.Type,
			Token:          input.PaymentMethod.Token,
			Details:        nil,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		})
		if err != nil {
			s.logger.Error("Failed to create payment method", err.Error())
			return entities.Order{}, err
		}
		s.logger.Debugf(`Created payment method [%s] for order [%s]`, paymentMethod.Id, order.Id)
	}

	// find subscriptions for the order and update the status to active
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Info("no subscriptions to process", err.Error())
	}

	for _, subscription := range subscriptions {
		s.logger.Debugf("Setting subscription [%s] to active", subscription.Id)

		// Set the payment method
		subscription.PaymentMethodId = paymentMethod.Id
		subscription.SetMetadata(input.Metadata)

		firstPaymentCharged := input.Payment.Amount > 0
		// Log the payment if it's the first payment
		if firstPaymentCharged {
			payment := entities.Payment{
				OrgId:          input.OrgId,
				Id:             lib.GenerateId("pmt"),
				Psp:            subscription.PspId,
				Recurring:      true,
				PspId:          input.Payment.PspId,
				Reference:      input.Payment.Reference,
				OrderId:        input.Id,
				SubscriptionId: subscription.Id,
				Status:         payments.PaymentStatusSucceeded,
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
			payment, err := s.paymentRepository.Create(ctx, payment)
			if err != nil {
				s.logger.Error("Failed to create payment", err.Error())
				return entities.Order{}, err
			}
		}

		// Set the activation dates
		subscription.SetActive(firstPaymentCharged)
		s.logger.Infof("Subscription [%s] activated. firstPaymentCharged=%t", subscription.Id, firstPaymentCharged)
		newSub, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription", err.Error())
			return entities.Order{}, err
		}

		s.logger.Debugf("Starting subscription workflow")
		err = s.workflowEngine.StartSubscriptionWorkflow(ctx, newSub)
		if err != nil {
			s.logger.Error("Failed to start workflow", err.Error())
			return entities.Order{}, err
		}
	}

	// publish order completed event
	_ = s.pubsub.Publish(order.OrgId, topic.OrderCompleted, order)

	return order, nil
}

// ListOrderSubscriptions retrieves the subscriptions for a given order.
func (s OrderService) ListOrderSubscriptions(ctx context.Context, orgId string, id string) ([]entities.Subscription, error) {
	s.logger.Info("Listing subscriptions for order [%s][%s]", orgId, id)

	// Find the order by Id
	_, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Order not found", err.Error())
		return nil, errors.New("order not found")
	}

	// Retrieve subscriptions associated with the order
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to retrieve subscriptions", err.Error())
		return nil, err
	}

	return subscriptions, nil
}

func (s OrderService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Order, int, error) {
	orders, total, err := s.orderRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list subscriptions", err.Error())
		return nil, 0, err
	}

	return orders, total, nil
}

func (s OrderService) FindById(ctx context.Context, orgId string, id string) (entities.Order, error) {
	order, err := s.orderRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Order not found", err.Error())
		return entities.Order{}, errors.New("order not found")
	}
	return order, nil
}
