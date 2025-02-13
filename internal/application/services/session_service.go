package services

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/sessions"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type SessionService struct {
	sessionRepository repositories.SessionRepository
	cartRepository    repositories.CartRepository
	pubsub            events.PubSub
	logger            lib.Logger
}

func NewSessionService(sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	logger lib.Logger,
	pubsub events.PubSub,
) SessionService {
	return SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error) {
	cartData := cart.New(cart.CreateCartOptions{
		Currency: input.Currency,
		Items:    make([]cart.Item, 0),
	})

	cartInstance, err := s.cartRepository.Create(ctx, entities.Cart{
		OrgId:    input.OrgId,
		Id:       lib.GenerateId("cart"),
		Data:     cartData,
		Metadata: nil,
	})
	if err != nil {
		s.logger.Error(`failed to create cart`, err)
		return entities.Session{}, err
	}

	session, err := s.sessionRepository.Create(ctx,
		entities.Session{
			OrgId:  input.OrgId,
			Id:     lib.GenerateId("session"),
			CartId: cartInstance.Id,
		})

	_ = s.pubsub.Publish(input.OrgId, topic.SessionCreated, session)
	return session, err
}
