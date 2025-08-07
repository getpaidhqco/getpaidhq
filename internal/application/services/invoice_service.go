package services

import (
	"context"
	"encoding/json"
	"fmt"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/lib/pdf"
	"payloop/internal/domain/email_providers"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_links"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"payloop/internal/lib/apperrors"
	"time"
)

type InvoiceService struct {
	invoiceRepository     repositories.InvoiceRepository
	customerRepository    repositories.CustomerRepository
	docSequenceRepository repositories.DocSequenceRepository
	orderRepository       repositories.OrderRepository
	paymentRepository     repositories.PaymentRepository
	orderItemRepository   repositories.OrderItemRepository
	errorReporter         lib.ErrorReporter
	pubsub                events.NotificationPublisher
	logger                logger.Logger
	transactionService    interfaces.TransactionService
	emailProvider         email_providers.Provider
	documentService       interfaces.DocumentService
	paymentLinkService    interfaces.PaymentLinkService
}

func NewInvoiceService(
	invoiceRepository repositories.InvoiceRepository,
	customerRepository repositories.CustomerRepository,
	docSequenceRepository repositories.DocSequenceRepository,
	orderRepository repositories.OrderRepository,
	paymentRepository repositories.PaymentRepository,
	orderItemRepository repositories.OrderItemRepository,
	errorReporter lib.ErrorReporter,
	pubsub events.NotificationPublisher,
	logger logger.Logger,
	transactionService interfaces.TransactionService,
	emailProvider email_providers.Provider,
	documentService interfaces.DocumentService,
	paymentLinkService interfaces.PaymentLinkService,
) interfaces.InvoiceService {
	service := InvoiceService{
		logger:                logger,
		invoiceRepository:     invoiceRepository,
		customerRepository:    customerRepository,
		docSequenceRepository: docSequenceRepository,
		orderRepository:       orderRepository,
		paymentRepository:     paymentRepository,
		orderItemRepository:   orderItemRepository,
		errorReporter:         errorReporter,
		pubsub:                pubsub,
		transactionService:    transactionService,
		emailProvider:         emailProvider,
		documentService:       documentService,
		paymentLinkService:    paymentLinkService,
	}

	// subscribe to order events to manage cohorts
	_, err := pubsub.Subscribe(topic.SubscriptionPaymentChargeSuccess, service.HandleSubscriptionPaymentSuccessEvent)
	if err != nil {
		logger.Errorf("Failed to subscribe to topic %s: %v", topic.SubscriptionPaymentChargeSuccess, err)
		panic(err)
	}

	return service
}

func (s InvoiceService) HandleSubscriptionPaymentSuccessEvent(eventTopic string, data []byte) {
	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	switch eventTopic {
	case topic.SubscriptionPaymentChargeSuccess:
		var paymentSuccess topic.SubscriptionPaymentChargeSuccessEvent
		payloadBytes, err := json.Marshal(payload.Data)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &paymentSuccess)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal event data: %v", err)
			return
		}

		s.logger.Infof("Creating invoice for subscription payment: %s", paymentSuccess.PaymentId)

		// Create an invoice for the subscription payment
		invoice, err := s.CreateInvoiceForSubscriptionPayment(context.Background(), paymentSuccess)
		if err != nil {
			s.logger.Errorf("Failed to create invoice for subscription payment: %v", err)
			s.errorReporter.ReportError(context.Background(), err, map[string]interface{}{
				"org_id": payload.OrgId,
				"id":     payload.Id,
			})
			return
		}

		s.logger.Infof("Successfully created invoice %s for subscription payment %s", invoice.Id, paymentSuccess.PaymentId)
	}
}

// CreateInvoiceForSubscriptionPayment creates an invoice for a subscription payment. Mark the invoice as
// paid
func (s InvoiceService) CreateInvoiceForSubscriptionPayment(ctx context.Context, paymentSuccess topic.SubscriptionPaymentChargeSuccessEvent) (entities.Invoice, error) {
	payment := paymentSuccess.Payment

	// get the line items from the order
	order, err := s.orderRepository.FindById(ctx, paymentSuccess.OrgId, paymentSuccess.OrderId)
	if err != nil {
		s.logger.Errorf("Failed to find order %s: %v", paymentSuccess.OrderId, err)
		return entities.Invoice{}, err
	}

	// Convert order items to invoice line items
	var lineItems []dto.CreateInvoiceLineItemInput
	for _, item := range order.Items {
		lineItem := dto.CreateInvoiceLineItemInput{
			ProductId:   item.ProductId,
			VariantId:   item.VariantId,
			PriceId:     item.PriceId,
			Description: item.Description,
			Quantity:    float64(item.Quantity),
			UnitPrice:   int(item.Subtotal / int64(item.Quantity)),
			Metadata:    item.Metadata,
		}
		lineItems = append(lineItems, lineItem)
	}

	// Create the invoice request
	invoiceReq := dto.CreateInvoiceInput{
		CustomerId:     order.CustomerId,
		OrderId:        order.Id,
		SubscriptionId: paymentSuccess.SubscriptionId,
		Status:         entities.InvoiceStatusPaid,
		Type:           entities.DocumentTypeInvoice,
		InvoiceType:    entities.InvoiceTypeRecurring,
		Currency:       payment.Currency,
		DueAt:          time.Now().UTC(),
		IssuedAt:       time.Now().UTC(),
		PaidAt:         paymentSuccess.Payment.CompletedAt,
		Notes:          fmt.Sprintf("Invoice for subscription payment %s", payment.Id),
		Metadata:       payment.Metadata,
		LineItems:      lineItems,
	}

	// Create the invoice
	invoice, err := s.Create(ctx, paymentSuccess.OrgId, invoiceReq)
	if err != nil {
		s.logger.Errorf("Failed to create invoice for subscription payment %s: %v", payment.Id, err)
		return entities.Invoice{}, apperrors.NewInternalError("Error creating invoice for subscription payment", err)
	}

	// update the payment with the invoice ID
	payment.InvoiceId = invoice.Id
	payment.UpdatedAt = time.Now().UTC()
	_, err = s.paymentRepository.Update(ctx, payment)
	if err != nil {
		s.logger.Errorf("Failed to update payment %s: %v", payment.Id, err)
		return entities.Invoice{}, apperrors.NewInternalError("Can't update payment with invoice ID", err)
	}

	return invoice, nil
}

