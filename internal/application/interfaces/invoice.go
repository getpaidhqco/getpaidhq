package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/pdf"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
)

type InvoiceService interface {
	// Invoice CRUD operations
	Create(ctx context.Context, orgId string, req dto.CreateInvoiceInput) (entities.Invoice, error)
	Get(ctx context.Context, orgId string, id string) (entities.Invoice, error)
	Update(ctx context.Context, orgId string, id string, req dto.UpdateInvoiceRequest) (entities.Invoice, error)
	List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Invoice, int, error)
	FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) ([]entities.Invoice, int, error)

	// Invoice actions
	PerformAction(ctx context.Context, orgId string, id string, req dto.InvoiceActionRequest) (entities.Invoice, error)

	// Line item operations
	AddLineItem(ctx context.Context, orgId string, invoiceId string, req dto.CreateInvoiceLineItemInput) (entities.InvoiceLineItem, error)
	UpdateLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string, req dto.UpdateInvoiceLineItemRequest) (entities.InvoiceLineItem, error)
	DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error
	ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error)

	// Invoice history
	ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error)

	// PDF generation
	GeneratePDF(ctx context.Context, orgId string, invoiceId string, options pdf.GenerateOptions) ([]byte, error)

	// Payment link generation
	CreatePaymentLink(ctx context.Context, orgId string, invoiceId string, input dto.CreateInvoicePaymentLinkInput) (entities.PaymentLink, error)

	// Payment initiation
	InitiatePayment(ctx context.Context, orgId string, invoiceId string, input dto.InitiatePaymentInput) (entities.Order, orders.CreateOrderResponse, error)
}
