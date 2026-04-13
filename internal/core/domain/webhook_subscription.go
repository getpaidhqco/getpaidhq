package domain

import "time"

type WebhookSubscription struct {
	OrgID     string    `json:"org_id"`
	Id        string    `json:"id"`
	Events    []string  `json:"events"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateWebhookSubscriptionInput struct {
	OrgId  string   `json:"org_id"`
	Url    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}
