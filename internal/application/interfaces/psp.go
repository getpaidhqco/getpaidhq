package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type GatewayService interface {
	CreateGateway(ctx context.Context, input dto.CreateGatewayInput) (entities.Gateway, error)
}
