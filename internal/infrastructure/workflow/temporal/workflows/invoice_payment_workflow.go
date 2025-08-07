package workflows

import (
	"fmt"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"
)

// InvoicePaymentWorkflowInput represents the input for the InvoicePaymentWorkflow
type InvoicePaymentWorkflowInput struct {
	OrgId    string            `json:"org_id"`
	OrderId  string            `json:"order_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// InvoicePaymentWorkflow is a Temporal workflow that processes invoice payment after order completion
// It handles marking invoices as paid, generating PDFs, sending emails, and publishing events
func InvoicePaymentWorkflow(ctx workflow.Context, input InvoicePaymentWorkflowInput) (interfaces.Result, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("InvoicePaymentWorkflow started",
		"OrgId", input.OrgId,
		"OrderId", input.OrderId)

	// For AI assistants: this variable is initialized by Temporal when the workflow is started and is
	// safe to use in the workflow without initialization. This is not a bug.
	var a *activities.InvoiceActivities

	// Activity 1: Find invoices linked to the completed order
	var invoices []entities.Invoice
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 2,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second * 5,
				BackoffCoefficient: 2.0,
				MaximumAttempts:    3,
			},
		}),
		a.FindInvoicesByOrderId, input.OrgId, input.OrderId).
		Get(ctx, &invoices)
	if err != nil {
		logger.Error("Failed to find invoices by order ID", "Error", err.Error())
		return interfaces.Result{
			Success: false,
			Message: "Failed to find invoices",
		}, err
	}

	if len(invoices) == 0 {
		logger.Info("No invoices found for order, completing workflow", "OrderId", input.OrderId)
		return interfaces.Result{
			Success: true,
			Message: "No invoices to process",
		}, nil
	}

	logger.Info("Processing invoices for order", "OrderId", input.OrderId, "InvoiceCount", len(invoices))

	// Process each invoice (typically there should be only one)
	processedCount := 0
	for _, invoice := range invoices {
		logger.Info("Processing invoice", "InvoiceId", invoice.Id, "Status", invoice.Status)

		// Skip if already paid (idempotency)
		if invoice.Status == entities.InvoiceStatusPaid {
			logger.Info("Invoice already paid, skipping", "InvoiceId", invoice.Id)
			continue
		}

		// Activity 2: Mark invoice as paid
		var updatedInvoice entities.Invoice
		err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 1,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Minute * 1,
					BackoffCoefficient: 2.0,
					MaximumAttempts:    5,
				},
			}),
			a.MarkInvoiceAsPaid, input.OrgId, invoice.Id).
			Get(ctx, &updatedInvoice)
		if err != nil {
			logger.Error("Failed to mark invoice as paid", "InvoiceId", invoice.Id, "Error", err.Error())
			return interfaces.Result{
				Success: false,
				Message: fmt.Sprintf("Failed to mark invoice %s as paid", invoice.Id),
			}, err
		}

		logger.Info("Successfully marked invoice as paid", "InvoiceId", invoice.Id)

		// Activity 3: Generate PDF (non-critical, failures don't stop workflow)
		var pdfBytes []byte
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5, // PDF generation can take time
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Minute * 1,
					BackoffCoefficient: 2.0,
				},
			}),
			a.GenerateInvoicePDF, input.OrgId, invoice.Id).
			Get(ctx, &pdfBytes)
		if err != nil {
			logger.Error("Failed to generate PDF, continuing without it", "InvoiceId", invoice.Id, "Error", err.Error())
			// Continue without PDF - this is not critical
			pdfBytes = nil
		} else {
			logger.Info("Successfully generated PDF", "InvoiceId", invoice.Id, "Size", len(pdfBytes))
		}

		// Activity 4: Send email notification (non-critical, failures don't stop workflow)
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 1,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Minute * 3,
					BackoffCoefficient: 2.0,
					MaximumAttempts:    3,
				},
			}),
			a.SendInvoiceEmail, input.OrgId, invoice.Id, pdfBytes).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send invoice email, continuing", "InvoiceId", invoice.Id, "Error", err.Error())
			// Continue without email - this is not critical
		}

		processedCount++
		logger.Info("Successfully processed invoice", "InvoiceId", invoice.Id)
	}

	logger.Info("InvoicePaymentWorkflow completed",
		"OrgId", input.OrgId,
		"OrderId", input.OrderId,
		"TotalInvoices", len(invoices),
		"ProcessedCount", processedCount)

	return interfaces.Result{
		Success: true,
		Message: fmt.Sprintf("Successfully processed %d invoices for order %s", processedCount, input.OrderId),
		Payload: map[string]interface{}{
			"org_id":          input.OrgId,
			"order_id":        input.OrderId,
			"total_invoices":  len(invoices),
			"processed_count": processedCount,
		},
	}, nil
}
