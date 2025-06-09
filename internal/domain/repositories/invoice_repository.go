package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type InvoiceRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Invoice, error)
	Create(ctx context.Context, entity entities.Invoice) (entities.Invoice, error)
	Update(ctx context.Context, entity entities.Invoice) (entities.Invoice, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Invoice, int, error)
	FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]entities.Invoice, int, error)
	FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Invoice, int, error)
	FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]entities.Invoice, int, error)
	
	// Line items
	AddLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error)
	UpdateLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error)
	DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error
	ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error)
	
	// Invoice history
	AddHistory(ctx context.Context, history entities.InvoiceHistory) (entities.InvoiceHistory, error)
	ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error)
}