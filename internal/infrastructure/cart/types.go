package cart

import "payloop/internal/infrastructure/cart/types"

type Cart struct {
	Id string `json:"id"`

	Currency string `json:"currency"`
	Total    int64  `json:"total"`
	SubTotal int64  `json:"sub_total"`
	Discount int64  `json:"discount"`
	Shipping int64  `json:"shipping"`
	Tax      int64  `json:"tax"`

	Items []Item `json:"items"`
}

type Item struct {
	ID          string `json:"id"`
	ProductId   string `json:"product_id"`
	Price       Price  `json:"price"`
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
}

type Summary struct {
	Type      CartType `json:"type"`
	DueNow    int64    `json:"due_now"`
	DueFuture int64    `json:"due_future"`
}

type Price struct {
	Id                 string                `json:"id"`
	Category           types.PriceCategory   `json:"category"`
	Scheme             types.PriceScheme     `json:"scheme"`
	Currency           string                `json:"currency"`
	Cycles             int64                 `json:"cycles"`
	UnitPrice          int64                 `json:"unit_price"`
	MinPrice           int64                 `json:"min_price"`
	SuggestedPrice     int64                 `json:"suggested_price"`
	BillingInterval    types.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int64                 `json:"billing_interval_qty"`
	TrialInterval      types.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int64                 `json:"trial_interval_qty"`
	TaxCode            string                `json:"tax_code"`
}

type CartType string

const (
	CartTypeOnceOff      CartType = "once_off"
	CartTypeSubscription CartType = "subscription"
)
