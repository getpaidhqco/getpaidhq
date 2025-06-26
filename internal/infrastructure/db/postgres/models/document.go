package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Document struct {
	OrgId           string             `json:"org_id"`
	Id              string             `json:"id"`
	InvoiceId       pgtype.Text        `json:"invoice_id"`
	CreditNoteId    pgtype.Text        `json:"credit_note_id"`
	Filename        string             `json:"filename"`
	OriginalName    string             `json:"original_name"`
	ContentType     string             `json:"content_type"`
	Size            int                `json:"size"`
	StorageProvider string             `json:"storage_provider"`
	StorageKey      string             `json:"storage_key"`
	Url             pgtype.Text        `json:"url"`
	Type            string             `json:"type"`
	Purpose         pgtype.Text        `json:"purpose"`
	IsPublic        bool               `json:"is_public"`
	AccessToken     pgtype.Text        `json:"access_token"`
	Metadata        []byte             `json:"metadata"`
	CreatedAt       pgtype.Timestamptz `json:"created_at"`
	UpdatedAt       pgtype.Timestamptz `json:"updated_at"`
}

func (d *Document) ToEntity() entities.Document {
	var metadata map[string]string

	// Handle JSON fields
	if d.Metadata != nil {
		_ = json.Unmarshal(d.Metadata, &metadata)
	}

	return entities.Document{
		OrgId:           d.OrgId,
		Id:              d.Id,
		InvoiceId:       d.InvoiceId.String,
		CreditNoteId:    d.CreditNoteId.String,
		Filename:        d.Filename,
		OriginalName:    d.OriginalName,
		ContentType:     d.ContentType,
		Size:            d.Size,
		StorageProvider: d.StorageProvider,
		StorageKey:      d.StorageKey,
		Url:             d.Url.String,
		Type:            entities.DocumentType(d.Type),
		Purpose:         d.Purpose.String,
		IsPublic:        d.IsPublic,
		AccessToken:     d.AccessToken.String,
		Metadata:        metadata,
		CreatedAt:       d.CreatedAt.Time,
		UpdatedAt:       d.UpdatedAt.Time,
	}
}