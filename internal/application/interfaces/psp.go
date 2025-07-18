package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type GatewayService interface {
	CreateGateway(ctx context.Context, input dto.CreateGatewayInput) (entities.Gateway, error)
	GetGateway(ctx context.Context, orgId string, id string) (entities.Gateway, map[string]string, error)
	ListGateways(ctx context.Context, filter dto.GatewayFilter) ([]entities.Gateway, error)
	UpdateGateway(ctx context.Context, input dto.UpdateGatewayInput) (entities.Gateway, error)
	DeleteGateway(ctx context.Context, orgId string, id string) error
}
