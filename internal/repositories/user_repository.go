package repositories

import (
	"context"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/db"
	"payloop/internal/models"
)

type UserRepository struct {
	db *db.PgDatabase
}

func NewUserRepository(db *db.PgDatabase) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	query := "SELECT id, username, email, password FROM users WHERE id=$1"
	row := r.db.QueryRow(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*models.User, error) {
	query := "SELECT id, username, email, password FROM users"
	rows, err := r.db.Query(ctx, query)
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
	_, err := r.db.Exec(ctx, query, user.Username, user.Email, user.Password)
	return err
}

func (r *UserRepository) Update(ctx context.Context, user models.User) error {
	query := "UPDATE users SET username=$1, email=$2, password=$3 WHERE id=$4"
	_, err := r.db.Exec(ctx, query, user.Username, user.Email, user.Password, user.ID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	query := "DELETE FROM users WHERE id=$1"
	_, err := r.db.Exec(ctx, query, id)
	return err
}
