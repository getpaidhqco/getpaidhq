package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type InvoiceRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewInvoiceRepository(primaryDb lib.Database, logger logger.Logger) repositories.InvoiceRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return InvoiceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r InvoiceRepository) WithTrx(trxHandle interface{}) InvoiceRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r InvoiceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Invoice, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, delivered_at, voided_at, marked_uncollectible_at,
              notes, customer_notes, metadata, exchange_rate, base_currency, created_at, updated_at
			  FROM invoices
			  WHERE org_id = @org_id AND id = @id`

	var invoiceModel models.Invoice

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&invoiceModel.OrgId,
		&invoiceModel.Id,
		&invoiceModel.CustomerId,
		&invoiceModel.OrderId,
		&invoiceModel.SubscriptionId,
		&invoiceModel.SequenceId,
		&invoiceModel.DocNumber,
		&invoiceModel.Type,
		&invoiceModel.InvoiceType,
		&invoiceModel.Status,
		&invoiceModel.IsImmutable,
		&invoiceModel.Currency,
		&invoiceModel.SubTotal,
		&invoiceModel.TaxTotal,
		&invoiceModel.DiscountTotal,
		&invoiceModel.Total,
		&invoiceModel.AmountPaid,
		&invoiceModel.AmountDue,
		&invoiceModel.TaxProvider,
		&invoiceModel.TaxTransactionId,
		&invoiceModel.TaxBreakdown,
		&invoiceModel.IssuedAt,
		&invoiceModel.DueAt,
		&invoiceModel.PaidAt,
		&invoiceModel.DeliveredAt,
		&invoiceModel.VoidedAt,
		&invoiceModel.MarkedUncollectibleAt,
		&invoiceModel.Notes,
		&invoiceModel.CustomerNotes,
		&invoiceModel.Metadata,
		&invoiceModel.ExchangeRate,
		&invoiceModel.BaseCurrency,
		&invoiceModel.CreatedAt,
		&invoiceModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Invoice{}, err
		}
		r.logger.Error(`failed to find Invoice by id`, err.Error())
		return entities.Invoice{}, err
	}

	// Convert model to entity
	invoice := invoiceModel.ToEntity()

	// Fetch line items for the invoice
	lineItems, err := r.ListLineItems(ctx, orgId, id)
	if err != nil {
		r.logger.Error(`failed to fetch line items for invoice`, err.Error())
		return entities.Invoice{}, err
	}

	// Attach line items to the invoice
	invoice.LineItems = lineItems

	return invoice, nil
}

func (r InvoiceRepository) Create(ctx context.Context, entity entities.Invoice) (entities.Invoice, error) {
	tx := r.getTransactionFromContext(ctx)

	// Store line items for later
	lineItems := entity.LineItems

	// Create the invoice without line items first
	query := `INSERT INTO invoices (org_id, id, customer_id, order_id, subscription_id, sequence_id, 
              doc_number, type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, delivered_at, voided_at, marked_uncollectible_at,
              notes, customer_notes, metadata, exchange_rate, base_currency, created_at, updated_at) 
			  VALUES (@org_id, @id, @customer_id, @order_id, @subscription_id, @sequence_id, 
              @doc_number, @type, @invoice_type, @status, @is_immutable, @currency, @sub_total, @tax_total, 
              @discount_total, @total, @amount_paid, @amount_due, @tax_provider, @tax_transaction_id, 
              @tax_breakdown, @issued_at, @due_at, @paid_at, @delivered_at, @voided_at, @marked_uncollectible_at,
              @notes, @customer_notes, @metadata, @exchange_rate, @base_currency, NOW(), NOW())`

	taxBreakdownJson, _ := json.Marshal(entity.TaxBreakdown)
	metadataJson, _ := json.Marshal(entity.Metadata)

	// Create a model from the entity
	customerIdText := pgtype.Text{String: entity.CustomerId, Valid: entity.CustomerId != ""}
	orderIdText := pgtype.Text{String: entity.OrderId, Valid: entity.OrderId != ""}
	subscriptionIdText := pgtype.Text{String: entity.SubscriptionId, Valid: entity.SubscriptionId != ""}
	taxProviderText := pgtype.Text{String: entity.TaxProvider, Valid: entity.TaxProvider != ""}
	taxTransactionIdText := pgtype.Text{String: entity.TaxTransactionId, Valid: entity.TaxTransactionId != ""}
	notesText := pgtype.Text{String: entity.Notes, Valid: entity.Notes != ""}
	customerNotesText := pgtype.Text{String: entity.CustomerNotes, Valid: entity.CustomerNotes != ""}
	baseCurrencyText := pgtype.Text{String: entity.BaseCurrency, Valid: entity.BaseCurrency != ""}

	issuedAtTimestamp := pgtype.Timestamptz{}
	if !entity.IssuedAt.IsZero() {
		issuedAtTimestamp.Time = entity.IssuedAt
		issuedAtTimestamp.Valid = true
	}

	dueAtTimestamp := pgtype.Timestamptz{}
	if !entity.DueAt.IsZero() {
		dueAtTimestamp.Time = entity.DueAt
		dueAtTimestamp.Valid = true
	}

	paidAtTimestamp := pgtype.Timestamptz{}
	if !entity.PaidAt.IsZero() {
		paidAtTimestamp.Time = entity.PaidAt
		paidAtTimestamp.Valid = true
	}

	deliveredAtTimestamp := pgtype.Timestamptz{}
	if !entity.DeliveredAt.IsZero() {
		deliveredAtTimestamp.Time = entity.DeliveredAt
		deliveredAtTimestamp.Valid = true
	}

	voidedAtTimestamp := pgtype.Timestamptz{}
	if !entity.VoidedAt.IsZero() {
		voidedAtTimestamp.Time = entity.VoidedAt
		voidedAtTimestamp.Valid = true
	}

	markedUncollectibleAtTimestamp := pgtype.Timestamptz{}
	if !entity.MarkedUncollectibleAt.IsZero() {
		markedUncollectibleAtTimestamp.Time = entity.MarkedUncollectibleAt
		markedUncollectibleAtTimestamp.Valid = true
	}

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":             entity.OrgId,
		"id":                 entity.Id,
		"customer_id":        customerIdText,
		"order_id":           orderIdText,
		"subscription_id":    subscriptionIdText,
		"sequence_id":        entity.SequenceId,
		"doc_number":         entity.DocNumber,
		"type":               string(entity.Type),
		"invoice_type":       string(entity.InvoiceType),
		"status":             string(entity.Status),
		"is_immutable":       entity.IsImmutable,
		"currency":           entity.Currency,
		"sub_total":          entity.SubTotal,
		"tax_total":          entity.TaxTotal,
		"discount_total":     entity.DiscountTotal,
		"total":              entity.Total,
		"amount_paid":        entity.AmountPaid,
		"amount_due":         entity.AmountDue,
		"tax_provider":       taxProviderText,
		"tax_transaction_id": taxTransactionIdText,
		"tax_breakdown":      taxBreakdownJson,
		"issued_at":          issuedAtTimestamp,
		"due_at":             dueAtTimestamp,
		"paid_at":            paidAtTimestamp,
		"delivered_at":       deliveredAtTimestamp,
		"voided_at":          voidedAtTimestamp,
		"marked_uncollectible_at": markedUncollectibleAtTimestamp,
		"notes":              notesText,
		"customer_notes":     customerNotesText,
		"metadata":           metadataJson,
		"exchange_rate":      entity.ExchangeRate,
		"base_currency":      baseCurrencyText,
	})

	if err != nil {
		r.logger.Error(`failed to insert Invoice`, err.Error())
		return entities.Invoice{}, err
	}

	// Get the created invoice
	createdInvoice, err := r.FindById(ctx, entity.OrgId, entity.Id)
	if err != nil {
		return entities.Invoice{}, err
	}

	// If there are line items, create them
	if len(lineItems) > 0 {
		var createdLineItems []entities.InvoiceLineItem
		for _, lineItem := range lineItems {
			lineItem.InvoiceId = createdInvoice.Id
			lineItem.OrgId = createdInvoice.OrgId

			createdLineItem, err := r.AddLineItem(ctx, lineItem)
			if err != nil {
				return entities.Invoice{}, err
			}
			createdLineItems = append(createdLineItems, createdLineItem)
		}

		// Attach line items to the created invoice
		createdInvoice.LineItems = createdLineItems

		// Recalculate totals based on line items
		createdInvoice.RecalculateTotals()

		// Update the invoice with recalculated totals
		updatedInvoice, err := r.Update(ctx, createdInvoice)
		if err != nil {
			return entities.Invoice{}, err
		}

		// Return the complete invoice with line items
		updatedInvoice.LineItems = createdLineItems
		return updatedInvoice, nil
	}

	return createdInvoice, nil
}

func (r InvoiceRepository) Update(ctx context.Context, entity entities.Invoice) (entities.Invoice, error) {
	tx := r.getTransactionFromContext(ctx)

	// Get existing invoice with line items
	existingInvoice, err := r.FindById(ctx, entity.OrgId, entity.Id)
	if err != nil {
		return entities.Invoice{}, err
	}

	// Update the base invoice
	query := `UPDATE invoices
			  SET customer_id = @customer_id, order_id = @order_id, subscription_id = @subscription_id, 
              sequence_id = @sequence_id, doc_number = @doc_number, type = @type, invoice_type = @invoice_type, 
              status = @status, is_immutable = @is_immutable, currency = @currency, sub_total = @sub_total, 
              tax_total = @tax_total, discount_total = @discount_total, total = @total, amount_paid = @amount_paid, 
              amount_due = @amount_due, tax_provider = @tax_provider, tax_transaction_id = @tax_transaction_id, 
              tax_breakdown = @tax_breakdown, issued_at = @issued_at, due_at = @due_at, paid_at = @paid_at, 
              notes = @notes, customer_notes = @customer_notes, metadata = @metadata, 
              exchange_rate = @exchange_rate, base_currency = @base_currency, updated_at = NOW()
			  WHERE org_id = @org_id AND id = @id`

	taxBreakdownJson, _ := json.Marshal(entity.TaxBreakdown)
	metadataJson, _ := json.Marshal(entity.Metadata)

	// Create pgtype values from the entity
	customerIdText := pgtype.Text{String: entity.CustomerId, Valid: entity.CustomerId != ""}
	orderIdText := pgtype.Text{String: entity.OrderId, Valid: entity.OrderId != ""}
	subscriptionIdText := pgtype.Text{String: entity.SubscriptionId, Valid: entity.SubscriptionId != ""}
	taxProviderText := pgtype.Text{String: entity.TaxProvider, Valid: entity.TaxProvider != ""}
	taxTransactionIdText := pgtype.Text{String: entity.TaxTransactionId, Valid: entity.TaxTransactionId != ""}
	notesText := pgtype.Text{String: entity.Notes, Valid: entity.Notes != ""}
	customerNotesText := pgtype.Text{String: entity.CustomerNotes, Valid: entity.CustomerNotes != ""}
	baseCurrencyText := pgtype.Text{String: entity.BaseCurrency, Valid: entity.BaseCurrency != ""}

	issuedAtTimestamp := pgtype.Timestamptz{}
	if !entity.IssuedAt.IsZero() {
		issuedAtTimestamp.Time = entity.IssuedAt
		issuedAtTimestamp.Valid = true
	}

	dueAtTimestamp := pgtype.Timestamptz{}
	if !entity.DueAt.IsZero() {
		dueAtTimestamp.Time = entity.DueAt
		dueAtTimestamp.Valid = true
	}

	paidAtTimestamp := pgtype.Timestamptz{}
	if !entity.PaidAt.IsZero() {
		paidAtTimestamp.Time = entity.PaidAt
		paidAtTimestamp.Valid = true
	}

	deliveredAtTimestamp := pgtype.Timestamptz{}
	if !entity.DeliveredAt.IsZero() {
		deliveredAtTimestamp.Time = entity.DeliveredAt
		deliveredAtTimestamp.Valid = true
	}

	voidedAtTimestamp := pgtype.Timestamptz{}
	if !entity.VoidedAt.IsZero() {
		voidedAtTimestamp.Time = entity.VoidedAt
		voidedAtTimestamp.Valid = true
	}

	markedUncollectibleAtTimestamp := pgtype.Timestamptz{}
	if !entity.MarkedUncollectibleAt.IsZero() {
		markedUncollectibleAtTimestamp.Time = entity.MarkedUncollectibleAt
		markedUncollectibleAtTimestamp.Valid = true
	}

	_, err = tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":             entity.OrgId,
		"id":                 entity.Id,
		"customer_id":        customerIdText,
		"order_id":           orderIdText,
		"subscription_id":    subscriptionIdText,
		"sequence_id":        entity.SequenceId,
		"doc_number":         entity.DocNumber,
		"type":               string(entity.Type),
		"invoice_type":       string(entity.InvoiceType),
		"status":             string(entity.Status),
		"is_immutable":       entity.IsImmutable,
		"currency":           entity.Currency,
		"sub_total":          entity.SubTotal,
		"tax_total":          entity.TaxTotal,
		"discount_total":     entity.DiscountTotal,
		"total":              entity.Total,
		"amount_paid":        entity.AmountPaid,
		"amount_due":         entity.AmountDue,
		"tax_provider":       taxProviderText,
		"tax_transaction_id": taxTransactionIdText,
		"tax_breakdown":      taxBreakdownJson,
		"issued_at":          issuedAtTimestamp,
		"due_at":             dueAtTimestamp,
		"paid_at":            paidAtTimestamp,
		"delivered_at":       deliveredAtTimestamp,
		"voided_at":          voidedAtTimestamp,
		"marked_uncollectible_at": markedUncollectibleAtTimestamp,
		"notes":              notesText,
		"customer_notes":     customerNotesText,
		"metadata":           metadataJson,
		"exchange_rate":      entity.ExchangeRate,
		"base_currency":      baseCurrencyText,
	})

	if err != nil {
		r.logger.Error(`failed to update Invoice`, err.Error())
		return entities.Invoice{}, err
	}

	// Get the updated invoice
	updatedInvoice, err := r.FindById(ctx, entity.OrgId, entity.Id)
	if err != nil {
		return entities.Invoice{}, err
	}

	// Handle line items if provided
	if entity.LineItems != nil {
		// Create maps for easier lookup
		existingItems := make(map[string]entities.InvoiceLineItem)
		for _, item := range existingInvoice.LineItems {
			existingItems[item.Id] = item
		}

		newItems := make(map[string]entities.InvoiceLineItem)
		for _, item := range entity.LineItems {
			newItems[item.Id] = item
		}

		// Update or create line items
		var finalLineItems []entities.InvoiceLineItem
		for _, item := range entity.LineItems {
			if item.Id == "" {
				// New item - create it
				item.InvoiceId = updatedInvoice.Id
				item.OrgId = updatedInvoice.OrgId
				createdItem, err := r.AddLineItem(ctx, item)
				if err != nil {
					return entities.Invoice{}, err
				}
				finalLineItems = append(finalLineItems, createdItem)
			} else if _, exists := existingItems[item.Id]; exists {
				// Existing item - update it
				updatedItem, err := r.UpdateLineItem(ctx, item)
				if err != nil {
					return entities.Invoice{}, err
				}
				finalLineItems = append(finalLineItems, updatedItem)
			}
		}

		// Delete items that are no longer present
		for _, existingItem := range existingInvoice.LineItems {
			if _, stillExists := newItems[existingItem.Id]; !stillExists {
				err := r.DeleteLineItem(ctx, updatedInvoice.OrgId, updatedInvoice.Id, existingItem.Id)
				if err != nil {
					return entities.Invoice{}, err
				}
			}
		}

		// Attach final line items
		updatedInvoice.LineItems = finalLineItems
		return updatedInvoice, nil
	}

	// If no line items provided, just return the updated invoice with existing line items
	updatedInvoice.LineItems = existingInvoice.LineItems
	return updatedInvoice, nil
}

func (r InvoiceRepository) List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var invoices = make([]entities.Invoice, 0)
	var count int

	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, notes, customer_notes, metadata, 
              exchange_rate, base_currency, created_at, updated_at,
              count(*) OVER()
			  FROM invoices
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
		r.logger.Error(`failed to list Invoices`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var invoiceModel models.Invoice

		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	// Then get line items for each invoice
	for i := range invoices {
		lineItems, err := r.ListLineItems(ctx, orgId, invoices[i].Id)
		if err != nil {
			// Log error but continue with other invoices
			r.logger.Error("failed to load line items for invoice", "invoice_id", invoices[i].Id, "error", err.Error())
			continue
		}
		invoices[i].LineItems = lineItems
	}

	return invoices, count, nil
}

func (r InvoiceRepository) FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var invoices = make([]entities.Invoice, 0)
	var count int

	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, notes, customer_notes, metadata, 
              exchange_rate, base_currency, created_at, updated_at,
              count(*) OVER()
			  FROM invoices
			  WHERE org_id = @org_id AND customer_id = @customer_id
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
		"org_id":      orgId,
		"customer_id": customerId,
		"lim":         pagination.Limit,
		"off":         pagination.Offset,
		"sort_col":    pagination.SortBy,
		"sort_dir":    pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to find Invoices by customer_id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var invoiceModel models.Invoice

		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	// Then get line items for each invoice
	for i := range invoices {
		lineItems, err := r.ListLineItems(ctx, orgId, invoices[i].Id)
		if err != nil {
			// Log error but continue with other invoices
			r.logger.Error("failed to load line items for invoice", "invoice_id", invoices[i].Id, "error", err.Error())
			continue
		}
		invoices[i].LineItems = lineItems
	}

	return invoices, count, nil
}

func (r InvoiceRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var invoices = make([]entities.Invoice, 0)
	var count int

	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, notes, customer_notes, metadata, 
              exchange_rate, base_currency, created_at, updated_at,
              count(*) OVER()
			  FROM invoices
			  WHERE org_id = @org_id AND order_id = @order_id
			  ORDER BY created_at DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"order_id": orderId,
	})

	if err != nil {
		r.logger.Error(`failed to find Invoices by order_id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var invoiceModel models.Invoice

		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	// Then get line items for each invoice
	for i := range invoices {
		lineItems, err := r.ListLineItems(ctx, orgId, invoices[i].Id)
		if err != nil {
			// Log error but continue with other invoices
			r.logger.Error("failed to load line items for invoice", "invoice_id", invoices[i].Id, "error", err.Error())
			continue
		}
		invoices[i].LineItems = lineItems
	}

	return invoices, count, nil
}

