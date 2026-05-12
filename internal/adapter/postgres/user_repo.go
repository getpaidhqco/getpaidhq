package postgres

import (
	"gorm.io/gorm"
	"getpaidhq/internal/core/port"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) port.UserRepository {
	return &UserRepo{db: db}
}
