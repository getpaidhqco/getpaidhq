package service

import (
	"getpaidhq/internal/core/port"
)

type UserService struct {
	repository port.UserRepository
}

func NewUserService(repo port.UserRepository) *UserService {
	return &UserService{repository: repo}
}
