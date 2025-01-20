package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"payloop/internal/db"
	"payloop/internal/lib"
	"payloop/internal/models"
	"payloop/internal/repository/tenants"
)

type TenantRepository struct {
	*db.PgDatabase
	logger lib.Logger
}

func NewTenantRepository(database db.Database, logger lib.Logger) TenantRepository {
	pgDatabase, ok := database.(*db.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return TenantRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *TenantRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Query(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *TenantRepository) FindAll(ctx context.Context) ([]*models.User, error) {
	query := ``
	rows, err := r.Query(ctx, query)
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

func (r *TenantRepository) Create(ctx context.Context, input tenants.CreateTenantInput) (models.Tenant, error) {
	tenantId := "t_" + ksuid.New().String()
	var tenant models.Tenant
	query := `INSERT INTO tenants (id, name, description, created_at, updated_at) 
			  VALUES (@id, @name, @description, NOW(), NOW())
			  RETURNING (id,name,description,created_at,updated_at)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"id":          tenantId,
		"name":        input.Name,
		"description": input.Description,
	}).Scan(&tenant)

	if err != nil {
		r.logger.Error(`failed to insert tenant`, err)
		return models.Tenant{}, err
	}

	return tenant, nil
}
