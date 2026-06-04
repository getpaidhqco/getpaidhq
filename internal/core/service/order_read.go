package service

import "getpaidhq/internal/core/domain"

// OrderDetails is the composed read model for "show me an order" queries.
// OrderItemDetails nests Price as a sub-read-model (per the rules doc: only
// top-level GETs earn read models; nested entities become sub-types).
type OrderDetails struct {
	Order    domain.Order
	Customer domain.Customer
	Items    []OrderItemDetails
}

// OrderItemDetails is the per-item composition used inside OrderDetails.
// Not a top-level read model — OrderItem isn't queried standalone.
type OrderItemDetails struct {
	Item  domain.OrderItem
	Price domain.Price
}
