package postgrespgx

import (
	"fmt"
	"regexp"
	"strings"

	"getpaidhq/internal/core/domain"
)

// safeIdentifier matches a single bare SQL identifier — lower/underscore start,
// then lower/digit/underscore. Deliberately conservative (no quoted or dotted
// identifiers) so a column name concatenated into ORDER BY can never carry an
// injection.
var safeIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// paginationClause renders the ORDER BY / LIMIT / OFFSET tail (with a leading
// space) for p, applying allowlist validation and clamps: an invalid sort
// column falls back to created_at, a non-ASC/DESC direction falls back to DESC,
// limit is clamped to [1,200] with a default of 10, and a negative offset
// becomes 0. The column is validated, not parameterized, because SQL does not
// allow binding identifiers.
func paginationClause(p domain.Pagination) string {
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

	return fmt.Sprintf(" ORDER BY %s %s LIMIT %d OFFSET %d", col, dir, limit, offset)
}

// emptyIfNil returns a non-nil map so a NOT NULL `metadata` jsonb column
// receives `{}` rather than SQL NULL. Applied before jsonCol wraps the value on
// the write path.
func emptyIfNil(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
