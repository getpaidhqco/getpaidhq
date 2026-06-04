package domain

import "time"

// WebhookSubscription represents a tenant-configured outbound webhook
// endpoint. Note the field is OrgID (not OrgId) — preserved for historical
// reasons; references throughout the codebase use this casing.
type WebhookSubscription struct {
	OrgID     string
	Id        string
	Events    []string
	URL       string
	Secret    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
