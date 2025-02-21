package mapper

import (
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities/orders"
)

func ToCartItems(cartItems []request.CartItem) []orders.CartItem {
	var items []orders.CartItem
	for _, item := range cartItems {
		items = append(items, orders.CartItem{
			ProductId: item.ProductId,
			PriceId:   item.PriceId,
			Quantity:  item.Quantity,
		})
	}
	return items
}
