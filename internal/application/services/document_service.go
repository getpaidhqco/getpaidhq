package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/storage/s3"
)

type DocumentService struct {
	documentRepo repositories.DocumentRepository
	s3Storage    *s3.S3Storage
}

func NewDocumentService(documentRepo repositories.DocumentRepository, s3Storage *s3.S3Storage) interfaces.DocumentService {
	return &DocumentService{
		documentRepo: documentRepo,
		s3Storage:    s3Storage,
	}
}

func (s *DocumentService) Upload(ctx context.Context, req interfaces.UploadRequest) (*interfaces.UploadResponse, error) {
	documentId := uuid.New().String()
	now := time.Now()

	storageKey := s.generateStorageKey(req.OrgId, req.Type, documentId, req.Filename)

	err := s.s3Storage.Upload(ctx, storageKey, req.Data, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	document := entities.Document{
		OrgId:           req.OrgId,
		Id:              documentId,
		InvoiceId:       req.InvoiceId,
		CreditNoteId:    req.CreditNoteId,
		Filename:        s.generateFilename(req.Type, documentId, req.Filename),
		OriginalName:    req.Filename,
		ContentType:     req.ContentType,
		Size:            len(req.Data),
		StorageProvider: "s3",
		StorageKey:      storageKey,
		Type:            req.Type,
		Purpose:         req.Purpose,
		IsPublic:        req.IsPublic,
		Metadata:        req.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if req.IsPublic {
		document.Url = s.s3Storage.GetPublicURL(storageKey)
	} else {
		document.AccessToken = uuid.New().String()
	}

	savedDocument, err := s.documentRepo.Create(ctx, document)
	if err != nil {
		s.s3Storage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	var url string
	if req.IsPublic {
		url = savedDocument.Url
	} else {
		url, err = s.s3Storage.GeneratePresignedURL(ctx, storageKey, 24*time.Hour)
		if err != nil {
			return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
		}
	}

	return &interfaces.UploadResponse{
		Document: &savedDocument,
		URL:      url,
	}, nil
}

func (s *DocumentService) GetDocument(ctx context.Context, orgId, documentId string) (entities.Document, error) {
	return s.documentRepo.FindById(ctx, orgId, documentId)
}

func (s *DocumentService) GetDocumentURL(ctx context.Context, orgId, documentId string, expiration time.Duration) (string, error) {
	document, err := s.documentRepo.FindById(ctx, orgId, documentId)
	if err != nil {
		return "", fmt.Errorf("document not found: %w", err)
	}

	if document.IsPublic && document.Url != "" {
		return document.Url, nil
	}

	return s.s3Storage.GeneratePresignedURL(ctx, document.StorageKey, expiration)
}

func (s *DocumentService) DownloadDocument(ctx context.Context, orgId, documentId string) ([]byte, error) {
	document, err := s.documentRepo.FindById(ctx, orgId, documentId)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	return s.s3Storage.Download(ctx, document.StorageKey)
}

func (s *DocumentService) DeleteDocument(ctx context.Context, orgId, documentId string) error {
	document, err := s.documentRepo.FindById(ctx, orgId, documentId)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	err = s.s3Storage.Delete(ctx, document.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	return s.documentRepo.Delete(ctx, orgId, documentId)
}

func (s *DocumentService) GetInvoiceDocuments(ctx context.Context, orgId, invoiceId string) ([]entities.Document, error) {
	return s.documentRepo.FindByInvoiceId(ctx, orgId, invoiceId)
}

func (s *DocumentService) GetCreditNoteDocuments(ctx context.Context, orgId, creditNoteId string) ([]entities.Document, error) {
	return s.documentRepo.FindByCreditNoteId(ctx, orgId, creditNoteId)
}

func (s *DocumentService) generateStorageKey(orgId string, docType entities.DocumentType, documentId, filename string) string {
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%s/documents/%s/%s%s", orgId, string(docType), documentId, ext)
}

func (s *DocumentService) generateFilename(docType entities.DocumentType, documentId, originalName string) string {
	ext := filepath.Ext(originalName)
	return fmt.Sprintf("%s_%s%s", string(docType), documentId[:8], ext)
}

func (s *DocumentService) UploadInvoicePDF(ctx context.Context, orgId, invoiceId string, pdfData []byte) (*entities.Document, error) {
	req := interfaces.UploadRequest{
		OrgId:       orgId,
		Data:        pdfData,
		Filename:    fmt.Sprintf("invoice_%s.pdf", invoiceId),
		ContentType: "application/pdf",
		Type:        entities.DocumentTypeInvoice,
		Purpose:     "invoice_pdf",
		InvoiceId:   invoiceId,
		IsPublic:    false,
		Metadata: map[string]string{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"version":      "1.0",
		},
	}

	response, err := s.Upload(ctx, req)
	if err != nil {
		return nil, err
	}

	return response.Document, nil
}
