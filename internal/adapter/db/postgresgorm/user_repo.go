package postgresgorm

import (
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) port.UserRepository {
	return &UserRepo{db: db}
}
