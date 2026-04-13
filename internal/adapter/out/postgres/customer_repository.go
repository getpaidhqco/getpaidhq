package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewCustomerRepository(primaryDb lib.Database, logger port.Logger) port.CustomerRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CustomerRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CustomerRepository) FindById(ctx context.Context, orgId string, id string) (domain.Customer, error) {
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
		return domain.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) FindByEmail(ctx context.Context, orgId string, email string) (domain.Customer, error) {
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
			return domain.Customer{}, nil // or a custom error indicating no rows found
		}
		return domain.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) Create(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
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
		return domain.Customer{}, mapError(err)
	}

	return customer.ToEntity(), nil
}

func (r CustomerRepository) FindPaymentMethodById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	var pm domain.PaymentMethod
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
		return domain.PaymentMethod{}, mapError(err)
	}
	return pm, nil
}

func (r CustomerRepository) Update(ctx context.Context, entity domain.Customer) (domain.Customer, error) {
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
		return domain.Customer{}, mapError(err)
	}

	return updatedCustomer.ToEntity(), nil
}

func (r CustomerRepository) AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (domain.Customer, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO customer_cohorts (org_id, customer_id, cohort_id, cohort_value, created_at, updated_at)
			  VALUES (@org_id, @customer_id, @cohort_id, @cohort_value, now(), now())
			  ON CONFLICT (org_id, customer_id, cohort_id) DO UPDATE SET
			  cohort_value = EXCLUDED.cohort_value,
			  updated_at = now()`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":       orgId,
		"customer_id":  customerId,
		"cohort_id":    cohortId,
		"cohort_value": cohortValue,
	})
	if err != nil {
		r.logger.Error("failed to create or update customer cohort", "orgId", orgId, "customerId", customerId, "cohortId", cohortId, "err", err.Error())
		return domain.Customer{}, mapError(err)
	}

	// Fetch the updated customer entity
	return r.FindById(ctx, orgId, customerId)
}

func (r CustomerRepository) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Customer, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var customers = make([]domain.Customer, 0)
	var count int
	query := `SELECT org_id, id, email, first_name, last_name,
                       phone, billing_address, metadata, default_payment_method_id,
                       created_at, updated_at, count(*) OVER()
              FROM customers
              WHERE org_id = @org_id
              ORDER BY
                -- Handle timestamp columns
                CASE
                    WHEN @sort_col = 'created_at' AND @sort_dir = 'asc' THEN created_at
                    ELSE NULL
                END ASC,
                CASE
                    WHEN @sort_col = 'created_at' AND @sort_dir = 'desc' THEN created_at
                    ELSE NULL
                END DESC,

                -- Handle text columns
                CASE
                    WHEN @sort_col = 'email' AND @sort_dir = 'asc' THEN email
                    ELSE NULL
                END ASC,
                CASE
                    WHEN @sort_col = 'email' AND @sort_dir = 'desc' THEN email
                    ELSE NULL
                END DESC,

                CASE
                    WHEN @sort_col = 'first_name' AND @sort_dir = 'asc' THEN first_name
                    ELSE NULL
                END ASC,
                CASE
                    WHEN @sort_col = 'first_name' AND @sort_dir = 'desc' THEN first_name
                    ELSE NULL
                END DESC,

                CASE
                    WHEN @sort_col = 'last_name' AND @sort_dir = 'asc' THEN last_name
                    ELSE NULL
                END ASC,
                CASE
                    WHEN @sort_col = 'last_name' AND @sort_dir = 'desc' THEN last_name
                    ELSE NULL
                END DESC
              LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Customers`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var customer models.Customer
		err := rows.Scan(
			&customer.OrgId,
			&customer.Id,
			&customer.Email,
			&customer.FirstName,
			&customer.LastName,
			&customer.Phone,
			&customer.BillingAddress,
			&customer.Metadata,
			&customer.DefaultPaymentMethodId,
			&customer.CreatedAt,
			&customer.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Customer`, err.Error())
			return nil, 0, err
		}
		customers = append(customers, customer.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return customers, count, nil
}

// Cohort operations

func (r CustomerRepository) FindCohortById(ctx context.Context, orgId string, id string) (domain.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	var cohort models.Cohort
	query := `SELECT org_id, id, name, type, metadata, created_at, updated_at
				FROM cohorts
				WHERE org_id=@org_id AND id=@id`
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		cohort.OrgId,
		&cohort.Id,
		&cohort.Name,
		&cohort.Type,
		&cohort.Metadata,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Cohort`, "orgId", orgId, "id", id, "err", err.Error())
		return domain.Cohort{}, err
	}
	return cohort.ToEntity(), nil
}

func (r CustomerRepository) CreateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO cohorts (org_id, id, name, type, metadata, created_at, updated_at)
			  VALUES (@org_id, @id, @name, @type, @metadata, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       input.Id,
		"name":     input.Name,
		"type":     input.Type,
		"metadata": input.Metadata,
	})
	if err != nil {
		r.logger.Error(`failed to create Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return domain.Cohort{}, err
	}

	return input, nil
}

func (r CustomerRepository) UpdateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE cohorts SET name=@name, type=@type, metadata=@metadata, updated_at=NOW()
			  WHERE org_id=@org_id AND id=@id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       input.Id,
		"name":     input.Name,
		"type":     input.Type,
		"metadata": input.Metadata,
	})
	if err != nil {
		r.logger.Error(`failed to update Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return domain.Cohort{}, err
	}

	return input, nil
}

func (r CustomerRepository) DeleteCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM cohorts WHERE org_id=@org_id AND id=@id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": input.OrgId,
		"id":     input.Id,
	})
	if err != nil {
		r.logger.Error(`failed to delete Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return domain.Cohort{}, err
	}

	return input, nil
}
