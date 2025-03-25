package mapper

import (
	"encoding/json"
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/cart"
)

func ToCartResponse(entity entities.Cart) response.CartResponse {

	data, err := json.Marshal(entity.Data)
	if err != nil {
		// handle error
	}
	var cartData cart.CartData
	if err := json.Unmarshal(data, &cartData); err != nil {
		// handle error
	}

	return response.CartResponse{
		CartData: entity.Data.(cart.CartData),
	}
}
