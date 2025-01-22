package services

import (
	"context"
	"payloop/internal/domain/accounts"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type AccountService struct {
	repository repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) AccountService {
	return AccountService{repository: repo}
}

func (s *AccountService) Create(ctx context.Context, input accounts.CreateAccountInput) (models.Account, error) {
	return s.repository.Create(ctx, input)
}
