package domain

import "time"

type Payment struct {
	OrgId          string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id             string            `gorm:"column:id;primaryKey" json:"id"`
	Psp            Gateway           `gorm:"column:psp" json:"psp"`
	PspId          string            `gorm:"column:psp_id" json:"psp_id"`
	Reference      string            `gorm:"column:reference" json:"reference"`
	OrderId        string            `gorm:"column:order_id" json:"order_id"`
	SubscriptionId string            `gorm:"column:subscription_id" json:"subscription_id"`
	Status         PaymentStatus     `gorm:"column:status" json:"status"`
	Recurring      bool              `gorm:"column:recurring" json:"recurring"`
	Currency       Currency          `gorm:"column:currency" json:"currency"`
	Amount         int64             `gorm:"column:amount" json:"amount"`
	PspFee         int64             `gorm:"column:psp_fee" json:"psp_fee"`
	PlatformFee    int64             `gorm:"column:platform_fee" json:"platform_fee"`
	NetAmount      int64             `gorm:"column:net_amount" json:"net_amount"`
	Metadata       map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CompletedAt    time.Time         `gorm:"column:completed_at" json:"completed_at"`
	CreatedAt      time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Payment) TableName() string { return "payments" }

// SetMetadata merges the existing metadata with the specified values.
func (p *Payment) SetMetadata(meta map[string]string) *Payment {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		p.Metadata[key] = value
	}
	return p
}
