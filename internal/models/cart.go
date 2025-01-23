package models

type Cart struct {
	ID     uint     `json:"id"`
	Data   CartData `json:"data"`
	Status string   `json:"status"`
	Total  int      `json:"total"`
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)

type CartData struct {
	Currency string `json:"currency"`
	Total    int    `json:"total"`
	SubTotal int    `json:"sub_total"`
	Discount int    `json:"discount"`
	Shipping int    `json:"shipping"`
	Tax      int    `json:"tax"`

	Items []CartItem `json:"items"`
}

type CartItem struct {
	ID          string `json:"id"`
	ProductId   string `json:"product_id"`
	VariantId   string `json:"variant_id"`
	PriceId     string `json:"price_id"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Price       int    `json:"price"`
	Total       int    `json:"total"`
}