func (s InvoiceService) Create(ctx context.Context, orgId string, input dto.CreateInvoiceInput) (entities.Invoice, error) {
	// Validate customer exists
	if input.CustomerId != "" {
		_, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
		if err != nil {
			return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
		}
	}

	// Get next invoice sequence number
	sequenceId := lib.GenerateId("seq")
	nextInvoiceNumber, err := s.docSequenceRepository.GetNextValue(ctx, orgId, sequenceId, "invoice")
	if err != nil {
		s.logger.Error("Failed to get next invoice sequence number: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error generating invoice sequence", err)
	}

	// Format the document number with the sequence
	docNumber := "INV-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", nextInvoiceNumber)

	status := entities.InvoiceStatusDraft
	if input.Status != "" {
		status = input.Status
	}

	// Create invoice
	invoice := entities.Invoice{
		OrgId:          orgId,
		Id:             lib.GenerateId("inv"),
		CustomerId:     input.CustomerId,
		OrderId:        input.OrderId,
		SubscriptionId: input.SubscriptionId,
		SequenceId:     sequenceId, // Using the sequence ID we generated
		DocNumber:      docNumber,  // Using the formatted document number
		Type:           input.Type,
		InvoiceType:    input.InvoiceType,
		Status:         status,
		IsImmutable:    status != entities.InvoiceStatusDraft,
		Currency:       input.Currency,
		SubTotal:       0, // Will be calculated from line items
		TaxTotal:       0, // Will be calculated from line items
		DiscountTotal:  0, // Will be calculated from line items
		Total:          0, // Will be calculated from line items
		AmountPaid:     0,
		AmountDue:      0, // Will be calculated
		IssuedAt:       input.IssuedAt,
		DueAt:          input.DueAt,
		PaidAt:         input.PaidAt,
		Notes:          input.Notes,
		CustomerNotes:  input.CustomerNotes,
		Metadata:       input.Metadata,
		ExchangeRate:   1,
		BaseCurrency:   input.Currency,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Add line items if provided
	if len(input.LineItems) > 0 {
		var lineItems []entities.InvoiceLineItem
		for _, item := range input.LineItems {
			lineItem := entities.InvoiceLineItem{
				OrgId:         orgId,
				InvoiceId:     invoice.Id,
				Id:            lib.GenerateId("ili"),
				ProductId:     item.ProductId,
				VariantId:     item.VariantId,
				PriceId:       item.PriceId,
				Description:   item.Description,
				Category:      item.Category,
				Quantity:      item.Quantity,
				UnitPrice:     item.UnitPrice,
				LineTotal:     int(item.Quantity * float64(item.UnitPrice)),
				DiscountType:  item.DiscountType,
				DiscountValue: item.DiscountValue,
				DiscountTotal: 0, // Will be calculated based on discount type and value
				TaxCode:       item.TaxCode,
				TaxRate:       item.TaxRate,
				TaxAmount:     0, // Will be calculated based on tax rate
				TaxExempt:     item.TaxExempt,
				Metadata:      item.Metadata,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			}

			// Calculate discount if applicable
			if item.DiscountType != "" && item.DiscountValue > 0 {
				if item.DiscountType == "percentage" {
					lineItem.DiscountTotal = int(float64(lineItem.LineTotal) * float64(item.DiscountValue) / 100.0)
				} else if item.DiscountType == "fixed" {
					lineItem.DiscountTotal = item.DiscountValue
				}
				lineItem.LineTotal -= lineItem.DiscountTotal
			}

			// Calculate tax if applicable and not exempt
			if !item.TaxExempt && item.TaxRate > 0 {
				lineItem.TaxAmount = int(float64(lineItem.LineTotal) * float64(item.TaxRate) / 100.0)
			}

			lineItems = append(lineItems, lineItem)
		}

		// Attach line items to invoice
		invoice.LineItems = lineItems
	}

	// Create invoice with line items in database
	createdInvoice, err := s.invoiceRepository.Create(ctx, invoice)
	if err != nil {
		s.logger.Error("Failed to create invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error creating invoice", err)
	}

	// Add invoice history entry
	history := entities.InvoiceHistory{
		OrgId:     orgId,
		Id:        lib.GenerateId("inh"),
		InvoiceId: createdInvoice.Id,
		Action:    entities.InvoiceHistoryActionCreated,
		Timestamp: time.Now().UTC(),
	}
	_, err = s.invoiceRepository.AddHistory(ctx, history)
	if err != nil {
		s.logger.Error("Failed to add invoice history: ", err)
		// Continue even if history creation fails
	}

	// create a PDF for the invoice
	// TODO this doesn't have to be done here, it can be done in a background job
	pdfGenerator := pdf.NewPDFGenerator(s.logger)
	_, err = pdfGenerator.Generate(createdInvoice, pdf.GenerateOptions{
		TemplateName: "one.liquid",
	})
	if err != nil {
		s.logger.Error("Failed to generate PDF for invoice: ", err)
	}

	_, err = s.CreatePaymentLink(ctx, orgId, createdInvoice.Id, dto.CreateInvoicePaymentLinkInput{
		Config: nil,
	})
	if err != nil {
		s.logger.Error("Failed to create payment link for invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error creating payment link", err)
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.InvoiceCreated, createdInvoice)

	return s.Get(ctx, orgId, createdInvoice.Id)
}

