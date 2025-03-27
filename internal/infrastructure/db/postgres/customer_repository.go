package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewCustomerRepository(database lib.Database, logger logger.Logger) repositories.CustomerRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CustomerRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CustomerRepository) FindById(ctx context.Context, orgId string, id string) (entities.Customer, error) {
	tx := r.getTransactionFromContext(ctx)

	var customer models.Customer
	query := `SELECT org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata, 
                       created_at, updated_at
              FROM customers WHERE org_id=@org_id AND id=@id`
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&customer.OrgId,
		&customer.Id,
		&customer.Email,
		&customer.FirstName,
		&customer.LastName,
		&customer.Phone,
		&customer.BillingAddress,
		&customer.Metadata,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		return entities.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) FindByEmail(ctx context.Context, orgId string, email string) (entities.Customer, error) {
	tx := r.getTransactionFromContext(ctx)

	var customer models.Customer
	query := `SELECT org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata,
                       created_at, updated_at
              FROM customers WHERE org_id=@org_id AND email=@email`
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"email":  email,
	}).Scan(
		&customer.OrgId,
		&customer.Id,
		&customer.Email,
		&customer.FirstName,
		&customer.LastName,
		&customer.Phone,
		&customer.BillingAddress,
		&customer.Metadata,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Customer{}, nil // or a custom error indicating no rows found
		}
		return entities.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) Create(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	tx := r.getTransactionFromContext(ctx)

	var customer models.Customer
	query := `INSERT INTO customers (org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata, 
                       created_at, updated_at) 
		VALUES (@org_id, @id, @email, @first_name,@last_name, 
		        @phone, @billing_address, @metadata, 
		        now(), now())
ON CONFLICT (org_id, email) DO UPDATE SET
		email = EXCLUDED.email,
		first_name = EXCLUDED.first_name,
		last_name = EXCLUDED.last_name,
		phone = EXCLUDED.phone,
		billing_address = EXCLUDED.billing_address,
		metadata = EXCLUDED.metadata,
		updated_at = now()
		RETURNING org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata, 
                       created_at, updated_at`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":          entity.OrgId,
		"id":              entity.Id,
		"email":           entity.Email,
		"first_name":      entity.FirstName,
		"last_name":       entity.LastName,
		"phone":           entity.Phone,
		"billing_address": entity.BillingAddress,
		"metadata":        entity.Metadata,
	}).Scan(
		&customer.OrgId,
		&customer.Id,
		&customer.Email,
		&customer.FirstName,
		&customer.LastName,
		&customer.Phone,
		&customer.BillingAddress,
		&customer.Metadata,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		return entities.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) FindPaymentMethodById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	var pm entities.PaymentMethod
	err := tx.QueryRow(ctx, `SELECT org_id,id,token,psp,name,customer_id,details,type FROM payment_methods WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&pm.OrgId,
		&pm.Id,
		&pm.Token,
		&pm.Psp,
		&pm.Name,
		&pm.CustomerId,
		&pm.Details,
		&pm.Type,
	)

	if err != nil {
		r.logger.Error(`failed to find payment method`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.PaymentMethod{}, mapError(err)
	}
	return pm, nil
}

func (r CustomerRepository) Update(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE customers SET email=@Email, 
                     first_name=@FirstName, 
                     last_name=@LastName,
                     default_payment_method_id=@default_payment_method_id,
                     phone=@Phone, 
                     billing_address=@BillingAddress, 
                     metadata=@Metadata,
                     updated_at=now()
              WHERE org_id=@OrgId AND id=@Id
              RETURNING org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata,
                       default_payment_method_id, created_at, updated_at`

	var updatedCustomer models.Customer
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"OrgId":                     entity.OrgId,
		"Id":                        entity.Id,
		"Email":                     entity.Email,
		"FirstName":                 entity.FirstName,
		"LastName":                  entity.LastName,
		"Phone":                     entity.Phone,
		"BillingAddress":            entity.BillingAddress,
		"Metadata":                  entity.Metadata,
		"default_payment_method_id": entity.DefaultPaymentMethodId,
	}).Scan(
		&updatedCustomer.OrgId,
		&updatedCustomer.Id,
		&updatedCustomer.Email,
		&updatedCustomer.FirstName,
		&updatedCustomer.LastName,
		&updatedCustomer.Phone,
		&updatedCustomer.BillingAddress,
		&updatedCustomer.Metadata,
		&updatedCustomer.DefaultPaymentMethodId,
		&updatedCustomer.CreatedAt,
		&updatedCustomer.UpdatedAt,
	)

	if err != nil {
		return entities.Customer{}, mapError(err)
	}

	return updatedCustomer.ToEntity(), nil
}
