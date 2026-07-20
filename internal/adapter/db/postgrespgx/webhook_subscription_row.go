package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// webhookSubscriptionRow is the postgres on-the-wire shape of a
// WebhookSubscription. Package-internal.
//
// Events maps to a native Postgres text[] column (schema: `events text[]`).
// pgx encodes/decodes a Go []string directly to/from text[], so no serializer
// is needed — a json-encoded slice would be rejected as a malformed array
// literal.
type webhookSubscriptionRow struct {
	OrgID     string
	Id        string
	Events    []string
	URL       string
	Secret    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const webhookSubscriptionColumns = `org_id, id, events, url, secret, created_at, updated_at`

func (r *webhookSubscriptionRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgID, &r.Id, &r.Events, &r.URL, &r.Secret, &r.CreatedAt, &r.UpdatedAt)
}

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
		Events:    w.Events,
		URL:       w.URL,
		Secret:    w.Secret,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
