package services

import (
	"context"
	"errors"
	"fmt"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
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
	var orderCart entities.Cart

	var createPspSession = true
	if input.SessionId == "" && input.PaymentMethodId == "" {
		return orders.CreateOrderResponse{}, lib.NewCustomError(
			lib.ValidationError,
			"You must specify a payment method or session_id",
			nil,
		)
	}
	if input.SessionId == "" {
		// no session, so we need to have a payment method
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
		orderCart = existingCart
	} else {
		// Create a cart from the items in the input
		inlineCart := cart.New(cart.CreateCartOptions{
			Currency: input.Currency,
			Items:    make([]cart.Item, 0),
		})
		for _, item := range input.CartItems {
			price, err := s.priceRepository.FindById(ctx, orgId, item.PriceId)
			if err != nil {
				s.logger.Error("Failed to find price", "price_id", item.PriceId, err.Error())
				return orders.CreateOrderResponse{}, lib.NewCustomError(
					lib.NotFoundError, fmt.Sprintf("Price %s not found", item.PriceId),
					err,
				)
			}

			_, err = inlineCart.AddItem(cart.Item{
				ID:          lib.GenerateId("item"),
				ProductId:   item.ProductId,
				Price:       price.ToCartItemPrice(),
				Description: price.Label,
				Quantity:    int64(item.Quantity),
			})
			if err != nil {
				s.logger.Error("Failed to add item to cart", err.Error())
				return orders.CreateOrderResponse{}, lib.NewCustomError(lib.InternalError, "Can't add item to cart", err)
			}
		}

		newCart, err := s.cartRepository.Create(ctx, entities.Cart{
			OrgId:    input.OrgId,
			Id:       lib.GenerateId("cart"),
			Data:     inlineCart,
			Metadata: nil,
		})
		if err != nil {
			s.logger.Error(`failed to create cart`, err)
			return orders.CreateOrderResponse{}, err
		}
		orderCart = newCart
	}

	// validate that the cart is not empty
	if len(orderCart.Data.Items) == 0 {
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
			s.logger.Error("Failed to create customer", err.Error())
			return orders.CreateOrderResponse{}, err
		}
	}

	order, err := s.orderRepository.Create(ctx, entities.Order{
		OrgId:      orgId,
		Id:         orderId,
		Reference:  orderId,
		CustomerId: customerEntity.Id,
		Status:     entities.OrderStatusPending,
		SessionId:  input.SessionId,
		CartId:     orderCart.Id,
		Currency:   input.Currency,
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
	for _, item := range orderCart.Data.Items {
		orderItem, err := s.orderItemRepository.Create(ctx, entities.OrderItem{
			OrgId:       orgId,
			Id:          lib.GenerateId("order_item"),
			OrderId:     orderId,
			PriceId:     item.Price.Id,
			Description: item.Description,
			Quantity:    int(item.Quantity),
			Metadata:    nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
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
			Cart:     orderCart.Data,
			Order:    order,
			Customer: customerEntity,
			Options:  input.Options,
		})
		if err != nil {
			s.logger.Error("Failed to initialise payment gateway", err.Error())
			return orders.CreateOrderResponse{}, err
		}
	}

	return orders.CreateOrderResponse{
		Order: order,
		Psp:   pspResponse,
	}, nil
}

// CompleteOrder marks a pending order as completed and activates the subscriptions. There is no payment involved, so the
// subscriptions will start charging as needed using the payment methods specified in the create process.
func (s OrderService) CompleteOrder(ctx context.Context, orgId string, orderId string) (entities.Order, error) {
	s.logger.Info("Completing order [%s][%s]", orgId, orderId)

	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		return entities.Order{}, errors.New("order not found")
	}

	// TODO validation

	// update the order status
	order.Status = entities.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return entities.Order{}, err
	}

	// find subscriptions for the order and update the status to active
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Info("no subscriptions to process", err.Error())
	}

	for _, subscription := range subscriptions {
		s.logger.Debugf("Setting subscription [%s] to active", subscription.Id)
		subscription.SetActive()

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
