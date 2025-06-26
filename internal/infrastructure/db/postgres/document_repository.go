package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type DocumentRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewDocumentRepository(primaryDb lib.Database, logger logger.Logger) repositories.DocumentRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return DocumentRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r DocumentRepository) WithTrx(trxHandle interface{}) DocumentRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r DocumentRepository) FindById(ctx context.Context, orgId string, id string) (entities.Document, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at
			  FROM documents 
			  WHERE org_id = @org_id AND id = @id`

	var documentModel models.Document

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&documentModel.OrgId,
		&documentModel.Id,
		&documentModel.InvoiceId,
		&documentModel.CreditNoteId,
		&documentModel.Filename,
		&documentModel.OriginalName,
		&documentModel.ContentType,
		&documentModel.Size,
		&documentModel.StorageProvider,
		&documentModel.StorageKey,
		&documentModel.Url,
		&documentModel.Type,
		&documentModel.Purpose,
		&documentModel.IsPublic,
		&documentModel.AccessToken,
		&documentModel.Metadata,
		&documentModel.CreatedAt,
		&documentModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Document{}, err
		}
		r.logger.Error(`failed to find Document by id`, err.Error())
		return entities.Document{}, err
	}

	// Convert model to entity
	document := documentModel.ToEntity()
	return document, nil
}

func (r DocumentRepository) Create(ctx context.Context, entity entities.Document) (entities.Document, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO documents (org_id, id, invoice_id, credit_note_id, filename, 
              original_name, content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at) 
			  VALUES (@org_id, @id, @invoice_id, @credit_note_id, @filename, 
              @original_name, @content_type, @size, @storage_provider, @storage_key, @url, @type, 
              @purpose, @is_public, @access_token, @metadata, NOW(), NOW())`

	metadataJson, _ := json.Marshal(entity.Metadata)

	// Create pgtype values from the entity
	invoiceIdText := pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""}
	creditNoteIdText := pgtype.Text{String: entity.CreditNoteId, Valid: entity.CreditNoteId != ""}
	urlText := pgtype.Text{String: entity.Url, Valid: entity.Url != ""}
	purposeText := pgtype.Text{String: entity.Purpose, Valid: entity.Purpose != ""}
	accessTokenText := pgtype.Text{String: entity.AccessToken, Valid: entity.AccessToken != ""}

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":           entity.OrgId,
		"id":               entity.Id,
		"invoice_id":       invoiceIdText,
		"credit_note_id":   creditNoteIdText,
		"filename":         entity.Filename,
		"original_name":    entity.OriginalName,
		"content_type":     entity.ContentType,
		"size":             entity.Size,
		"storage_provider": entity.StorageProvider,
		"storage_key":      entity.StorageKey,
		"url":              urlText,
		"type":             string(entity.Type),
		"purpose":          purposeText,
		"is_public":        entity.IsPublic,
		"access_token":     accessTokenText,
		"metadata":         metadataJson,
	})

	if err != nil {
		r.logger.Error(`failed to insert Document`, err.Error())
		return entities.Document{}, err
	}

	// Get the created document
	createdDocument, err := r.FindById(ctx, entity.OrgId, entity.Id)
	if err != nil {
		return entities.Document{}, err
	}

	return createdDocument, nil
}

