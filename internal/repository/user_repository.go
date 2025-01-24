package repository

import (
	"context"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"

	"payloop/internal/models"
)

type UserRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewUserRepository(database lib.Database, logger lib.Logger) UserRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return UserRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	query := "SELECT id, name, email FROM users"
	row, _ := r.PgDatabase.Tx.Query(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*models.User, error) {
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

func (r *UserRepository) Create(ctx context.Context, user models.User) error {
	query := "INSERT INTO users (username, email, password) VALUES ($1, $2, $3)"
	_, err := r.Tx.Exec(ctx, query, user.Username, user.Email, user.Password)
	return err
}

func (r *UserRepository) Update(ctx context.Context, user models.User) error {
	query := "UPDATE users SET username=$1, email=$2, password=$3 WHERE id=$4"
	_, err := r.Tx.Exec(ctx, query, user.Username, user.Email, user.Password, user.ID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	query := "DELETE FROM users WHERE id=$1"
	_, err := r.Tx.Exec(ctx, query, id)
	return err
}
