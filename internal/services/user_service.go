package services

import (
	"context"
	"payloop/internal/models"
	"payloop/internal/repositories"
)

type UserService struct {
	repo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetOneUser(id uint) (*models.User, error) {
	return s.repo.FindByID(context.Background(), id)
}

func (s *UserService) GetAllUser() ([]*models.User, error) {
	return s.repo.FindAll(context.Background())
}

func (s *UserService) CreateUser(user models.User) error {
	return s.repo.Create(context.Background(), user)
}

func (s *UserService) UpdateUser(user models.User) error {
	return s.repo.Update(context.Background(), user)
}

func (s *UserService) DeleteUser(id uint) error {
	return s.repo.Delete(context.Background(), id)
}
