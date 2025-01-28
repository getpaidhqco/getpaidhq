package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewCustomerRepository(database lib.Database, logger lib.Logger) repositories.CustomerRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CustomerRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CustomerRepository) FindById(ctx context.Context, id string) (entities.Customer, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Tx.Query(ctx, query, id)

	var user entities.Customer
	err := row.Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		return entities.Customer{}, err
	}
	return user, nil
}

func (r CustomerRepository) Create(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	var customer entities.Customer
	query := `INSERT INTO customers (org_id, id, email, name, created_at, updated_at) 
		VALUES (@org_id, @id, @email, @name, now(), now())
		RETURNING (org_id, id, name, email)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": entity.OrgId,
		"id":     entity.ID,
		"email":  entity.Email,
		"name":   entity.Name,
	}).Scan(&customer)

	if err != nil {
		return entities.Customer{}, err
	}

	return customer, nil
}
