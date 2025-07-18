package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Gateway struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	PspId     string            `json:"psp"`
	Active    bool              `json:"active"`
	Settings  map[string]string `json:"settings,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func NewGatewayFromEntity(entity entities.Gateway) Gateway {
	return Gateway{
		Id:        entity.Id,
		Name:      entity.Name,
		PspId:     string(entity.PspId),
		Active:    entity.Active,
		UpdatedAt: entity.UpdatedAt,
		CreatedAt: entity.CreatedAt,
	}
}

func NewGatewayWithSettingsFromEntity(entity entities.Gateway, settings map[string]string) Gateway {
	gateway := NewGatewayFromEntity(entity)
	gateway.Settings = settings
	return gateway
}

type GatewayList struct {
	Gateways []Gateway `json:"gateways"`
}

func NewGatewayListFromEntities(entities []entities.Gateway) GatewayList {
	gateways := make([]Gateway, len(entities))
	for i, entity := range entities {
		gateways[i] = NewGatewayFromEntity(entity)
	}
	return GatewayList{
		Gateways: gateways,
	}
}
