package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type InvoiceService interface {
	// Invoice CRUD operations
	Create(ctx context.Context, orgId string, req request.CreateInvoiceRequest) (entities.Invoice, error)
	Get(ctx context.Context, orgId string, id string) (entities.Invoice, error)
	Update(ctx context.Context, orgId string, id string, req request.UpdateInvoiceRequest) (entities.Invoice, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Invoice, int, error)
	FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]entities.Invoice, int, error)
	
	// Invoice actions
	PerformAction(ctx context.Context, orgId string, id string, req request.InvoiceActionRequest) (entities.Invoice, error)
	
	// Line item operations
	AddLineItem(ctx context.Context, orgId string, invoiceId string, req request.CreateInvoiceLineItemRequest) (entities.InvoiceLineItem, error)
	UpdateLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string, req request.UpdateInvoiceLineItemRequest) (entities.InvoiceLineItem, error)
	DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error
	ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error)
	
	// Invoice history
	ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error)
}