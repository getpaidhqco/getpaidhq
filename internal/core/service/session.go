package service

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type SessionService struct {
	sessionRepository port.SessionRepository
	cartRepository    port.CartRepository
	pubsub            port.PubSub
	cartFactory       *CartFactory
	logger            port.Logger
}

func NewSessionService(
	sessionRepository port.SessionRepository,
	cartRepository port.CartRepository,
	logger port.Logger,
	cartFactory *CartFactory,
	pubsub port.PubSub,
) *SessionService {
	return &SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		cartFactory:       cartFactory,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input domain.CreateSessionInput) (domain.Session, error) {
	// TODO: resolve cart factory - currently CartFactory does not have NewCart method
	// because cart types are still in infrastructure. For now, create cart entity directly.
	cartInstance, err := s.cartRepository.Create(ctx, domain.Cart{
		OrgId:    input.OrgId,
		Id:       lib.GenerateId("cart"),
		Metadata: nil,
	})
	if err != nil {
		s.logger.Error(`failed to create cart`, err)
		return domain.Session{}, err
	}

	session, err := s.sessionRepository.Create(ctx,
		domain.Session{
			OrgId:  input.OrgId,
			Id:     lib.GenerateId("session"),
			CartId: cartInstance.Id,
		})

	_ = s.pubsub.Publish(input.OrgId, port.TopicSessionCreated, session)
	return session, err
}
