package response

import "payloop/internal/infrastructure/cart"

type CartResponse struct {
	cart.CartData
}
