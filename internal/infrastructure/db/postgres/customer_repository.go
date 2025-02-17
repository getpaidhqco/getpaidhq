package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	*lib.PgDatabase
	logger logger.Logger
}

func NewCustomerRepository(database lib.Database, logger logger.Logger) repositories.CustomerRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CustomerRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CustomerRepository) FindById(ctx context.Context, orgId string, id string) (entities.Customer, error) {
	var customer entities.Customer
	query := `SELECT org_id, id, email, name, created_at, updated_at FROM customers WHERE org_id=@org_id AND id=@id`
	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&customer.OrgId, &customer.Id, &customer.Email, &customer.Name, &customer.CreatedAt, &customer.UpdatedAt)

	if err != nil {
		return entities.Customer{}, err
	}

	return customer, nil
}

func (r CustomerRepository) Create(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	var p queryRower = r.Pool
	tx := ctx.Value(lib.DBTransaction)
	if tx != nil {
		p = tx.(queryRower)
	}

	var customer entities.Customer
	query := `INSERT INTO customers (org_id, id, email, name, created_at, updated_at) 
		VALUES (@org_id, @id, @email, @name, now(), now())
		RETURNING (org_id, id, name, email)`

	err := p.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": entity.OrgId,
		"id":     entity.Id,
		"email":  entity.Email,
		"name":   entity.Name,
	}).Scan(&customer)

	if err != nil {
		return entities.Customer{}, err
	}

	return customer, nil
}

func (r CustomerRepository) FindPaymentMethodById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	var pm entities.PaymentMethod
	err := r.Pool.QueryRow(ctx, `SELECT org_id,id,token,psp,name,customer_id,is_default,details,type FROM payment_methods WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&pm.OrgId,
		&pm.Id,
		&pm.Token,
		&pm.Psp,
		&pm.Name,
		&pm.CustomerId,
		&pm.IsDefault,
		&pm.Details,
		&pm.Type,
	)

	if err != nil {
		r.logger.Error(`failed to find payment method`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.PaymentMethod{}, err
	}
	return pm, nil
}

func (r CustomerRepository) CreatePaymentMethod(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error) {
	query := `INSERT INTO payment_methods (org_id, id,token, psp,name, customer_id, is_default, details, type, created_at, updated_at)
			  VALUES (@org_id, @id,@token,@psp, @name, @customer_id, @is_default, @details, @type, now(), now())
			  RETURNING org_id, id,token, psp,name, customer_id, is_default, details, type, created_at, updated_at`

	var newEntity entities.PaymentMethod
	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      entity.OrgId,
		"id":          entity.Id,
		"name":        entity.Name,
		"psp":         entity.Psp,
		"token":       entity.Token,
		"customer_id": entity.CustomerId,
		"is_default":  entity.IsDefault,
		"details":     entity.Details,
		"type":        entity.Type,
	}).Scan(
		&newEntity.OrgId,
		&newEntity.Id,
		&newEntity.Token,
		&newEntity.Psp,
		&newEntity.Name,
		&newEntity.CustomerId,
		&newEntity.IsDefault,
		&newEntity.Details,
		&newEntity.Type,
		&newEntity.CreatedAt,
		&newEntity.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to insert PaymentMethod`, err.Error())
		return entities.PaymentMethod{}, err
	}

	return newEntity, nil
}
