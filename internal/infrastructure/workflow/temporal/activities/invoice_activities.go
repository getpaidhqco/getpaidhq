package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/pdf"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type InvoiceActivities struct {
	invoiceService  interfaces.InvoiceService
	customerService interfaces.CustomerService
	paymentService  interfaces.PaymentService
	pubsub          events.NotificationPublisher
	errorReporter   lib.ErrorReporter
}

func NewInvoiceActivities(
	invoiceService interfaces.InvoiceService,
	customerService interfaces.CustomerService,
	paymentService interfaces.PaymentService,
	pubsub events.NotificationPublisher,
	errorReporter lib.ErrorReporter,
) InvoiceActivities {
	return InvoiceActivities{
		invoiceService:  invoiceService,
		customerService: customerService,
		paymentService:  paymentService,
		pubsub:          pubsub,
		errorReporter:   errorReporter,
	}
}

// FindInvoicesByOrderId finds invoices linked to a completed order
func (a *InvoiceActivities) FindInvoicesByOrderId(ctx context.Context, orgId, orderId string) ([]entities.Invoice, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FindInvoicesByOrderId", "orgId", orgId, "orderId", orderId)

	invoices, count, err := a.invoiceService.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		logger.Error("Failed to find invoices by order ID", "error", err.Error(), "orgId", orgId, "orderId", orderId)
		return nil, temporal.NewNonRetryableApplicationError("Failed to find invoices", "invoice_lookup", err)
	}

	if count == 0 {
		logger.Info("No invoices found for order", "orgId", orgId, "orderId", orderId)
		return []entities.Invoice{}, nil
	}

	logger.Info("Found invoices for order", "count", count, "orgId", orgId, "orderId", orderId)
	return invoices, nil
}

// FindPaymentByOrderId finds the payment associated with a completed order
func (a *InvoiceActivities) FindPaymentByOrderId(ctx context.Context, orgId, orderId string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FindPaymentByOrderId", "orgId", orgId, "orderId", orderId)

	// Note: We need to find payments by OrderId, but the current PaymentService interface
	// might not have this method. For now, we'll return empty string and let the workflow
	// handle the case where PaymentId is not found.
	// TODO: Add FindByOrderId method to PaymentService interface

	logger.Info("Payment lookup by OrderId not yet implemented, returning empty PaymentId", "orgId", orgId, "orderId", orderId)
	return "", nil
}

// MarkInvoiceAsPaid updates invoice status to paid with proper timestamps and amounts
func (a *InvoiceActivities) MarkInvoiceAsPaid(ctx context.Context, orgId, invoiceId string) (entities.Invoice, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("MarkInvoiceAsPaid", "orgId", orgId, "invoiceId", invoiceId)

	// Get current invoice
	invoice, err := a.invoiceService.Get(ctx, orgId, invoiceId)
	if err != nil {
		logger.Error("Failed to get invoice", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId)
		return entities.Invoice{}, temporal.NewNonRetryableApplicationError("Invoice not found", "invoice_not_found", err)
	}

	// Check if already paid (idempotency)
	if invoice.Status == entities.InvoiceStatusPaid {
		logger.Info("Invoice already paid, skipping", "orgId", orgId, "invoiceId", invoiceId)
		return invoice, nil
	}

	// Delegate to service to mark invoice as paid
	// Note: This requires adding a method to InvoiceService
	updatedInvoice, err := a.invoiceService.MarkAsPaid(ctx, orgId, invoiceId)
	if err != nil {
		logger.Error("Failed to mark invoice as paid", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId)
		return entities.Invoice{}, temporal.NewApplicationError("Failed to mark invoice as paid", "", false, err)
	}

	logger.Info("Successfully marked invoice as paid", "orgId", orgId, "invoiceId", invoiceId)
	return updatedInvoice, nil
}

// ProcessInvoicePayment processes a payment for an invoice, handling partial payments correctly
func (a *InvoiceActivities) ProcessInvoicePayment(ctx context.Context, orgId, invoiceId, paymentId string) (entities.Invoice, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessInvoicePayment", "orgId", orgId, "invoiceId", invoiceId, "paymentId", paymentId)

	// Get current invoice
	invoice, err := a.invoiceService.Get(ctx, orgId, invoiceId)
	if err != nil {
		logger.Error("Failed to get invoice", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId)
		return entities.Invoice{}, temporal.NewNonRetryableApplicationError("Invoice not found", "invoice_not_found", err)
	}

	// Check if already paid (idempotency)
	if invoice.Status == entities.InvoiceStatusPaid {
		logger.Info("Invoice already paid, skipping", "orgId", orgId, "invoiceId", invoiceId)
		return invoice, nil
	}

	// Delegate to service to process invoice payment with payment information
	updatedInvoice, err := a.invoiceService.ProcessInvoicePayment(ctx, orgId, invoiceId, paymentId)
	if err != nil {
		logger.Error("Failed to process invoice payment", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId, "paymentId", paymentId)
		return entities.Invoice{}, temporal.NewApplicationError("Failed to process invoice payment", "", false, err)
	}

	logger.Info("Successfully processed invoice payment", "orgId", orgId, "invoiceId", invoiceId, "paymentId", paymentId, "status", updatedInvoice.Status)
	return updatedInvoice, nil
}

// GenerateInvoicePDF generates a PDF for the invoice
func (a *InvoiceActivities) GenerateInvoicePDF(ctx context.Context, orgId, invoiceId string) ([]byte, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GenerateInvoicePDF", "orgId", orgId, "invoiceId", invoiceId)

	pdfBytes, err := a.invoiceService.GeneratePDF(ctx, orgId, invoiceId, pdf.GenerateOptions{})
	if err != nil {
		logger.Error("Failed to generate PDF", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId)
		// PDF generation failure should not fail the entire workflow
		return nil, temporal.NewApplicationError("Failed to generate PDF", "", false, err)
	}

	logger.Info("Successfully generated invoice PDF", "orgId", orgId, "invoiceId", invoiceId, "size", len(pdfBytes))
	return pdfBytes, nil
}

// SendInvoiceEmail sends an email notification to the customer with PDF attachment
func (a *InvoiceActivities) SendInvoiceEmail(ctx context.Context, orgId, invoiceId string, pdfBytes []byte) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendInvoiceEmail", "orgId", orgId, "invoiceId", invoiceId)

	// Get invoice details
	invoice, err := a.invoiceService.Get(ctx, orgId, invoiceId)
	if err != nil {
		logger.Error("Failed to get invoice for email", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId)
		return temporal.NewNonRetryableApplicationError("Invoice not found", "invoice_not_found", err)
	}

	// Get customer details
	customer, err := a.customerService.Get(ctx, orgId, invoice.CustomerId)
	if err != nil {
		logger.Error("Failed to get customer for email", "error", err.Error(), "orgId", orgId, "customerId", invoice.CustomerId)
		return temporal.NewNonRetryableApplicationError("Customer not found", "customer_not_found", err)
	}

	if customer.Email == "" {
		logger.Warn("Customer has no email address, skipping email", "orgId", orgId, "customerId", customer.Id)
		return nil
	}

	// Delegate to service to send email
	// Note: This requires adding a method to InvoiceService
	err = a.invoiceService.SendInvoiceEmail(ctx, orgId, invoiceId, customer, invoice, pdfBytes)
	if err != nil {
		logger.Error("Failed to send invoice email", "error", err.Error(), "orgId", orgId, "invoiceId", invoiceId, "customerEmail", customer.Email)
		// Email failure should not fail the entire workflow, but should be retryable
		return temporal.NewApplicationError("Failed to send invoice email", "", false, err)
	}

	return nil
}
