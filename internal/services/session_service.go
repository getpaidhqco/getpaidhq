package services

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"
	"payloop/internal/repository"
)

type SessionService struct {
	sessionRepository repository.SessionRepository
	cartRepository    repository.CartRepository
	logger            lib.Logger
}

func NewSessionService(sessionRepository repository.SessionRepository, cartRepository repository.CartRepository, logger lib.Logger) SessionService {
	return SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		logger:            logger,
	}
}

// WithTrx enables repository with transaction
func (s *SessionService) WithTrx(trxHandle interface{}) *SessionService {
	s.sessionRepository = *s.sessionRepository.WithTrx(trxHandle)
	return s
}

func (s *SessionService) CreateSession(ctx context.Context, input sessions.CreateSessionRequest) (entities.Session, error) {
	cartData := cart.New(cart.CreateCartOptions{
		Currency: input.Currency,
		Items:    make([]cart.Item, 0),
	})

	cartInstance, err := s.cartRepository.Create(ctx, carts.CreateCartInput{
		AccountId: input.AccountId,
		Cart:      cartData,
		Metadata:  nil,
	})
	if err != nil {
		s.logger.Error(`failed to create cart`, err)
		return entities.Session{}, err
	}

	session, err := s.sessionRepository.Create(ctx,
		sessions.CreateSessionInput{
			AccountId: input.AccountId,
			Id:        lib.GenerateId("session"),
			CartId:    cartInstance.Id,
			Metadata:  nil,
		})

	return session, err
}