func (s InvoiceService) Get(ctx context.Context, orgId string, id string) (entities.Invoice, error) {
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	return invoice, nil
}

func (s InvoiceService) Update(ctx context.Context, orgId string, id string, req dto.UpdateInvoiceRequest) (entities.Invoice, error) {
	// Get existing invoice with line items
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Check if invoice is immutable
	if invoice.IsImmutable {
		return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice is immutable and cannot be updated", nil)
	}

	// Update fields
	if req.Notes != "" {
		invoice.Notes = req.Notes
	}
	if req.CustomerNotes != "" {
		invoice.CustomerNotes = req.CustomerNotes
	}
	if !req.DueAt.IsZero() {
		invoice.DueAt = req.DueAt
	}
	if req.Metadata != nil {
		invoice.Metadata = req.Metadata
	}
	invoice.UpdatedAt = time.Now().UTC()

	// Update line items if provided
	if req.LineItems != nil && len(req.LineItems) > 0 {
		// Create a map of existing line items for easier lookup
		existingLineItems := make(map[string]entities.InvoiceLineItem)
		for _, item := range invoice.LineItems {
			existingLineItems[item.Id] = item
		}

		var updatedLineItems []entities.InvoiceLineItem

		for _, item := range req.LineItems {
			if item.Id != "" && item.Id != "0" {
				// Update existing line item
				if existingItem, exists := existingLineItems[item.Id]; exists {
					// Update fields that are provided
					if item.Description != "" {
						existingItem.Description = item.Description
					}
					if item.Category != "" {
						existingItem.Category = item.Category
					}
					if item.Quantity != 0 {
						existingItem.Quantity = item.Quantity
					}
					if item.UnitPrice != 0 {
						existingItem.UnitPrice = item.UnitPrice
					}
					if item.DiscountType != "" {
						existingItem.DiscountType = item.DiscountType
					}
					if item.DiscountValue != 0 {
						existingItem.DiscountValue = item.DiscountValue
					}
					if item.TaxCode != "" {
						existingItem.TaxCode = item.TaxCode
					}
					if item.TaxRate != 0 {
						existingItem.TaxRate = item.TaxRate
					}
					existingItem.TaxExempt = item.TaxExempt
					if item.Metadata != nil {
						existingItem.Metadata = item.Metadata
					}
					existingItem.UpdatedAt = time.Now().UTC()

					// Recalculate line total
					existingItem.LineTotal = int(existingItem.Quantity * float64(existingItem.UnitPrice))

					// Recalculate discount if applicable
					if existingItem.DiscountType != "" && existingItem.DiscountValue > 0 {
						if existingItem.DiscountType == "percentage" {
							existingItem.DiscountTotal = int(float64(existingItem.LineTotal) * float64(existingItem.DiscountValue) / 100.0)
						} else if existingItem.DiscountType == "fixed" {
							existingItem.DiscountTotal = existingItem.DiscountValue
						}
						existingItem.LineTotal -= existingItem.DiscountTotal
					}

					// Recalculate tax if applicable and not exempt
					if !existingItem.TaxExempt && existingItem.TaxRate > 0 {
						existingItem.TaxAmount = int(float64(existingItem.LineTotal) * float64(existingItem.TaxRate) / 100.0)
					} else {
						existingItem.TaxAmount = 0
					}

					updatedLineItems = append(updatedLineItems, existingItem)
				}
			} else {
				// Create new line item
				newItem := entities.InvoiceLineItem{
					OrgId:         orgId,
					InvoiceId:     invoice.Id,
					Id:            lib.GenerateId("ili"),
					ProductId:     item.ProductId,
					VariantId:     item.VariantId,
					PriceId:       item.PriceId,
					Description:   item.Description,
					Category:      item.Category,
					Quantity:      item.Quantity,
					UnitPrice:     item.UnitPrice,
					LineTotal:     int(item.Quantity * float64(item.UnitPrice)),
					DiscountType:  item.DiscountType,
					DiscountValue: item.DiscountValue,
					DiscountTotal: 0, // Will be calculated
					TaxCode:       item.TaxCode,
					TaxRate:       item.TaxRate,
					TaxAmount:     0, // Will be calculated
					TaxExempt:     item.TaxExempt,
					Metadata:      item.Metadata,
					CreatedAt:     time.Now().UTC(),
					UpdatedAt:     time.Now().UTC(),
				}

				// Calculate discount if applicable
				if item.DiscountType != "" && item.DiscountValue > 0 {
					if item.DiscountType == "percentage" {
						newItem.DiscountTotal = int(float64(newItem.LineTotal) * float64(item.DiscountValue) / 100.0)
					} else if item.DiscountType == "fixed" {
						newItem.DiscountTotal = item.DiscountValue
					}
					newItem.LineTotal -= newItem.DiscountTotal
				}

				// Calculate tax if applicable and not exempt
				if !item.TaxExempt && item.TaxRate > 0 {
					newItem.TaxAmount = int(float64(newItem.LineTotal) * float64(item.TaxRate) / 100.0)
				}

				updatedLineItems = append(updatedLineItems, newItem)
			}
		}

		// Update the invoice's line items
		invoice.LineItems = updatedLineItems

		// Recalculate invoice totals
		invoice.RecalculateTotals()
	}

	// Update invoice in database
	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		s.logger.Error("Failed to update invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error updating invoice", err)
	}

	// Add invoice history entry
	history := entities.InvoiceHistory{
		OrgId:     orgId,
		Id:        lib.GenerateId("inh"),
		InvoiceId: updatedInvoice.Id,
		Action:    entities.InvoiceHistoryActionUpdated,
		Timestamp: time.Now().UTC(),
	}
	_, err = s.invoiceRepository.AddHistory(ctx, history)
	if err != nil {
		s.logger.Error("Failed to add invoice history: ", err)
		// Continue even if history creation fails
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.InvoiceUpdated, updatedInvoice)

	return updatedInvoice, nil
}