func (r InvoiceRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var invoices = make([]entities.Invoice, 0)
	var count int

	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, notes, customer_notes, metadata, 
              exchange_rate, base_currency, created_at, updated_at,
              count(*) OVER()
			  FROM invoices
			  WHERE org_id = @org_id AND subscription_id = @subscription_id
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
		"org_id":          orgId,
		"subscription_id": subscriptionId,
		"lim":             pagination.Limit,
		"off":             pagination.Offset,
		"sort_col":        pagination.SortBy,
		"sort_dir":        pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to find Invoices by subscription_id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var invoiceModel models.Invoice

		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	// Then get line items for each invoice
	for i := range invoices {
		lineItems, err := r.ListLineItems(ctx, orgId, invoices[i].Id)
		if err != nil {
			// Log error but continue with other invoices
			r.logger.Error("failed to load line items for invoice", "invoice_id", invoices[i].Id, "error", err.Error())
			continue
		}
		invoices[i].LineItems = lineItems
	}

	return invoices, count, nil
}

// Line items
func (r InvoiceRepository) AddLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO invoice_line_items (org_id, invoice_id, id, product_id, variant_id, price_id, 
              description, category, quantity, unit_price, line_total, discount_type, discount_value, 
              discount_total, tax_code, tax_rate, tax_amount, tax_exempt, seq, metadata, created_at, updated_at) 
			  VALUES (@org_id, @invoice_id, @id, @product_id, @variant_id, @price_id, 
              @description, @category, @quantity, @unit_price, @line_total, @discount_type, @discount_value, 
              @discount_total, @tax_code, @tax_rate, @tax_amount, @tax_exempt, @seq, @metadata, NOW(), NOW())`

	// Create a model from the entity
	lineItemModel := models.InvoiceLineItem{
		OrgId:         lineItem.OrgId,
		InvoiceId:     lineItem.InvoiceId,
		Id:            lineItem.Id,
		ProductId:     pgtype.Text{String: lineItem.ProductId, Valid: lineItem.ProductId != ""},
		VariantId:     pgtype.Text{String: lineItem.VariantId, Valid: lineItem.VariantId != ""},
		PriceId:       pgtype.Text{String: lineItem.PriceId, Valid: lineItem.PriceId != ""},
		Description:   lineItem.Description,
		Category:      pgtype.Text{String: lineItem.Category, Valid: lineItem.Category != ""},
		Quantity:      lineItem.Quantity,
		UnitPrice:     lineItem.UnitPrice,
		LineTotal:     lineItem.LineTotal,
		DiscountType:  pgtype.Text{String: lineItem.DiscountType, Valid: lineItem.DiscountType != ""},
		DiscountValue: lineItem.DiscountValue,
		DiscountTotal: lineItem.DiscountTotal,
		TaxCode:       pgtype.Text{String: lineItem.TaxCode, Valid: lineItem.TaxCode != ""},
		TaxRate:       lineItem.TaxRate,
		TaxAmount:     lineItem.TaxAmount,
		TaxExempt:     lineItem.TaxExempt,
		Seq:           lineItem.Seq,
	}

	metadataJson, _ := json.Marshal(lineItem.Metadata)
	lineItemModel.Metadata = metadataJson

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":         lineItemModel.OrgId,
		"invoice_id":     lineItemModel.InvoiceId,
		"id":             lineItemModel.Id,
		"product_id":     lineItemModel.ProductId,
		"variant_id":     lineItemModel.VariantId,
		"price_id":       lineItemModel.PriceId,
		"description":    lineItemModel.Description,
		"category":       lineItemModel.Category,
		"quantity":       lineItemModel.Quantity,
		"unit_price":     lineItemModel.UnitPrice,
		"line_total":     lineItemModel.LineTotal,
		"discount_type":  lineItemModel.DiscountType,
		"discount_value": lineItemModel.DiscountValue,
		"discount_total": lineItemModel.DiscountTotal,
		"tax_code":       lineItemModel.TaxCode,
		"tax_rate":       lineItemModel.TaxRate,
		"tax_amount":     lineItemModel.TaxAmount,
		"tax_exempt":     lineItemModel.TaxExempt,
		"seq":            lineItemModel.Seq,
		"metadata":       lineItemModel.Metadata,
	})

	if err != nil {
		r.logger.Error(`failed to insert Invoice Line Item`, err.Error())
		return entities.InvoiceLineItem{}, err
	}

	return r.findLineItemById(ctx, lineItem.OrgId, lineItem.InvoiceId, lineItem.Id)
}

func (r InvoiceRepository) UpdateLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE invoice_line_items
			  SET product_id = @product_id, variant_id = @variant_id, price_id = @price_id, 
              description = @description, category = @category, quantity = @quantity, 
              unit_price = @unit_price, line_total = @line_total, discount_type = @discount_type, 
              discount_value = @discount_value, discount_total = @discount_total, tax_code = @tax_code, 
              tax_rate = @tax_rate, tax_amount = @tax_amount, tax_exempt = @tax_exempt, 
              seq = @seq, metadata = @metadata, updated_at = NOW()
			  WHERE org_id = @org_id AND invoice_id = @invoice_id AND id = @id`

	metadataJson, _ := json.Marshal(lineItem.Metadata)

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":         lineItem.OrgId,
		"invoice_id":     lineItem.InvoiceId,
		"id":             lineItem.Id,
		"product_id":     pgtype.Text{String: lineItem.ProductId, Valid: lineItem.ProductId != ""},
		"variant_id":     pgtype.Text{String: lineItem.VariantId, Valid: lineItem.VariantId != ""},
		"price_id":       pgtype.Text{String: lineItem.PriceId, Valid: lineItem.PriceId != ""},
		"description":    lineItem.Description,
		"category":       pgtype.Text{String: lineItem.Category, Valid: lineItem.Category != ""},
		"quantity":       lineItem.Quantity,
		"unit_price":     lineItem.UnitPrice,
		"line_total":     lineItem.LineTotal,
		"discount_type":  pgtype.Text{String: lineItem.DiscountType, Valid: lineItem.DiscountType != ""},
		"discount_value": lineItem.DiscountValue,
		"discount_total": lineItem.DiscountTotal,
		"tax_code":       pgtype.Text{String: lineItem.TaxCode, Valid: lineItem.TaxCode != ""},
		"tax_rate":       lineItem.TaxRate,
		"tax_amount":     lineItem.TaxAmount,
		"tax_exempt":     lineItem.TaxExempt,
		"seq":            lineItem.Seq,
		"metadata":       metadataJson,
	})

	if err != nil {
		r.logger.Error(`failed to update Invoice Line Item`, err.Error())
		return entities.InvoiceLineItem{}, err
	}

	return r.findLineItemById(ctx, lineItem.OrgId, lineItem.InvoiceId, lineItem.Id)
}

