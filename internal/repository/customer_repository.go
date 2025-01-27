package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
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

func (r *CustomerRepository) FindByID(ctx context.Context, id uint) (*entities.User, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Tx.Query(ctx, query, id)

	var user entities.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *CustomerRepository) FindAll(ctx context.Context) ([]*entities.User, error) {
	query := ``
	rows, err := r.Tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		var user entities.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *CustomerRepository) Create(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	var customer entities.Customer
	query := `INSERT INTO customers (org_id, id, email, name, created_at, updated_at) 
		VALUES (@org_id, @id, @email, @name, now(), now())
		RETURNING (org_id, id, email, name)`

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