func (s InvoiceService) List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	invoices, total, err := s.invoiceRepository.List(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list invoices: ", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error listing invoices", err)
	}

	return invoices, total, nil
}

func (s InvoiceService) FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	invoices, total, err := s.invoiceRepository.FindByCustomerId(ctx, orgId, customerId, pagination)
	if err != nil {
		s.logger.Error("Failed to find invoices by customer ID: ", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error finding invoices", err)
	}

	return invoices, total, nil
}

func (s InvoiceService) PerformAction(ctx context.Context, orgId string, id string, req dto.InvoiceActionRequest) (entities.Invoice, error) {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Perform action based on request using business logic in service layer
	switch req.Action {
	case "finalize":
		// Finalize invoice (draft -> open) - Business Logic in Service
		if invoice.Status != entities.InvoiceStatusDraft {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only draft invoices can be finalized", nil)
		}
		
		// Apply business rules
		invoice.Status = entities.InvoiceStatusOpen
		invoice.IssuedAt = time.Now().UTC()
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to finalize invoice: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error finalizing invoice", err)
		}
		invoice = updatedInvoice

	case "send":
		// Mark invoice as delivered (email sent) - Business Logic in Service
		if !invoice.IsPayable() {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only payable invoices can be sent", nil)
		}
		
		// Apply business rules
		invoice.DeliveredAt = time.Now().UTC()
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to mark invoice as delivered: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error marking invoice as delivered", err)
		}
		invoice = updatedInvoice

	case "mark_paid":
		// Mark invoice as paid - Business Logic in Service
		if invoice.Status == entities.InvoiceStatusPaid {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice is already paid", nil)
		}
		if !invoice.IsPayable() {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice cannot accept payments in current status", nil)
		}
		
		// Apply business rules
		invoice.Status = entities.InvoiceStatusPaid
		invoice.PaidAt = time.Now().UTC()
		invoice.AmountPaid = invoice.Total
		invoice.AmountDue = 0
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to update invoice as paid: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error updating invoice", err)
		}
		invoice = updatedInvoice

	case "mark_overdue":
		// Mark invoice as overdue - Business Logic in Service
		if invoice.Status != entities.InvoiceStatusOpen {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only open invoices can be marked as overdue", nil)
		}
		
		// Apply business rules
		invoice.Status = entities.InvoiceStatusOverdue
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to mark invoice as overdue: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error updating invoice", err)
		}
		invoice = updatedInvoice

	case "void":
		// Void invoice - Business Logic in Service
		if !invoice.CanVoid() {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice cannot be voided in current status", nil)
		}
		
		// Apply business rules
		invoice.Status = entities.InvoiceStatusVoid
		invoice.VoidedAt = time.Now().UTC()
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to void invoice: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error voiding invoice", err)
		}
		invoice = updatedInvoice

		// Add history record for void action
		history := entities.InvoiceHistory{
			OrgId:     orgId,
			Id:        lib.GenerateId("inh"),
			InvoiceId: invoice.Id,
			Action:    entities.InvoiceHistoryActionVoided,
			Reason:    req.Reason,
			Timestamp: time.Now().UTC(),
		}
		_, err = s.invoiceRepository.AddHistory(ctx, history)
		if err != nil {
			s.logger.Error("Failed to add void history: ", err)
			// Continue even if history creation fails
		}

	case "mark_uncollectible":
		// Mark invoice as uncollectible - Business Logic in Service
		if !invoice.CanMarkUncollectible() {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice cannot be marked as uncollectible in current status", nil)
		}
		
		// Apply business rules
		invoice.Status = entities.InvoiceStatusUncollectible
		invoice.MarkedUncollectibleAt = time.Now().UTC()
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

		// Persist changes
		updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
		if err != nil {
			s.logger.Error("Failed to mark invoice as uncollectible: ", err)
			return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error marking invoice as uncollectible", err)
		}
		invoice = updatedInvoice

		// Add history record for uncollectible action
		history := entities.InvoiceHistory{
			OrgId:     orgId,
			Id:        lib.GenerateId("inh"),
			InvoiceId: invoice.Id,
			Action:    entities.InvoiceHistoryActionUncollectible,
			Reason:    req.Reason,
			Timestamp: time.Now().UTC(),
		}
		_, err = s.invoiceRepository.AddHistory(ctx, history)
		if err != nil {
			s.logger.Error("Failed to add uncollectible history: ", err)
			// Continue even if history creation fails
		}

	default:
		return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invalid action", nil)
	}

	// Add invoice history entry for actions that don't already add history
	if req.Action != "void" && req.Action != "mark_uncollectible" {
		var action entities.InvoiceHistoryAction
		switch req.Action {
		case "finalize":
			action = entities.InvoiceHistoryActionUpdated
		case "send":
			action = entities.InvoiceHistoryActionSent
		case "mark_paid":
			action = entities.InvoiceHistoryActionPaid
		case "mark_overdue":
			action = entities.InvoiceHistoryActionOverdue
		}

		history := entities.InvoiceHistory{
			OrgId:     orgId,
			Id:        lib.GenerateId("inh"),
			InvoiceId: invoice.Id,
			Action:    action,
			Reason:    req.Reason,
			Timestamp: time.Now().UTC(),
		}
		_, err = s.invoiceRepository.AddHistory(ctx, history)
		if err != nil {
			s.logger.Error("Failed to add invoice history: ", err)
			// Continue even if history creation fails
		}
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.InvoiceUpdated, invoice)

	return invoice, nil
}

