package services

import (
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres"
)

type UserService struct {
	repository repositories.UserRepository `json:"orgRepository,omitempty"`
}

func NewUserService(repo postgres.UserRepository) UserService {
	return UserService{repository: repo}
}
