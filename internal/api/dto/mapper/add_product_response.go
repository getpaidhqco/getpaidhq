package mapper

import (
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
)

func ToCartResponse(cart entities.Cart) response.CartResponse {
	return response.CartResponse{
		cart.Data,
	}
}
