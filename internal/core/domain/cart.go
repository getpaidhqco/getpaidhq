package domain

type Cart struct {
	OrgId    string      `json:"org_id"`
	Id       string      `json:"id"`
	Data     interface{} `json:"data"`
	Status   string      `json:"status"`
	Total    int64       `json:"total"`
	Metadata interface{} `json:"metadata"`
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)

type AddProductCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemCommand struct {
	OrgId  string `json:"org_id"`
	CartId string `json:"cart_id"`
	Id     string `json:"id"`
}

type AdjustCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}
