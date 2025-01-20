package tenants

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"

	"payloop/internal/lib"

	"payloop/internal/models"
)

type Repository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewTenantRepository(database lib.Database, logger lib.Logger) Repository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return Repository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *Repository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Query(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindAll(ctx context.Context) ([]*models.User, error) {
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

func (r *Repository) Create(ctx context.Context, input CreateTenantInput) (models.Tenant, error) {
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
