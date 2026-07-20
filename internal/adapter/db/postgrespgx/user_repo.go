package postgrespgx

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/port"
)

// UserRepo holds the pool for user persistence. port.UserRepository is
// currently `any` (no methods wired), so this is a placeholder for when user
// persistence is implemented.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) port.UserRepository {
	return &UserRepo{pool: pool}
}
