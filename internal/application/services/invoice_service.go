package services

import (
	"context"
	"fmt"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/lib/pdf"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type InvoiceService struct {
	invoiceRepository    repositories.InvoiceRepository
	customerRepository   repositories.CustomerRepository
	docSequenceRepository repositories.DocSequenceRepository
	pubsub               events.PubSub
	logger               logger.Logger
}

func NewInvoiceService(
	invoiceRepository repositories.InvoiceRepository,
	customerRepository repositories.CustomerRepository,
	docSequenceRepository repositories.DocSequenceRepository,
	pubsub events.PubSub,
	logger logger.Logger,
) interfaces.InvoiceService {
	return InvoiceService{
		invoiceRepository:    invoiceRepository,
		customerRepository:   customerRepository,
		docSequenceRepository: docSequenceRepository,
		pubsub:               pubsub,
		logger:               logger,
	}
}

func (s InvoiceService) Create(ctx context.Context, orgId string, req request.CreateInvoiceRequest) (entities.Invoice, error) {
	// Validate customer exists
	if req.CustomerId != "" {
		_, err := s.customerRepository.FindById(ctx, orgId, req.CustomerId)
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

	// Create invoice
	invoice := entities.Invoice{
		OrgId:          orgId,
		Id:             lib.GenerateId("inv"),
		CustomerId:     req.CustomerId,
		OrderId:        req.OrderId,
		SubscriptionId: req.SubscriptionId,
		SequenceId:     sequenceId, // Using the sequence ID we generated
		DocNumber:      docNumber, // Using the formatted document number
		Type:           req.Type,
		InvoiceType:    req.InvoiceType,
		Status:         entities.InvoiceStatusDraft,
		IsImmutable:    false,
		Currency:       req.Currency,
		SubTotal:       0, // Will be calculated from line items
		TaxTotal:       0, // Will be calculated from line items
		DiscountTotal:  0, // Will be calculated from line items
		Total:          0, // Will be calculated from line items
		AmountPaid:     0,
		AmountDue:      0, // Will be calculated
		Notes:          req.Notes,
		CustomerNotes:  req.CustomerNotes,
		Metadata:       req.Metadata,
		DueAt:          req.DueAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Create invoice in database
	createdInvoice, err := s.invoiceRepository.Create(ctx, invoice)
	if err != nil {
		s.logger.Error("Failed to create invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error creating invoice", err)
	}

	// Add line items if provided
	if len(req.LineItems) > 0 {
		for _, item := range req.LineItems {
			lineItem := entities.InvoiceLineItem{
				OrgId:         orgId,
				InvoiceId:     createdInvoice.Id,
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
				DiscountTotal: 0, // Would be calculated based on discount type and value
				TaxCode:       item.TaxCode,
				TaxRate:       item.TaxRate,
				TaxAmount:     0, // Would be calculated based on tax rate
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

			_, err := s.invoiceRepository.AddLineItem(ctx, lineItem)
			if err != nil {
				s.logger.Error("Failed to add line item: ", err)
				// Continue with other line items even if one fails
			}
		}

		// Recalculate invoice totals
		updatedInvoice, err := s.recalculateInvoiceTotals(ctx, orgId, createdInvoice.Id)
		if err != nil {
			s.logger.Error("Failed to recalculate invoice totals: ", err)
			// Return the created invoice even if recalculation fails
			return createdInvoice, nil
		}
		createdInvoice = updatedInvoice
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

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.InvoiceCreated, createdInvoice)

	return createdInvoice, nil
}

func (s InvoiceService) Get(ctx context.Context, orgId string, id string) (entities.Invoice, error) {
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	return invoice, nil
}

func (s InvoiceService) Update(ctx context.Context, orgId string, id string, req request.UpdateInvoiceRequest) (entities.Invoice, error) {
	// Get existing invoice
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

func (s InvoiceService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	invoices, total, err := s.invoiceRepository.List(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list invoices: ", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error listing invoices", err)
	}

	return invoices, total, nil
}

func (s InvoiceService) FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	invoices, total, err := s.invoiceRepository.FindByCustomerId(ctx, orgId, customerId, pagination)
	if err != nil {
		s.logger.Error("Failed to find invoices by customer ID: ", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error finding invoices", err)
	}

	return invoices, total, nil
}

func (s InvoiceService) PerformAction(ctx context.Context, orgId string, id string, req request.InvoiceActionRequest) (entities.Invoice, error) {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", err)
	}

	// Perform action based on request
	switch req.Action {
	case "finalize":
		// Finalize invoice (make it immutable)
		if invoice.Status != entities.InvoiceStatusDraft {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only draft invoices can be finalized", nil)
		}
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

	case "send":
		// Send invoice to customer
		if invoice.Status != entities.InvoiceStatusDraft && invoice.Status != entities.InvoiceStatusOverdue {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only draft or overdue invoices can be sent", nil)
		}
		invoice.Status = entities.InvoiceStatusSent
		invoice.IssuedAt = time.Now().UTC()
		invoice.IsImmutable = true
		invoice.UpdatedAt = time.Now().UTC()

	case "mark_paid":
		// Mark invoice as paid
		if invoice.Status == entities.InvoiceStatusPaid {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invoice is already paid", nil)
		}
		invoice.Status = entities.InvoiceStatusPaid
		invoice.PaidAt = time.Now().UTC()
		invoice.AmountPaid = invoice.Total
		invoice.AmountDue = 0
		invoice.UpdatedAt = time.Now().UTC()

	case "mark_overdue":
		// Mark invoice as overdue
		if invoice.Status != entities.InvoiceStatusSent {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Only sent invoices can be marked as overdue", nil)
		}
		invoice.Status = entities.InvoiceStatusOverdue
		invoice.UpdatedAt = time.Now().UTC()

	case "cancel":
		// Cancel invoice
		if invoice.Status == entities.InvoiceStatusPaid || invoice.Status == entities.InvoiceStatusRefunded {
			return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Paid or refunded invoices cannot be cancelled", nil)
		}
		invoice.Status = entities.InvoiceStatusCancelled
		invoice.UpdatedAt = time.Now().UTC()

	default:
		return entities.Invoice{}, lib.NewCustomError(lib.BadRequestError, "Invalid action", nil)
	}

	// Update invoice in database
	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		s.logger.Error("Failed to update invoice: ", err)
		return entities.Invoice{}, lib.NewCustomError(lib.InternalError, "Error updating invoice", err)
	}

	// Add invoice history entry
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
	case "cancel":
		action = entities.InvoiceHistoryActionVoided
	}

	history := entities.InvoiceHistory{
		OrgId:     orgId,
		Id:        lib.GenerateId("inh"),
		InvoiceId: updatedInvoice.Id,
		Action:    action,
		Reason:    req.Reason,
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

func (s InvoiceService) AddLineItem(ctx context.Context, orgId string, invoiceId string, req request.CreateInvoiceLineItemRequest) (entities.InvoiceLineItem, error) {
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

func (s InvoiceService) UpdateLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string, req request.UpdateInvoiceLineItemRequest) (entities.InvoiceLineItem, error) {
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
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
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

	// Create PDF generator
	pdfGenerator := pdf.NewPDFGenerator()

	// Generate PDF
	pdfBytes, err := pdfGenerator.Generate(invoice, lineItems, options)
	if err != nil {
		s.logger.Error("Failed to generate PDF: ", err)
		return nil, lib.NewCustomError(lib.InternalError, "Error generating PDF", err)
	}

	return pdfBytes, nil
}

// Helper function to recalculate invoice totals based on line items
func (s InvoiceService) recalculateInvoiceTotals(ctx context.Context, orgId string, invoiceId string) (entities.Invoice, error) {
	// Get existing invoice
	invoice, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return entities.Invoice{}, err
	}

	// Get line items
	lineItems, err := s.invoiceRepository.ListLineItems(ctx, orgId, invoiceId)
	if err != nil {
		return entities.Invoice{}, err
	}

	// Calculate totals
	subTotal := 0
	taxTotal := 0
	discountTotal := 0

	for _, item := range lineItems {
		subTotal += int(item.Quantity * float64(item.UnitPrice))
		discountTotal += item.DiscountTotal
		taxTotal += item.TaxAmount
	}

	// Update invoice
	invoice.SubTotal = subTotal
	invoice.DiscountTotal = discountTotal
	invoice.TaxTotal = taxTotal
	invoice.Total = subTotal - discountTotal + taxTotal
	invoice.AmountDue = invoice.Total - invoice.AmountPaid
	invoice.UpdatedAt = time.Now().UTC()

	// Update invoice in database
	updatedInvoice, err := s.invoiceRepository.Update(ctx, invoice)
	if err != nil {
		return entities.Invoice{}, err
	}

	return updatedInvoice, nil
}