func (r InvoiceRepository) DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM invoice_line_items
			  WHERE org_id = $1 AND invoice_id = $2 AND id = $3`

	_, err := tx.Exec(ctx, query, orgId, invoiceId, lineItemId)

	if err != nil {
		r.logger.Error(`failed to delete Invoice Line Item`, err.Error())
		return err
	}

	return nil
}

func (r InvoiceRepository) ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error) {
	tx := r.getTransactionFromContext(ctx)

	var lineItems = make([]entities.InvoiceLineItem, 0)

	query := `SELECT org_id, invoice_id, id, product_id, variant_id, price_id, description, category, 
              quantity, unit_price, line_total, discount_type, discount_value, discount_total, 
              tax_code, tax_rate, tax_amount, tax_exempt, seq, metadata, created_at, updated_at
			  FROM invoice_line_items
			  WHERE org_id = $1 AND invoice_id = $2
			  ORDER BY seq ASC`

	rows, err := tx.Query(ctx, query, orgId, invoiceId)

	if err != nil {
		r.logger.Error(`failed to find Invoice Line Items`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lineItemModel models.InvoiceLineItem

		err := rows.Scan(
			&lineItemModel.OrgId,
			&lineItemModel.InvoiceId,
			&lineItemModel.Id,
			&lineItemModel.ProductId,
			&lineItemModel.VariantId,
			&lineItemModel.PriceId,
			&lineItemModel.Description,
			&lineItemModel.Category,
			&lineItemModel.Quantity,
			&lineItemModel.UnitPrice,
			&lineItemModel.LineTotal,
			&lineItemModel.DiscountType,
			&lineItemModel.DiscountValue,
			&lineItemModel.DiscountTotal,
			&lineItemModel.TaxCode,
			&lineItemModel.TaxRate,
			&lineItemModel.TaxAmount,
			&lineItemModel.TaxExempt,
			&lineItemModel.Seq,
			&lineItemModel.Metadata,
			&lineItemModel.CreatedAt,
			&lineItemModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice Line Item`, err.Error())
			return nil, err
		}

		// Convert model to entity
		lineItem := lineItemModel.ToEntity()
		lineItems = append(lineItems, lineItem)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return lineItems, nil
}