func (s InvoiceService) AddLineItem(ctx context.Context, orgId string, invoiceId string, req dto.CreateInvoiceLineItemInput) (entities.InvoiceLineItem, error) {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Check if invoice is immutable
	if invoice.IsImmutable {
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.BadRequestError, "Invoice is immutable and cannot be updated", nil)
	}

	// Create line item
	lineItem := entities.InvoiceLineItem{
		OrgId:         orgId,
		InvoiceId:     invoiceId,
		Id:            lib.GenerateId("ili"),
		ProductId:     req.ProductId,
		VariantId:     req.VariantId,
		PriceId:       req.PriceId,
		Description:   req.Description,
		Category:      req.Category,
		Quantity:      req.Quantity,
		UnitPrice:     req.UnitPrice,
		LineTotal:     int(req.Quantity * float64(req.UnitPrice)),
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		DiscountTotal: 0, // Will be calculated
		TaxCode:       req.TaxCode,
		TaxRate:       req.TaxRate,
		TaxAmount:     0, // Will be calculated
		TaxExempt:     req.TaxExempt,
		Metadata:      req.Metadata,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	// Calculate discount if applicable
	if req.DiscountType != "" && req.DiscountValue > 0 {
		if req.DiscountType == "percentage" {
			lineItem.DiscountTotal = int(float64(lineItem.LineTotal) * float64(req.DiscountValue) / 100.0)
		} else if req.DiscountType == "fixed" {
			lineItem.DiscountTotal = req.DiscountValue
		}
		lineItem.LineTotal -= lineItem.DiscountTotal
	}

	// Calculate tax if applicable and not exempt
	if !req.TaxExempt && req.TaxRate > 0 {
		lineItem.TaxAmount = int(float64(lineItem.LineTotal) * float64(req.TaxRate) / 100.0)
	}

	// Add line item to database
	createdLineItem, err := s.invoiceRepository.AddLineItem(ctx, lineItem)
	if err != nil {
		s.logger.Error("Failed to add line item: ", err)
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.InternalError, "Error adding line item", err)
	}

	// Recalculate invoice totals
	_, err = s.recalculateInvoiceTotals(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to recalculate invoice totals: ", err)
		// Return the created line item even if recalculation fails
	}

	return createdLineItem, nil
}

