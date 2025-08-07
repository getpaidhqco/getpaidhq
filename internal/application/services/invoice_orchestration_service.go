package services

import (
	"context"
	"encoding/json"
	"payloop/internal/lib"

	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	domainevents "payloop/internal/domain/events"
	"payloop/internal/lib/apperrors"
)

// InvoiceOrchestrationService is an extension of the InvoiceService that orchestrates invoice workflows.
type InvoiceOrchestrationService struct {
	interfaces.InvoiceService
	orderService    interfaces.OrderService
	customerService interfaces.CustomerService
	workflowEngine  interfaces.Engine
	pubsub          events.NotificationPublisher
	errorReporter   lib.ErrorReporter
	logger          logger.Logger
}

// NewInvoiceOrchestrationService creates a new InvoiceOrchestrationService
func NewInvoiceOrchestrationService(
	invoiceService interfaces.InvoiceService,
	orderService interfaces.OrderService,
	customerService interfaces.CustomerService,
	workflowEngine interfaces.Engine,
	pubsub events.NotificationPublisher,
	errorReporter lib.ErrorReporter,
	logger logger.Logger,
) interfaces.InvoiceOrchestrationService {
	svc := &InvoiceOrchestrationService{
		InvoiceService:  invoiceService,
		orderService:    orderService,
		customerService: customerService,
		workflowEngine:  workflowEngine,
		pubsub:          pubsub,
		errorReporter:   errorReporter,
		logger:          logger,
	}

	logger.Debugf("[InvoiceOrchestrationService] Subscribing to OrderCompleted topic")
	_, err := pubsub.Subscribe(topic.OrderCompleted, svc.HandleOrderCompletedEvent)
	if err != nil {
		logger.Error("Failed to subscribe to OrderCompleted topic", err.Error())
		panic(err)
	}

	return svc
}

// HandleOrderCompletedEvent starts an invoice payment workflow when an order is completed
func (s InvoiceOrchestrationService) HandleOrderCompletedEvent(t string, data []byte) {
	s.logger.Infof("[InvoiceOrchestrationService] handling topic: %s", t)

	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	var orderCompletedEvent domainevents.OrderCompletedEvent
	payloadBytes, err := json.Marshal(payload.Data)
	if err != nil {
		s.logger.Errorf("Failed to marshal payload data: %v", err)
		return
	}

	err = json.Unmarshal(payloadBytes, &orderCompletedEvent)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal order completed event: %v", err)
		return
	}

	order := orderCompletedEvent.Order
	payment := orderCompletedEvent.Payment

	// Start the invoice payment workflow
	workflowId, runId, err := s.workflowEngine.StartInvoicePaymentWorkflow(context.Background(), dto.InvoicePaymentWorkflowInput{
		OrgId:     order.OrgId,
		OrderId:   order.Id,
		PaymentId: payment.Id, // Use payment ID from the event
		Metadata: map[string]string{
			"triggered_by": "order_completed",
			"order_id":     order.Id,
		},
	})

	if err != nil {
		s.logger.Errorf("Failed to start invoice payment workflow: %v", err.Error())
		s.errorReporter.ReportError(context.Background(), err, map[string]interface{}{
			"operation": "start_invoice_payment_workflow",
			"org_id":    order.OrgId,
			"order_id":  order.Id,
		})
	} else {
		s.logger.Infof("Started invoice payment workflow successfully - WorkflowId: %s, RunId: %s", workflowId, runId)
	}
}

// InitiatePayment creates an order from the invoice and initiates payment with the specified PSP
func (s InvoiceOrchestrationService) InitiatePayment(ctx context.Context, orgId string, invoiceId string, input dto.InitiatePaymentInput) (orders.CreateOrderResponse, error) {
	// 1. Fetch and validate invoice
	invoice, err := s.Get(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to fetch invoice for payment initiation", err)
		return orders.CreateOrderResponse{}, apperrors.NotFound{Message: "Invoice not found", Err: err}
	}

	// Check if invoice is payable
	if invoice.Status == entities.InvoiceStatusPaid {
		return orders.CreateOrderResponse{}, apperrors.NewInvalidOperation("Invoice is already fully paid", nil)
	}

	if invoice.Status == entities.InvoiceStatusCancelled {
		return orders.CreateOrderResponse{}, apperrors.NewInvalidOperation("Cannot pay cancelled invoice", nil)
	}

	// 2. Get invoice line items to build cart items
	lineItems, err := s.ListLineItems(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to fetch invoice line items for payment", err)
		return orders.CreateOrderResponse{}, apperrors.InternalError{Message: "Error fetching invoice details", Err: err}
	}

	// 3. Get customer
	customer, err := s.customerService.Get(ctx, orgId, invoice.CustomerId)
	if err != nil {
		s.logger.Error("Failed to fetch customer for payment", err)
		return orders.CreateOrderResponse{}, apperrors.NotFound{Message: "Customer not found", Err: err}
	}

	// 4. Convert invoice line items to cart items
	cartItems := make([]orders.CartItem, len(lineItems))
	for i, item := range lineItems {
		cartItems[i] = orders.CartItem{
			ProductId: item.ProductId,
			PriceId:   item.PriceId,
			Quantity:  int(item.Quantity),
		}
	}

	// 5. Convert payment processor string to gateway enum
	var pspId common.Gateway
	switch input.PaymentProcessor {
	case "paystack":
		pspId = common.Paystack
	case "checkout_com":
		pspId = common.CheckoutDotCom
	default:
		return orders.CreateOrderResponse{}, apperrors.InvalidArgument{Message: "Unsupported payment processor", Err: nil}
	}

	// 6. Build order creation input
	orderInput := orders.CreateOrderInput{
		OrgId: orgId,
		Customer: orders.CreateOrderCommandCustomer{
			Id:        customer.Id,
			Email:     customer.Email,
			FirstName: customer.FirstName,
			LastName:  customer.LastName,
			Phone:     customer.Phone,
			Metadata:  customer.Metadata,
		},
		Currency:  invoice.Currency,
		CartItems: cartItems,
		PspId:     pspId,
		Metadata: map[string]string{
			"invoice_id": invoiceId,
			"source":     "invoice_payment",
		},
		Options: map[string]string{
			"success_url": input.SuccessUrl,
			"cancel_url":  input.CancelUrl,
		},
	}

	// Merge any additional metadata from input
	for k, v := range input.Metadata {
		orderInput.Metadata[k] = v
	}

	// 7. Create order using OrderService (this handles PSP InitPayment automatically)
	orderResponse, err := s.orderService.CreateOrder(ctx, orderInput)
	if err != nil {
		s.logger.Error("Failed to create order for invoice payment", err)
		return orders.CreateOrderResponse{}, apperrors.InternalError{Message: "Error initiating payment", Err: err}
	}

	s.logger.Info("Payment initiated for invoice", "invoice_id", invoiceId, "order_id", orderResponse.Order.Id, "psp", input.PaymentProcessor)

	return orderResponse, nil
}

