package postgrespgx

import "time"

// row_helpers.go houses small utilities shared across <entity>Row → domain
// mappers. The gorm adapter gets nullable-string and zero-time↔NULL behaviour
// from struct tags + serializers; in pgx we do it explicitly in the mappers
// with these helpers.

// strOrEmpty dereferences a nullable string column to "" when NULL. Read path.
func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nilIfEmpty maps the domain's "" sentinel to a SQL NULL for nullable FK/unique
// columns (external_id, default_payment_method_id, …) so "" never lands in a
// column where it would violate an FK or collide in a unique index. Write path.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullTime maps the domain's zero-time "unset" sentinel to a SQL NULL for
// nullable timestamp columns. Write path — the pgx mirror of the gorm
// `serializer:nulltime` Value direction.
func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// timeOrZero maps a NULL nullable-timestamp column back to the zero time.
// Read path — the mirror of the nulltime serializer's Scan direction.
func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
