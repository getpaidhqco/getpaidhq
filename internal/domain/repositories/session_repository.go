package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/sessions"
)

type SessionRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Session, error)
	Create(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error)
}
