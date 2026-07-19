package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orgRow is the postgres on-the-wire shape of an Org. Enum columns are held as
// plain strings and converted at the domain boundary so pgx never has to encode
// a defined enum type.
type orgRow struct {
	Id        string
	Name      string
	Country   string
	Timezone  string
	Status    string
	Metadata  jsonCol[map[string]string]
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r orgRow) toDomain() domain.Org {
	return domain.Org{
		Id:        r.Id,
		Name:      r.Name,
		Country:   r.Country,
		Timezone:  r.Timezone,
		Status:    domain.OrgStatus(r.Status),
		Metadata:  r.Metadata.V,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func orgRowFromDomain(o domain.Org) orgRow {
	return orgRow{
		Id:        o.Id,
		Name:      o.Name,
		Country:   o.Country,
		Timezone:  o.Timezone,
		Status:    string(o.Status),
		Metadata:  newJSON(emptyIfNil(o.Metadata)),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
