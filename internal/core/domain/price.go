package domain

import "time"

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
