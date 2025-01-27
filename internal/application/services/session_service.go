package services

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"
)

type SessionService struct {
	sessionRepository repositories.SessionRepository
	cartRepository    repositories.CartRepository
	logger            lib.Logger
}

func NewSessionService(sessionRepository repositories.SessionRepository, cartRepository repositories.CartRepository, logger lib.Logger) SessionService {
	return SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		logger:            logger,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input sessions.CreateSessionRequest) (entities.Session, error) {
	cartData := cart.New(cart.CreateCartOptions{
		Currency: input.Currency,
		Items:    make([]cart.Item, 0),
	})

	cartInstance, err := s.cartRepository.Create(ctx, carts.CreateCartInput{
		OrgId:    input.OrgId,
		Cart:     cartData,
		Metadata: nil,
	})
	if err != nil {
		s.logger.Error(`failed to create cart`, err)
		return entities.Session{}, err
	}

	session, err := s.sessionRepository.Create(ctx,
		sessions.CreateSessionInput{
			OrgId:    input.OrgId,
			Id:       lib.GenerateId("session"),
			CartId:   cartInstance.Id,
			Metadata: nil,
		})

	return session, err
}
