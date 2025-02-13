package request

type AddItemRequest struct {
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemRequest struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`
}
