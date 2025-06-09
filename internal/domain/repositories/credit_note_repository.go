package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type CreditNoteRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.CreditNote, error)
	Create(ctx context.Context, entity entities.CreditNote) (entities.CreditNote, error)
	Update(ctx context.Context, entity entities.CreditNote) (entities.CreditNote, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.CreditNote, int, error)
	FindByInvoiceId(ctx context.Context, orgId string, invoiceId string) ([]entities.CreditNote, error)
	
	// Line items
	AddLineItem(ctx context.Context, lineItem entities.CreditNoteLineItem) (entities.CreditNoteLineItem, error)
	UpdateLineItem(ctx context.Context, lineItem entities.CreditNoteLineItem) (entities.CreditNoteLineItem, error)
	DeleteLineItem(ctx context.Context, orgId string, creditNoteId string, lineItemId string) error
	ListLineItems(ctx context.Context, orgId string, creditNoteId string) ([]entities.CreditNoteLineItem, error)
}