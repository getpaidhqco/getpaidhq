package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Gateway struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGatewayFromEntity(entity entities.Gateway) Gateway {

	return Gateway{
		Id:        entity.Id,
		Name:      entity.Name,
		UpdatedAt: entity.UpdatedAt,
		CreatedAt: entity.CreatedAt,
	}
}
