package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type DocSequenceRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.DocSequence, error)
	FindByType(ctx context.Context, orgId string, sequenceType string) ([]entities.DocSequence, error)
	Create(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error)
	Update(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error)
	GetNextValue(ctx context.Context, orgId string, id string, sequenceType string) (int, error)
}