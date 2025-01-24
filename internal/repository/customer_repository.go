package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"payloop/internal/domain/customers"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"

	"payloop/internal/models"
)

type CustomerRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewCustomerRepository(database lib.Database, logger lib.Logger) CustomerRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CustomerRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *CustomerRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Tx.Query(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *CustomerRepository) FindAll(ctx context.Context) ([]*models.User, error) {
	query := ``
	rows, err := r.Tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *CustomerRepository) Create(ctx context.Context, input customers.CreateCustomerInput) (models.Customer, error) {
	var customer models.Customer
	query := `INSERT INTO customers (acct_id, id, email, name, metadata, created_at, updated_at) 
		VALUES (@acct_id, @id, @email, @name, @metadata, now(), now())`

	metaJson, _ := json.Marshal(input.Metadata)

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acct_id":  input.AccountId,
		"id":       lib.GenerateId("order"),
		"email":    input.Email,
		"name":     input.Name,
		"metadata": metaJson,
	}).Scan(&customer)

	if err != nil {
		return models.Customer{}, err
	}

	return customer, nil
}