func (s InvoiceService) UpdateLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string, req dto.UpdateInvoiceLineItemRequest) (entities.InvoiceLineItem, error) {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Check if invoice is immutable
	if invoice.IsImmutable {
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.BadRequestError, "Invoice is immutable and cannot be updated", nil)
	}

	// Get line items
	lineItems, err := s.invoiceRepository.ListLineItems(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to list line items: ", err)
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.InternalError, "Error listing line items", err)
	}

	// Find the line item to update
	var lineItem entities.InvoiceLineItem
	found := false
	for _, item := range lineItems {
		if item.Id == lineItemId {
			lineItem = item
			found = true
			break
		}
	}

	if !found {
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.NotFoundError, "Line item not found", nil)
	}

	// Update fields
	if req.Description != "" {
		lineItem.Description = req.Description
	}
	if req.Category != "" {
		lineItem.Category = req.Category
	}
	if req.Quantity != 0 {
		lineItem.Quantity = req.Quantity
	}
	if req.UnitPrice != 0 {
		lineItem.UnitPrice = req.UnitPrice
	}
	if req.DiscountType != "" {
		lineItem.DiscountType = req.DiscountType
	}
	if req.DiscountValue != 0 {
		lineItem.DiscountValue = req.DiscountValue
	}
	if req.TaxCode != "" {
		lineItem.TaxCode = req.TaxCode
	}
	if req.TaxRate != 0 {
		lineItem.TaxRate = req.TaxRate
	}
	lineItem.TaxExempt = req.TaxExempt
	if req.Metadata != nil {
		lineItem.Metadata = req.Metadata
	}
	lineItem.UpdatedAt = time.Now().UTC()

	// Recalculate line total
	lineItem.LineTotal = int(lineItem.Quantity * float64(lineItem.UnitPrice))

	// Recalculate discount if applicable
	if lineItem.DiscountType != "" && lineItem.DiscountValue > 0 {
		if lineItem.DiscountType == "percentage" {
			lineItem.DiscountTotal = int(float64(lineItem.LineTotal) * float64(lineItem.DiscountValue) / 100.0)
		} else if lineItem.DiscountType == "fixed" {
			lineItem.DiscountTotal = lineItem.DiscountValue
		}
		lineItem.LineTotal -= lineItem.DiscountTotal
	}

	// Recalculate tax if applicable and not exempt
	if !lineItem.TaxExempt && lineItem.TaxRate > 0 {
		lineItem.TaxAmount = int(float64(lineItem.LineTotal) * float64(lineItem.TaxRate) / 100.0)
	} else {
		lineItem.TaxAmount = 0
	}

	// Update line item in database
	updatedLineItem, err := s.invoiceRepository.UpdateLineItem(ctx, lineItem)
	if err != nil {
		s.logger.Error("Failed to update line item: ", err)
		return entities.InvoiceLineItem{}, lib.NewCustomError(lib.InternalError, "Error updating line item", err)
	}

	// Recalculate invoice totals
	_, err = s.recalculateInvoiceTotals(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to recalculate invoice totals: ", err)
		// Return the updated line item even if recalculation fails
	}

	return updatedLineItem, nil
}

func (s InvoiceService) DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Check if invoice is immutable
	if invoice.IsImmutable {
		return lib.NewCustomError(lib.BadRequestError, "Invoice is immutable and cannot be updated", nil)
	}

	// Delete line item from database
	err = s.invoiceRepository.DeleteLineItem(ctx, orgId, invoiceId, lineItemId)
	if err != nil {
		s.logger.Error("Failed to delete line item: ", err)
		return lib.NewCustomError(lib.InternalError, "Error deleting line item", err)
	}

	// Recalculate invoice totals
	_, err = s.recalculateInvoiceTotals(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to recalculate invoice totals: ", err)
		// Return success even if recalculation fails
	}

	return nil
}

func (s InvoiceService) ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error) {
	// Get existing invoice
	_, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return nil, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Get line items
	lineItems, err := s.invoiceRepository.ListLineItems(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to list line items: ", err)
		return nil, lib.NewCustomError(lib.InternalError, "Error listing line items", err)
	}

	return lineItems, nil
}

func (s InvoiceService) ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error) {
	// Get existing invoice
	_, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return nil, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Get history
	history, err := s.invoiceRepository.ListHistory(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to list invoice history: ", err)
		return nil, lib.NewCustomError(lib.InternalError, "Error listing invoice history", err)
	}

	return history, nil
}

func (s InvoiceService) GeneratePDF(ctx context.Context, orgId string, invoiceId string, options pdf.GenerateOptions) ([]byte, error) {
	// Get invoice with line items in a single call
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice with line items: ", err)
		return nil, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Create PDF generator
	pdfGenerator := pdf.NewPDFGenerator(s.logger)

	// Generate PDF with complete invoice aggregate
	if options.TemplateName == "" {
		options.TemplateName = "one.liquid"
	}

	pdfBytes, err := pdfGenerator.Generate(invoice, options)
	if err != nil {
		s.logger.Error("Failed to generate PDF: ", err)
		return nil, lib.NewCustomError(lib.InternalError, "Error generating PDF", err)
	}

	return pdfBytes, nil
}

func (s InvoiceService) GenerateAndStorePDF(ctx context.Context, orgId string, invoiceId string, options pdf.GenerateOptions) (*entities.Document, error) {
	// Generate PDF
	pdfBytes, err := s.GeneratePDF(ctx, orgId, invoiceId, options)
	if err != nil {
		return nil, err
	}

	// Upload PDF to storage
	document, err := s.documentService.UploadInvoicePDF(ctx, orgId, invoiceId, pdfBytes)
	if err != nil {
		s.logger.Error("Failed to upload invoice PDF: ", err)
		return nil, lib.NewCustomError(lib.InternalError, "Error storing invoice PDF", err)
	}

	s.logger.Info("Invoice PDF generated and stored successfully", "invoice_id", invoiceId, "document_id", document.Id)
	return document, nil
}

