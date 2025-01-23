package cart

import "payloop/internal/models"

type CreateCartInput struct {
	AccountId string            `json:"account_id"`
	Cart      models.CartData   `json:"cart"`
	Metadata  map[string]string `json:"metadata"`
}