// Invoice history
func (r InvoiceRepository) AddHistory(ctx context.Context, history entities.InvoiceHistory) (entities.InvoiceHistory, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO invoice_history (org_id, id, invoice_id, action, field, old_value, new_value, 
              user_id, user_email, ip_address, user_agent, reason, metadata, timestamp) 
			  VALUES (@org_id, @id, @invoice_id, @action, @field, @old_value, @new_value, 
              @user_id, @user_email, @ip_address, @user_agent, @reason, @metadata, @timestamp)`

	oldValueJson, _ := json.Marshal(history.OldValue)
	newValueJson, _ := json.Marshal(history.NewValue)
	metadataJson, _ := json.Marshal(history.Metadata)

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":     history.OrgId,
		"id":         history.Id,
		"invoice_id": history.InvoiceId,
		"action":     history.Action,
		"field":      pgtype.Text{String: history.Field, Valid: history.Field != ""},
		"old_value":  oldValueJson,
		"new_value":  newValueJson,
		"user_id":    pgtype.Text{String: history.UserId, Valid: history.UserId != ""},
		"user_email": pgtype.Text{String: history.UserEmail, Valid: history.UserEmail != ""},
		"ip_address": pgtype.Text{String: history.IpAddress, Valid: history.IpAddress != ""},
		"user_agent": pgtype.Text{String: history.UserAgent, Valid: history.UserAgent != ""},
		"reason":     pgtype.Text{String: history.Reason, Valid: history.Reason != ""},
		"metadata":   metadataJson,
		"timestamp":  history.Timestamp,
	})

	if err != nil {
		r.logger.Error(`failed to insert Invoice History`, err.Error())
		return entities.InvoiceHistory{}, err
	}

	return r.findHistoryById(ctx, history.OrgId, history.InvoiceId, history.Id)
}

func (r InvoiceRepository) ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error) {
	tx := r.getTransactionFromContext(ctx)

	var histories = make([]entities.InvoiceHistory, 0)

	query := `SELECT org_id, id, invoice_id, action, field, old_value, new_value, 
              user_id, user_email, ip_address, user_agent, reason, metadata, timestamp
			  FROM invoice_history
			  WHERE org_id = $1 AND invoice_id = $2
			  ORDER BY timestamp DESC`

	rows, err := tx.Query(ctx, query, orgId, invoiceId)

	if err != nil {
		r.logger.Error(`failed to find Invoice History`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var historyModel models.InvoiceHistory

		err := rows.Scan(
			&historyModel.OrgId,
			&historyModel.Id,
			&historyModel.InvoiceId,
			&historyModel.Action,
			&historyModel.Field,
			&historyModel.OldValue,
			&historyModel.NewValue,
			&historyModel.UserId,
			&historyModel.UserEmail,
			&historyModel.IpAddress,
			&historyModel.UserAgent,
			&historyModel.Reason,
			&historyModel.Metadata,
			&historyModel.Timestamp,
		)

		if err != nil {
			r.logger.Error(`failed to scan Invoice History`, err.Error())
			return nil, err
		}

		// Convert model to entity
		history := historyModel.ToEntity()
		histories = append(histories, history)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return histories, nil
}

// Helper methods
func (r InvoiceRepository) findLineItemById(ctx context.Context, orgId string, invoiceId string, lineItemId string) (entities.InvoiceLineItem, error) {
	tx := r.getTransactionFromContext(ctx)

	var lineItemModel models.InvoiceLineItem

	query := `SELECT org_id, invoice_id, id, product_id, variant_id, price_id, description, category, 
              quantity, unit_price, line_total, discount_type, discount_value, discount_total, 
              tax_code, tax_rate, tax_amount, tax_exempt, seq, metadata, created_at, updated_at
			  FROM invoice_line_items
			  WHERE org_id = $1 AND invoice_id = $2 AND id = $3`

	err := tx.QueryRow(ctx, query, orgId, invoiceId, lineItemId).Scan(
		&lineItemModel.OrgId,
		&lineItemModel.InvoiceId,
		&lineItemModel.Id,
		&lineItemModel.ProductId,
		&lineItemModel.VariantId,
		&lineItemModel.PriceId,
		&lineItemModel.Description,
		&lineItemModel.Category,
		&lineItemModel.Quantity,
		&lineItemModel.UnitPrice,
		&lineItemModel.LineTotal,
		&lineItemModel.DiscountType,
		&lineItemModel.DiscountValue,
		&lineItemModel.DiscountTotal,
		&lineItemModel.TaxCode,
		&lineItemModel.TaxRate,
		&lineItemModel.TaxAmount,
		&lineItemModel.TaxExempt,
		&lineItemModel.Seq,
		&lineItemModel.Metadata,
		&lineItemModel.CreatedAt,
		&lineItemModel.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			r.logger.Error("failed to find Invoice Line Item", "err", pgErr.Message, "code", pgErr.Code)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("Invoice Line Item not found")
		}
		return entities.InvoiceLineItem{}, err
	}

	// Convert model to entity
	lineItem := lineItemModel.ToEntity()
	return lineItem, nil
}

func (r InvoiceRepository) findHistoryById(ctx context.Context, orgId string, invoiceId string, historyId string) (entities.InvoiceHistory, error) {
	tx := r.getTransactionFromContext(ctx)

	var historyModel models.InvoiceHistory

	query := `SELECT org_id, id, invoice_id, action, field, old_value, new_value, 
              user_id, user_email, ip_address, user_agent, reason, metadata, timestamp
			  FROM invoice_history
			  WHERE org_id = $1 AND invoice_id = $2 AND id = $3`

	err := tx.QueryRow(ctx, query, orgId, invoiceId, historyId).Scan(
		&historyModel.OrgId,
		&historyModel.Id,
		&historyModel.InvoiceId,
		&historyModel.Action,
		&historyModel.Field,
		&historyModel.OldValue,
		&historyModel.NewValue,
		&historyModel.UserId,
		&historyModel.UserEmail,
		&historyModel.IpAddress,
		&historyModel.UserAgent,
		&historyModel.Reason,
		&historyModel.Metadata,
		&historyModel.Timestamp,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			r.logger.Error("failed to find Invoice History", "err", pgErr.Message, "code", pgErr.Code)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("Invoice History not found")
		}
		return entities.InvoiceHistory{}, err
	}

	// Convert model to entity
	history := historyModel.ToEntity()
	return history, nil
}


// FindByStatus finds invoices by status with pagination
func (r InvoiceRepository) FindByStatus(ctx context.Context, orgId string, status entities.InvoiceStatus, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	// Count query
	countQuery := `SELECT COUNT(*) FROM invoices WHERE org_id = @org_id AND status = @status`
	var totalCount int
	err := tx.QueryRow(ctx, countQuery, pgx.NamedArgs{
		"org_id": orgId,
		"status": string(status),
	}).Scan(&totalCount)

	if err != nil {
		r.logger.Error("failed to count invoices by status", "err", err.Error())
		return nil, 0, err
	}

	// Main query with pagination
	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, delivered_at, voided_at, marked_uncollectible_at,
              notes, customer_notes, metadata, exchange_rate, base_currency, created_at, updated_at
			  FROM invoices 
			  WHERE org_id = @org_id AND status = @status
			  ORDER BY created_at DESC
			  LIMIT @limit OFFSET @offset`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"status": string(status),
		"limit":  pagination.Limit,
		"offset": pagination.Offset,
	})

	if err != nil {
		r.logger.Error("failed to find invoices by status", "err", err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []entities.Invoice
	for rows.Next() {
		var invoiceModel models.Invoice
		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.DeliveredAt,
			&invoiceModel.VoidedAt,
			&invoiceModel.MarkedUncollectibleAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan invoice", "err", err.Error())
			return nil, 0, err
		}

		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	return invoices, totalCount, nil
}

// FindOverdueInvoices finds all overdue invoices
func (r InvoiceRepository) FindOverdueInvoices(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Invoice, int, error) {
	tx := r.getTransactionFromContext(ctx)

	// Count query
	countQuery := `SELECT COUNT(*) FROM invoices 
				   WHERE org_id = @org_id 
				   AND (status = 'overdue' OR (status = 'open' AND due_at < NOW()))`
	var totalCount int
	err := tx.QueryRow(ctx, countQuery, pgx.NamedArgs{
		"org_id": orgId,
	}).Scan(&totalCount)

	if err != nil {
		r.logger.Error("failed to count overdue invoices", "err", err.Error())
		return nil, 0, err
	}

	// Main query
	query := `SELECT org_id, id, customer_id, order_id, subscription_id, sequence_id, doc_number, 
              type, invoice_type, status, is_immutable, currency, sub_total, tax_total, 
              discount_total, total, amount_paid, amount_due, tax_provider, tax_transaction_id, 
              tax_breakdown, issued_at, due_at, paid_at, delivered_at, voided_at, marked_uncollectible_at,
              notes, customer_notes, metadata, exchange_rate, base_currency, created_at, updated_at
			  FROM invoices 
			  WHERE org_id = @org_id 
			  AND (status = 'overdue' OR (status = 'open' AND due_at < NOW()))
			  ORDER BY due_at ASC
			  LIMIT @limit OFFSET @offset`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"limit":  pagination.Limit,
		"offset": pagination.Offset,
	})

	if err != nil {
		r.logger.Error("failed to find overdue invoices", "err", err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []entities.Invoice
	for rows.Next() {
		var invoiceModel models.Invoice
		err := rows.Scan(
			&invoiceModel.OrgId,
			&invoiceModel.Id,
			&invoiceModel.CustomerId,
			&invoiceModel.OrderId,
			&invoiceModel.SubscriptionId,
			&invoiceModel.SequenceId,
			&invoiceModel.DocNumber,
			&invoiceModel.Type,
			&invoiceModel.InvoiceType,
			&invoiceModel.Status,
			&invoiceModel.IsImmutable,
			&invoiceModel.Currency,
			&invoiceModel.SubTotal,
			&invoiceModel.TaxTotal,
			&invoiceModel.DiscountTotal,
			&invoiceModel.Total,
			&invoiceModel.AmountPaid,
			&invoiceModel.AmountDue,
			&invoiceModel.TaxProvider,
			&invoiceModel.TaxTransactionId,
			&invoiceModel.TaxBreakdown,
			&invoiceModel.IssuedAt,
			&invoiceModel.DueAt,
			&invoiceModel.PaidAt,
			&invoiceModel.DeliveredAt,
			&invoiceModel.VoidedAt,
			&invoiceModel.MarkedUncollectibleAt,
			&invoiceModel.Notes,
			&invoiceModel.CustomerNotes,
			&invoiceModel.Metadata,
			&invoiceModel.ExchangeRate,
			&invoiceModel.BaseCurrency,
			&invoiceModel.CreatedAt,
			&invoiceModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan overdue invoice", "err", err.Error())
			return nil, 0, err
		}

		invoice := invoiceModel.ToEntity()
		invoices = append(invoices, invoice)
	}

	return invoices, totalCount, nil
}