// CreatePaymentLink creates a single-use payment link for an invoice. the default expiry date is 30 days from the invoice due date.
func (s InvoiceService) CreatePaymentLink(ctx context.Context, orgId string, invoiceId string, input dto.CreateInvoicePaymentLinkInput) (dto.InvoicePaymentLinkCreationResult, error) {
	// Get the invoice to validate it exists and extract details
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		s.logger.Error("Failed to get invoice for payment link creation", err)
		return dto.InvoicePaymentLinkCreationResult{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Validate invoice can have a payment link
	if invoice.Status == entities.InvoiceStatusPaid || invoice.Status == entities.InvoiceStatusVoid {
		return dto.InvoicePaymentLinkCreationResult{}, lib.NewCustomError(lib.ValidationError, "Cannot create payment link for paid or voided invoice", nil)
	}

	// Generate unique slug for the payment link
	slug := fmt.Sprintf("invoice-%s-%d", invoice.DocNumber, time.Now().Unix())

	// Set default expiration to invoice due date if not provided
	if input.ExpiresAt.IsZero() && !invoice.DueAt.IsZero() {
		input.ExpiresAt = invoice.DueAt.Add(30 * 24 * time.Hour)
	}

	// Prepare payment link data with invoice details
	paymentLinkData := map[string]interface{}{
		"invoice_id":  invoice.Id,
		"amount":      invoice.Total,
		"currency":    invoice.Currency,
		"customer_id": invoice.CustomerId,
		"type":        "invoice",
		"description": fmt.Sprintf("Payment for Invoice %s", invoice.DocNumber),
	}

	// Prepare default config with optional overrides
	paymentLinkConfig := map[string]interface{}{
		"payment_provider": "paystack", // default provider
		"success_url":      fmt.Sprintf("/invoices/%s/success", invoiceId),
		"cancel_url":       fmt.Sprintf("/invoices/%s/cancel", invoiceId),
	}

	// Apply user-provided config overrides
	if input.Config != nil {
		for key, value := range input.Config {
			paymentLinkConfig[key] = value
		}
	}

	// Override success/cancel URLs if provided in input
	if input.SuccessUrl != "" {
		paymentLinkConfig["success_url"] = input.SuccessUrl
	}
	if input.CancelUrl != "" {
		paymentLinkConfig["cancel_url"] = input.CancelUrl
	}

	// Create payment link using payment link service
	createInput := payment_links.CreatePaymentLinkInput{
		Slug:      slug,
		Data:      paymentLinkData,
		Config:    paymentLinkConfig,
		SingleUse: true, // Always single-use for invoices
		ExpiresAt: input.ExpiresAt.Format(time.RFC3339),
	}

	result, err := s.paymentLinkService.CreatePaymentLink(ctx, orgId, createInput)
	if err != nil {
		s.logger.Error("Failed to create payment link for invoice", err)
		return dto.InvoicePaymentLinkCreationResult{}, lib.NewCustomError(lib.InternalError, "Error creating payment link", err)
	}

	s.logger.Info("Payment link created for invoice", "invoice_id", invoiceId, "payment_link_id", result.PaymentLink.Id, "token", result.Token)
	return dto.InvoicePaymentLinkCreationResult{
		PaymentLink: result.PaymentLink,
		Token:       result.Token,
	}, nil
}

// Helper function to recalculate invoice totals based on line items
func (s InvoiceService) recalculateInvoiceTotals(ctx context.Context, orgId string, invoiceId string) (entities.Invoice, error) {
	// Get invoice with line items in a single call
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return entities.Invoice{}, err
	}

	// Use entity business logic to recalculate totals
	invoice.RecalculateTotals()
	invoice.UpdatedAt = time.Now().UTC()

	// Update invoice in database
	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		return entities.Invoice{}, err
	}

	return updatedInvoice, nil
}

// InitiatePayment creates an order from the invoice and initiates payment with the specified PSP

// FindByOrderId finds invoices linked to an order - delegates to repository
func (s InvoiceService) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Invoice, int, error) {
	return s.invoiceRepository.FindByOrderId(ctx, orgId, orderId)
}

// MarkAsPaid marks an invoice as paid with proper timestamps and amounts
func (s InvoiceService) MarkAsPaid(ctx context.Context, orgId string, invoiceId string) (entities.Invoice, error) {
	// Get current invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return entities.Invoice{}, apperrors.NotFound{Message: "Invoice not found", Err: err}
	}

	// Check if already paid (idempotency)
	if invoice.Status == entities.InvoiceStatusPaid {
		s.logger.Infof("Invoice %s is already paid", invoiceId)
		return invoice, nil
	}

	// Update invoice to paid status
	invoice.Status = entities.InvoiceStatusPaid
	invoice.PaidAt = time.Now()
	invoice.AmountPaid = invoice.Total
	invoice.AmountDue = 0
	invoice.IsImmutable = true // Paid invoices are immutable

	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		return entities.Invoice{}, apperrors.InternalError{Message: "Failed to mark invoice as paid", Err: err}
	}

	err = s.pubsub.Publish(orgId, topic.InvoicePaid, invoice)
	if err != nil {
		s.logger.Error("Failed to publish invoice paid event: ", err)
		return entities.Invoice{}, apperrors.InternalError{Message: "Failed to publish invoice paid event", Err: err}
	}
	s.logger.Infof("Successfully marked invoice %s as paid", invoiceId)
	return updatedInvoice, nil
}

