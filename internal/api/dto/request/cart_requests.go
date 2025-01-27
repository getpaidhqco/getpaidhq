package request

type AddItemRequest struct {
	OrgId     string `json:"org_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemRequest struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`
}
