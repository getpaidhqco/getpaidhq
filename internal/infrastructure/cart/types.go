package cart

import "payloop/internal/infrastructure/cart/types"

type CartData struct {
	Currency      string `json:"currency"`
	Total         int64  `json:"total"`
	SubTotal      int64  `json:"sub_total"`
	DiscountTotal int64  `json:"discount"`
	ShippingTotal int64  `json:"shipping"`
	TaxTotal      int64  `json:"tax"`

	Items []Item `json:"items"`
}

type AddItemInput struct {
	ProductId string
	PriceId   string
	Quantity  int
}

type Item struct {
	Id            string `json:"id"`
	ProductId     string `json:"product_id"`
	Price         Price  `json:"price"`
	Description   string `json:"description"`
	Quantity      int64  `json:"quantity"`
	UnitPrice     int64  `json:"unit_price"`
	TaxTotal      int64  `json:"tax_total"`
	SubTotal      int64  `json:"sub_total"`
	DiscountTotal int64  `json:"discount_total"`
	ShippingTotal int64  `json:"shipping_total"`
	Total         int64  `json:"total"`
}

func (i *Item) Calculate() {
	i.SubTotal = i.UnitPrice * i.Quantity
	i.Total = i.SubTotal
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
