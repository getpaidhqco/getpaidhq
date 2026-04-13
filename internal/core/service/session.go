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
	logger            port.Logger
}

func NewSessionService(
	sessionRepository port.SessionRepository,
	cartRepository port.CartRepository,
	logger port.Logger,
	pubsub port.PubSub,
) *SessionService {
	return &SessionService{
		sessionRepository: sessionRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input domain.CreateSessionInput) (domain.Session, error) {
	cartEntity, err := s.cartRepository.Create(ctx, domain.Cart{
		OrgId: input.OrgId,
		Id:    lib.GenerateId("cart"),
		Data: domain.CartData{
			Currency: input.Currency,
		},
	})
	if err != nil {
		s.logger.Error(`failed to create cart`, err)
		return domain.Session{}, err
	}

	session, err := s.sessionRepository.Create(ctx,
		domain.Session{
			OrgId:  input.OrgId,
			Id:     lib.GenerateId("session"),
			CartId: cartEntity.Id,
		})

	_ = s.pubsub.Publish(input.OrgId, port.TopicSessionCreated, session)
	return session, err
}
