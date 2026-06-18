package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// sessionRow is the postgres on-the-wire shape of a Session. The sessions table
// also carries a `metadata` jsonb column, but Session has no metadata field, so
// the gorm row never mapped it and neither does this one. cart_id is a nullable
// column; the domain's "" sentinel maps to SQL NULL.
type sessionRow struct {
	OrgId     string
	Id        string
	CartId    *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const sessionColumns = `org_id, id, cart_id, created_at, updated_at`

func (r *sessionRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CartId, &r.CreatedAt, &r.UpdatedAt)
}

func (r sessionRow) toDomain() domain.Session {
	return domain.Session{
		OrgId:     r.OrgId,
		Id:        r.Id,
		CartId:    strOrEmpty(r.CartId),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func sessionRowFromDomain(s domain.Session) sessionRow {
	return sessionRow{
		OrgId:     s.OrgId,
		Id:        s.Id,
		CartId:    nilIfEmpty(s.CartId),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
