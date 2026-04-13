package domain

import "time"

type WebhookSubscription struct {
	OrgID     string    `gorm:"column:org_id" json:"org_id"`
	Id        string    `gorm:"column:id;primaryKey" json:"id"`
	Events    []string  `gorm:"column:events;serializer:json" json:"events"`
	URL       string    `gorm:"column:url" json:"url"`
	Secret    string    `gorm:"column:secret" json:"secret,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (WebhookSubscription) TableName() string { return "webhook_subscriptions" }

type CreateWebhookSubscriptionInput struct {
	OrgId  string   `json:"org_id"`
	Url    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}
