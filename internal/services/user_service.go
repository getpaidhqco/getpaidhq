package services

import (
	"context"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type UserService struct {
	repository repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{repository: repo}
}

func (s *UserService) GetUser(id uint) (*models.User, error) {
	return s.repository.FindByID(context.Background(), id)
}

func (s *UserService) GetAllUsers() ([]*models.User, error) {
	return s.repository.FindAll(context.Background())
}

func (s *UserService) CreateUser(user models.User) error {
	return s.repository.Create(context.Background(), user)
}

func (s *UserService) UpdateUser(user models.User) error {
	return s.repository.Update(context.Background(), user)
}

func (s *UserService) DeleteUser(id uint) error {
	return s.repository.Delete(context.Background(), id)
}
