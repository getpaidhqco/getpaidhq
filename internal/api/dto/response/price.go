package response

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"time"
)

type Price struct {
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           common.Currency        `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`
	Metadata           map[string]string      `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

func NewPriceFromEntity(entity entities.Price) Price {
	return Price{
		Id:                 entity.Id,
		VariantId:          entity.VariantId,
		Category:           entity.Category,
		Scheme:             entity.Scheme,
		Cycles:             entity.Cycles,
		Currency:           entity.Currency,
		UnitPrice:          entity.UnitPrice,
		MinPrice:           entity.MinPrice,
		SuggestedPrice:     entity.SuggestedPrice,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		TrialInterval:      entity.TrialInterval,
		TrialIntervalQty:   entity.TrialIntervalQty,
		TaxCode:            entity.TaxCode,
		Metadata:           entity.Metadata,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}