func (r DocumentRepository) Update(ctx context.Context, entity entities.Document) (entities.Document, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE documents
			  SET filename = @filename, original_name = @original_name, content_type = @content_type, 
              size = @size, storage_provider = @storage_provider, storage_key = @storage_key, 
              url = @url, type = @type, purpose = @purpose, is_public = @is_public, 
              access_token = @access_token, metadata = @metadata, updated_at = NOW()
			  WHERE org_id = @org_id AND id = @id`

	metadataJson, _ := json.Marshal(entity.Metadata)

	// Create pgtype values from the entity
	invoiceIdText := pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""}
	creditNoteIdText := pgtype.Text{String: entity.CreditNoteId, Valid: entity.CreditNoteId != ""}
	urlText := pgtype.Text{String: entity.Url, Valid: entity.Url != ""}
	purposeText := pgtype.Text{String: entity.Purpose, Valid: entity.Purpose != ""}
	accessTokenText := pgtype.Text{String: entity.AccessToken, Valid: entity.AccessToken != ""}

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":           entity.OrgId,
		"id":               entity.Id,
		"invoice_id":       invoiceIdText,
		"credit_note_id":   creditNoteIdText,
		"filename":         entity.Filename,
		"original_name":    entity.OriginalName,
		"content_type":     entity.ContentType,
		"size":             entity.Size,
		"storage_provider": entity.StorageProvider,
		"storage_key":      entity.StorageKey,
		"url":              urlText,
		"type":             string(entity.Type),
		"purpose":          purposeText,
		"is_public":        entity.IsPublic,
		"access_token":     accessTokenText,
		"metadata":         metadataJson,
	})

	if err != nil {
		r.logger.Error(`failed to update Document`, err.Error())
		return entities.Document{}, err
	}

	// Get the updated document
	updatedDocument, err := r.FindById(ctx, entity.OrgId, entity.Id)
	if err != nil {
		return entities.Document{}, err
	}

	return updatedDocument, nil
}

func (r DocumentRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM documents WHERE org_id = $1 AND id = $2`
	_, err := tx.Exec(ctx, query, orgId, id)

	if err != nil {
		r.logger.Error(`failed to delete Document`, err.Error())
		return err
	}

	return nil
}

func (r DocumentRepository) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Document, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var documents = make([]entities.Document, 0)
	var count int

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at,
              count(*) OVER()
			  FROM documents
			  WHERE org_id = @org_id
			  ORDER BY
			  CASE
				WHEN @sort_dir = 'asc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				ASC,
			  CASE
				WHEN @sort_dir = 'desc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				DESC
			  LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      pagination.Limit,
		"off":      pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to list Documents`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentModel models.Document

		err := rows.Scan(
			&documentModel.OrgId,
			&documentModel.Id,
			&documentModel.InvoiceId,
			&documentModel.CreditNoteId,
			&documentModel.Filename,
			&documentModel.OriginalName,
			&documentModel.ContentType,
			&documentModel.Size,
			&documentModel.StorageProvider,
			&documentModel.StorageKey,
			&documentModel.Url,
			&documentModel.Type,
			&documentModel.Purpose,
			&documentModel.IsPublic,
			&documentModel.AccessToken,
			&documentModel.Metadata,
			&documentModel.CreatedAt,
			&documentModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Document`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		document := documentModel.ToEntity()
		documents = append(documents, document)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return documents, count, nil
}

func (r DocumentRepository) FindByInvoiceId(ctx context.Context, orgId string, invoiceId string) ([]entities.Document, error) {
	tx := r.getTransactionFromContext(ctx)

	var documents = make([]entities.Document, 0)

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at
			  FROM documents
			  WHERE org_id = $1 AND invoice_id = $2
			  ORDER BY created_at DESC`

	rows, err := tx.Query(ctx, query, orgId, invoiceId)

	if err != nil {
		r.logger.Error(`failed to find Documents by invoice_id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentModel models.Document

		err := rows.Scan(
			&documentModel.OrgId,
			&documentModel.Id,
			&documentModel.InvoiceId,
			&documentModel.CreditNoteId,
			&documentModel.Filename,
			&documentModel.OriginalName,
			&documentModel.ContentType,
			&documentModel.Size,
			&documentModel.StorageProvider,
			&documentModel.StorageKey,
			&documentModel.Url,
			&documentModel.Type,
			&documentModel.Purpose,
			&documentModel.IsPublic,
			&documentModel.AccessToken,
			&documentModel.Metadata,
			&documentModel.CreatedAt,
			&documentModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error(`failed to scan Document`, err.Error())
			return nil, err
		}

		// Convert model to entity
		document := documentModel.ToEntity()
		documents = append(documents, document)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return documents, nil
}

