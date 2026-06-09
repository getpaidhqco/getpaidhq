package postgres

import (
	"time"

	"github.com/lib/pq"

	"getpaidhq/internal/core/domain"
)

// webhookSubscriptionRow is the postgres on-the-wire shape of a
// WebhookSubscription. Package-internal.
//
// Events maps to a native Postgres text[] column (Prisma `events String[]`), so it
// must use pq.StringArray. The previous `serializer:json` tag JSON-encoded the slice
// and wrote `["x"]` into the text[] column, which Postgres rejects with
// "malformed array literal" — breaking every webhook create.
type webhookSubscriptionRow struct {
	OrgID     string         `gorm:"column:org_id"`
	Id        string         `gorm:"column:id;primaryKey"`
	Events    pq.StringArray `gorm:"column:events;type:text[]"`
	URL       string         `gorm:"column:url"`
	Secret    string         `gorm:"column:secret"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
}

func (webhookSubscriptionRow) TableName() string { return "webhook_subscriptions" }

func (r webhookSubscriptionRow) toDomain() domain.WebhookSubscription {
	return domain.WebhookSubscription{
		OrgID:     r.OrgID,
		Id:        r.Id,
		Events:    []string(r.Events),
		URL:       r.URL,
		Secret:    r.Secret,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func webhookSubscriptionRowFromDomain(w domain.WebhookSubscription) webhookSubscriptionRow {
	return webhookSubscriptionRow{
		OrgID:     w.OrgID,
		Id:        w.Id,
		Events:    pq.StringArray(w.Events),
		URL:       w.URL,
		Secret:    w.Secret,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
