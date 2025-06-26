package interfaces

import (
	"context"
	"time"

	"payloop/internal/domain/entities"
)

type DocumentService interface {
	Upload(ctx context.Context, req UploadRequest) (*UploadResponse, error)
	GetDocument(ctx context.Context, orgId, documentId string) (entities.Document, error)
	GetDocumentURL(ctx context.Context, orgId, documentId string, expiration time.Duration) (string, error)
	DownloadDocument(ctx context.Context, orgId, documentId string) ([]byte, error)
	DeleteDocument(ctx context.Context, orgId, documentId string) error
	GetInvoiceDocuments(ctx context.Context, orgId, invoiceId string) ([]entities.Document, error)
	GetCreditNoteDocuments(ctx context.Context, orgId, creditNoteId string) ([]entities.Document, error)
	UploadInvoicePDF(ctx context.Context, orgId, invoiceId string, pdfData []byte) (*entities.Document, error)
}

type UploadRequest struct {
	OrgId        string
	Data         []byte
	Filename     string
	ContentType  string
	Type         entities.DocumentType
	Purpose      string
	InvoiceId    string
	CreditNoteId string
	IsPublic     bool
	Metadata     map[string]string
}

type UploadResponse struct {
	Document *entities.Document
	URL      string
}