func (r DocumentRepository) FindByCreditNoteId(ctx context.Context, orgId string, creditNoteId string) ([]entities.Document, error) {
	tx := r.getTransactionFromContext(ctx)

	var documents = make([]entities.Document, 0)

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at
			  FROM documents
			  WHERE org_id = $1 AND credit_note_id = $2
			  ORDER BY created_at DESC`

	rows, err := tx.Query(ctx, query, orgId, creditNoteId)

	if err != nil {
		r.logger.Error(`failed to find Documents by credit_note_id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentModel models.Document

		err := rows.Scan(
			&documentModel.OrgId,
			&documentModel.Id,
			&documentModel.InvoiceId,
			&documentModel.CreditNoteId,
			&documentModel.Filename,
			&documentModel.OriginalName,
			&documentModel.ContentType,
			&documentModel.Size,
			&documentModel.StorageProvider,
			&documentModel.StorageKey,
			&documentModel.Url,
			&documentModel.Type,
			&documentModel.Purpose,
			&documentModel.IsPublic,
			&documentModel.AccessToken,
			&documentModel.Metadata,
			&documentModel.CreatedAt,
			&documentModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error(`failed to scan Document`, err.Error())
			return nil, err
		}

		// Convert model to entity
		document := documentModel.ToEntity()
		documents = append(documents, document)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return documents, nil
}

func (r DocumentRepository) FindByType(ctx context.Context, orgId string, docType entities.DocumentType, pagination request.Pagination) ([]entities.Document, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var documents = make([]entities.Document, 0)
	var count int

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at,
              count(*) OVER()
			  FROM documents
			  WHERE org_id = @org_id AND type = @type
			  ORDER BY
			  CASE
				WHEN @sort_dir = 'asc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				ASC,
			  CASE
				WHEN @sort_dir = 'desc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				DESC
			  LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"type":     string(docType),
		"lim":      pagination.Limit,
		"off":      pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to find Documents by type`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentModel models.Document

		err := rows.Scan(
			&documentModel.OrgId,
			&documentModel.Id,
			&documentModel.InvoiceId,
			&documentModel.CreditNoteId,
			&documentModel.Filename,
			&documentModel.OriginalName,
			&documentModel.ContentType,
			&documentModel.Size,
			&documentModel.StorageProvider,
			&documentModel.StorageKey,
			&documentModel.Url,
			&documentModel.Type,
			&documentModel.Purpose,
			&documentModel.IsPublic,
			&documentModel.AccessToken,
			&documentModel.Metadata,
			&documentModel.CreatedAt,
			&documentModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Document`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		document := documentModel.ToEntity()
		documents = append(documents, document)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return documents, count, nil
}

func (r DocumentRepository) FindByPurpose(ctx context.Context, orgId string, purpose string, pagination request.Pagination) ([]entities.Document, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var documents = make([]entities.Document, 0)
	var count int

	query := `SELECT org_id, id, invoice_id, credit_note_id, filename, original_name, 
              content_type, size, storage_provider, storage_key, url, type, 
              purpose, is_public, access_token, metadata, created_at, updated_at,
              count(*) OVER()
			  FROM documents
			  WHERE org_id = @org_id AND purpose = @purpose
			  ORDER BY
			  CASE
				WHEN @sort_dir = 'asc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				ASC,
			  CASE
				WHEN @sort_dir = 'desc' THEN
					CASE @sort_col
						WHEN 'created_at' THEN created_at
						ELSE NULL
					END
				ELSE
					NULL
				END
				DESC
			  LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"purpose":  purpose,
		"lim":      pagination.Limit,
		"off":      pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to find Documents by purpose`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentModel models.Document

		err := rows.Scan(
			&documentModel.OrgId,
			&documentModel.Id,
			&documentModel.InvoiceId,
			&documentModel.CreditNoteId,
			&documentModel.Filename,
			&documentModel.OriginalName,
			&documentModel.ContentType,
			&documentModel.Size,
			&documentModel.StorageProvider,
			&documentModel.StorageKey,
			&documentModel.Url,
			&documentModel.Type,
			&documentModel.Purpose,
			&documentModel.IsPublic,
			&documentModel.AccessToken,
			&documentModel.Metadata,
			&documentModel.CreatedAt,
			&documentModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Document`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		document := documentModel.ToEntity()
		documents = append(documents, document)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return documents, count, nil
}
