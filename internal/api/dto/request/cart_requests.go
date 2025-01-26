package request

type AddItemRequest struct {
	AccountId string `json:"account_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemRequest struct {
	AccountId string `json:"account_id"`
	Id        string `json:"id"`
}
