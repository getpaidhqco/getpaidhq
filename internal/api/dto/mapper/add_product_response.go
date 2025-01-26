package mapper

import (
	"payloop/internal/api/dto/response"
	"payloop/internal/models"
)

func ToCartResponse(cart models.Cart) response.CartResponse {
	return response.CartResponse{
		cart.Data,
	}
}
