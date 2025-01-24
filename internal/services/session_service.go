package services

import (
	"context"
	"payloop/internal/domain/sessions"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type SessionService struct {
	repo repository.SessionRepository
}

func NewSessionService(repo repository.SessionRepository) SessionService {
	return SessionService{repo: repo}
}

// WithTrx enables repository with transaction
func (s *SessionService) WithTrx(trxHandle interface{}) *SessionService {
	s.repo = *s.repo.WithTrx(trxHandle)
	return s
}

func (s *SessionService) CreateSession(ctx context.Context, input sessions.CreateSessionInput) (models.Session, error) {
	return s.repo.Create(ctx, input)
}