// ProcessInvoicePayment processes a payment for an invoice, handling partial payments correctly
func (s InvoiceService) ProcessInvoicePayment(ctx context.Context, orgId string, invoiceId string, paymentId string) (entities.Invoice, error) {
	// Get current invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return entities.Invoice{}, apperrors.NotFound{Message: "Invoice not found", Err: err}
	}

	// Check if already paid (idempotency)
	if invoice.Status == entities.InvoiceStatusPaid {
		s.logger.Infof("Invoice %s is already paid", invoiceId)
		return invoice, nil
	}

	// Get the payment information
	payment, err := s.paymentRepository.FindById(ctx, orgId, paymentId)
	if err != nil {
		return entities.Invoice{}, apperrors.NotFound{Message: "Payment not found", Err: err}
	}

	// Ensure payment is linked to the invoice
	if payment.InvoiceId != invoiceId {
		// Link the payment to the invoice if not already linked
		payment.InvoiceId = invoiceId
		_, err = s.paymentRepository.Update(ctx, payment)
		if err != nil {
			return entities.Invoice{}, apperrors.InternalError{Message: "Failed to link payment to invoice", Err: err}
		}
		s.logger.Infof("Linked payment %s to invoice %s", paymentId, invoiceId)
	}

	// Get all payments for this invoice to calculate total amount paid
	payments, _, err := s.paymentRepository.FindByInvoiceId(ctx, orgId, invoiceId, entities.Pagination{
		Page:  1,
		Limit: 1000, // Get all payments for this invoice
	})
	if err != nil {
		return entities.Invoice{}, apperrors.InternalError{Message: "Failed to get invoice payments", Err: err}
	}

	// Calculate total amount paid from all successful payments
	var totalAmountPaid int64 = 0
	for _, p := range payments {
		// Only count successful payments
		if p.Status == "completed" || p.Status == "succeeded" {
			totalAmountPaid += p.Amount
		}
	}

	// Update invoice with payment information
	invoice.AmountPaid = int(totalAmountPaid)
	invoice.AmountDue = invoice.Total - int(totalAmountPaid)

	// If invoice is fully paid or overpaid, mark as paid
	if totalAmountPaid >= int64(invoice.Total) {
		invoice.Status = entities.InvoiceStatusPaid
		invoice.PaidAt = time.Now()
		invoice.AmountDue = 0
		invoice.IsImmutable = true // Paid invoices are immutable
		s.logger.Infof("Invoice %s is fully paid with total payments of %d", invoiceId, totalAmountPaid)
	} else {
		// Partial payment - keep as open but update amounts
		s.logger.Infof("Invoice %s partially paid: %d of %d", invoiceId, totalAmountPaid, invoice.Total)
	}

	// Update the invoice
	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		return entities.Invoice{}, apperrors.InternalError{Message: "Failed to update invoice with payment", Err: err}
	}

	// Publish event only if invoice is fully paid
	if invoice.Status == entities.InvoiceStatusPaid {
		err = s.pubsub.Publish(orgId, topic.InvoicePaid, invoice)
		if err != nil {
			s.logger.Error("Failed to publish invoice paid event: ", err)
			return entities.Invoice{}, apperrors.InternalError{Message: "Failed to publish invoice paid event", Err: err}
		}
		s.logger.Infof("Successfully marked invoice %s as paid", invoiceId)
	}

	return updatedInvoice, nil
}

// SendInvoiceEmail sends an email to the customer with invoice PDF attachment using existing email provider
func (s InvoiceService) SendInvoiceEmail(ctx context.Context, orgId, invoiceId string, customer entities.Customer, invoice entities.Invoice, pdfBytes []byte) error {
	// Use orgId as organization name (can be enhanced later to fetch org details)
	orgName := orgId

	// Prepare invoice email data variables for the template
	dataVariables := map[string]string{
		// Customer information
		"customer_first_name": customer.FirstName,
		"customer_last_name":  customer.LastName,
		"customer_email":      customer.Email,

		// Invoice information
		"invoice_number":      invoice.DocNumber,
		"invoice_id":          invoice.Id,
		"invoice_total":       fmt.Sprintf("%.2f", invoice.Total),
		"invoice_currency":    invoice.Currency,
		"invoice_due_date":    invoice.DueAt.Format("January 2, 2006"),
		"invoice_issued_date": invoice.IssuedAt.Format("January 2, 2006"),

		// Organization information
		"org_name": orgName,
		"org_id":   invoice.OrgId,

		// Additional metadata
		"payment_status": string(invoice.Status),
	}

	// Add invoice metadata if present
	if invoice.Metadata != nil {
		for key, value := range invoice.Metadata {
			dataVariables["invoice_"+key] = fmt.Sprintf("%v", value)
		}
	}

	// Add customer metadata if present
	if customer.Metadata != nil {
		for key, value := range customer.Metadata {
			dataVariables["customer_"+key] = fmt.Sprintf("%v", value)
		}
	}

	// Create PDF attachment
	attachments := []email_providers.EmailAttachment{ // TODO: Uncomment when PDF generation is allowed on Loops
		//{
		//	Filename:    fmt.Sprintf("invoice_%s.pdf", invoice.DocNumber),
		//	ContentType: "application/pdf",
		//	Data:        pdfBytes,
		//},
	}

	// Send invoice email using the new unified method
	_, err := s.emailProvider.SendEmail(ctx, orgId, email_providers.EmailTypeInvoicePaid, email_providers.SendEmailInput{
		To:      customer.Email,
		Subject: fmt.Sprintf("Invoice %s - %s", invoice.DocNumber, orgName),
		Variables: map[string]interface{}{
			"name":             customer.FirstName,
			"invoiceReference": invoice.DocNumber,
			"replyTo":          "no-reply@getpaidhq.co", // TODO
			"subject":          fmt.Sprintf("Invoice %s from %s", invoice.DocNumber, orgName),
			"preview":          fmt.Sprintf("You have received an invoice from %s", orgName),
		},
		Attachments: attachments,
		Metadata:    dataVariables,
	})
	if err != nil {
		return apperrors.InternalError{Message: fmt.Sprintf("Failed to send invoice email to %s", customer.Email), Err: err}
	}

	s.logger.Infof("Successfully sent invoice email for %s to %s", invoice.DocNumber, customer.Email)
	return nil
}
