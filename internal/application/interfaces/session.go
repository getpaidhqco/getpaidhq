package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/sessions"
)

type SessionService interface {
	CreateSession(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error)
}
