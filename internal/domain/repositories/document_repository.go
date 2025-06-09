package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type DocumentRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Document, error)
	Create(ctx context.Context, entity entities.Document) (entities.Document, error)
	Update(ctx context.Context, entity entities.Document) (entities.Document, error)
	Delete(ctx context.Context, orgId string, id string) error
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Document, int, error)
	FindByInvoiceId(ctx context.Context, orgId string, invoiceId string) ([]entities.Document, error)
	FindByCreditNoteId(ctx context.Context, orgId string, creditNoteId string) ([]entities.Document, error)
	FindByType(ctx context.Context, orgId string, docType entities.DocumentType, pagination request.Pagination) ([]entities.Document, int, error)
	FindByPurpose(ctx context.Context, orgId string, purpose string, pagination request.Pagination) ([]entities.Document, int, error)
}
