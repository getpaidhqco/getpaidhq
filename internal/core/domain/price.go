package domain

import (
	"payloop/internal/lib"
	"time"
)

type Price struct {
	OrgId              string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id                 string            `gorm:"column:id;primaryKey" json:"id"`
	VariantId          string            `gorm:"column:variant_id" json:"variant_id"`
	Label              string            `gorm:"column:label" json:"label"`
	Category           PriceCategory     `gorm:"column:category" json:"category"`
	Scheme             PriceScheme       `gorm:"column:scheme" json:"scheme"`
	Cycles             int               `gorm:"column:cycles" json:"cycles"`
	Currency           Currency          `gorm:"column:currency" json:"currency"`
	UnitPrice          int64             `gorm:"column:unit_price" json:"unit_price"`
	MinPrice           int64             `gorm:"column:min_price" json:"min_price"`
	SuggestedPrice     int64             `gorm:"column:suggested_price" json:"suggested_price"`
	BillingInterval    BillingInterval   `gorm:"column:billing_interval" json:"billing_interval"`
	BillingIntervalQty int               `gorm:"column:billing_interval_qty" json:"billing_interval_qty"`
	TrialInterval      BillingInterval   `gorm:"column:trial_interval" json:"trial_interval"`
	TrialIntervalQty   int               `gorm:"column:trial_interval_qty" json:"trial_interval_qty"`
	TaxCode            string            `gorm:"column:tax_code" json:"tax_code"`
	Metadata           map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt          time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Price) TableName() string { return "prices" }

func NewPrice(orgId, variantId string, input CreatePriceInput) Price {
	if input.BillingInterval == "" {
		input.BillingInterval = BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = BillingIntervalNone
	}

	return Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

type CreatePriceInput struct {
	OrgId              string            `json:"org_id"`
	Label              string            `json:"label"`
	VariantId          string            `json:"variant_id"`
	Category           PriceCategory     `json:"category"`
	Scheme             PriceScheme       `json:"scheme"`
	Cycles             int               `json:"cycles"`
	Currency           string            `json:"currency"`
	UnitPrice          int64             `json:"unit_price"`
	MinPrice           int64             `json:"min_price"`
	SuggestedPrice     int64             `json:"suggested_price"`
	BillingInterval    BillingInterval   `json:"billing_interval"`
	BillingIntervalQty int               `json:"billing_interval_qty"`
	TrialInterval      BillingInterval   `json:"trial_interval"`
	TrialIntervalQty   int               `json:"trial_interval_qty"`
	TaxCode            string            `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
}
