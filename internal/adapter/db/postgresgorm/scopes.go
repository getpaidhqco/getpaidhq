package postgresgorm

import (
	"regexp"
	"strings"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
)

func OrgScope(orgId string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("org_id = ?", orgId)
	}
}

// emptyIfNil returns a non-nil map so a `serializer:json` metadata column that
// is NOT NULL in the schema receives `{}` rather than SQL NULL. A nil Go map
// serializes to NULL, which violates those constraints (orders, order_items,
// subscriptions). Callers apply this on the write path before insert.
func emptyIfNil(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// safeIdentifier matches a single bare SQL identifier — lower/underscore
// start, then lower/digit/underscore. We deliberately keep it conservative
// (no quoted identifiers, no schema.table.column dotted form) so that
// concatenation into the ORDER BY string can never carry an injection,
// even if a caller forgets to gate the input upstream.
var safeIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Paginate applies sort + limit + offset to a query. GORM does not
// parameterize identifiers in .Order(), so the column and direction are
// validated here against an allowlist-style check; anything that doesn't
// pass falls back to the safe default `created_at DESC`. This is the
// single chokepoint every list endpoint goes through, so the validation
// has to live here.
//
// `limit` is clamped to [1, 200] to bound result-set memory and DB load.
func Paginate(p domain.Pagination) func(db *gorm.DB) *gorm.DB {
	col := strings.ToLower(strings.TrimSpace(p.SortBy))
	if !safeIdentifier.MatchString(col) {
		col = "created_at"
	}

	dir := strings.ToUpper(strings.TrimSpace(p.SortDirection))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	limit := p.Limit
	switch {
	case limit <= 0:
		limit = 10
	case limit > 200:
		limit = 200
	}

	offset := p.Offset
	if offset < 0 {
		offset = 0
	}

	return func(db *gorm.DB) *gorm.DB {
		return db.Order(col + " " + dir).Limit(limit).Offset(offset)
	}
}
