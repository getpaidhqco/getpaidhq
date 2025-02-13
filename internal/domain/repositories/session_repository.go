package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type SessionRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Session, error)
	Create(ctx context.Context, input entities.Session) (entities.Session, error)
}
