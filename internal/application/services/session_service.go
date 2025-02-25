package services

import (
	"context"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/sessions"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/cart"
	"payloop/internal/lib"
)

type SessionService struct {
	sessionRepository repositories.SessionRepository
	cartRepository    repositories.CartRepository
	pubsub            events.PubSub
	logger            logger.Logger
}

func NewSessionService(sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	logger logger.Logger,
	pubsub events.PubSub,
) interfaces.SessionService {
	return SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s SessionService) CreateSession(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error) {
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
