package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type PspService interface {
	CreateGateway(ctx context.Context, input dto.CreateGatewayInput) (entities.PaymentServiceProvider, error)
}