// CreateOrderFromInvoice creates an order from an invoice via payment link and updates the invoice with the order ID
func (s InvoiceOrchestrationService) CreateOrderFromInvoice(ctx context.Context, orgId string, invoiceId string, input dto.CreateOrderFromInvoiceInput) (orders.CreateOrderResponse, error) {
	// 1. Fetch and validate invoice
	invoice, err := s.Get(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to fetch invoice for order creation", err)
		return orders.CreateOrderResponse{}, apperrors.NotFound{Message: "Invoice not found", Err: err}
	}

	// Check if invoice is payable
	if invoice.Status == entities.InvoiceStatusPaid {
		return orders.CreateOrderResponse{}, apperrors.NewInvalidOperation("Invoice is already fully paid", nil)
	}

	if invoice.Status == entities.InvoiceStatusCancelled {
		return orders.CreateOrderResponse{}, apperrors.NewInvalidOperation("Cannot pay cancelled invoice", nil)
	}

	// 2. Get invoice line items to build cart items
	lineItems, err := s.ListLineItems(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to fetch invoice line items for order creation", err)
		return orders.CreateOrderResponse{}, apperrors.InternalError{Message: "Error fetching invoice details", Err: err}
	}

	// 3. Get customer
	customer, err := s.customerService.Get(ctx, orgId, invoice.CustomerId)
	if err != nil {
		s.logger.Error("Failed to fetch customer for payment", err)
		return orders.CreateOrderResponse{}, apperrors.NotFound{Message: "Customer not found", Err: err}
	}

	// 4. Convert invoice line items to cart items
	cartItems := make([]orders.CartItem, len(lineItems))
	for i, item := range lineItems {
		cartItems[i] = orders.CartItem{
			ProductId: item.ProductId,
			PriceId:   item.PriceId,
			Quantity:  int(item.Quantity),
		}
	}

	// 5. Convert payment processor string to gateway enum
	var pspId common.Gateway
	switch input.PaymentProcessor {
	case "paystack":
		pspId = common.Paystack
	case "checkout_com":
		pspId = common.CheckoutDotCom
	default:
		return orders.CreateOrderResponse{}, apperrors.InvalidArgument{Message: "Unsupported payment processor: " + input.PaymentProcessor, Err: nil}
	}

	// 6. Build order creation input
	orderInput := orders.CreateOrderInput{
		OrgId: orgId,
		Customer: orders.CreateOrderCommandCustomer{
			Id:        customer.Id,
			Email:     customer.Email,
			FirstName: customer.FirstName,
			LastName:  customer.LastName,
			Phone:     customer.Phone,
			Metadata:  customer.Metadata,
		},
		Currency:  invoice.Currency,
		CartItems: cartItems,
		PspId:     pspId,
		Metadata: map[string]string{
			"invoice_id": invoiceId,
			"source":     "invoice_payment_link",
		},
		Options: map[string]string{
			"success_url": input.SuccessUrl,
			"cancel_url":  input.CancelUrl,
		},
	}

	// Merge any additional metadata from input
	for k, v := range input.Metadata {
		orderInput.Metadata[k] = v
	}

	// 7. Create an order using OrderService (this handles PSP InitPayment automatically)
	orderResponse, err := s.orderService.CreateOrder(ctx, orderInput)
	if err != nil {
		s.logger.Error("Failed to create order for invoice payment", err)
		return orders.CreateOrderResponse{}, apperrors.InternalError{Message: "Error creating order", Err: err}
	}

	// 8. Update invoice with the created order ID
	// We need to update via the underlying invoice service, but we need to use the repository directly for this update
	invoice.OrderId = orderResponse.Order.Id
	_, err = s.Update(ctx, orgId, invoice.Id, dto.UpdateInvoiceRequest{
		Metadata: map[string]string{
			"order_id": orderResponse.Order.Id,
		},
	})
	if err != nil {
		s.logger.Error("Failed to update invoice with order ID", err)
		// Continue even if update fails since the order was created successfully
	}

	s.logger.Info("Order created from invoice via payment link", "invoice_id", invoiceId, "order_id", orderResponse.Order.Id, "psp", input.PaymentProcessor)

	return orderResponse, nil
}
