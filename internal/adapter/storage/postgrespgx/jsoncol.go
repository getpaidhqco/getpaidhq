package postgrespgx

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// jsonCol[T] adapts a Go value of type T to a Postgres json/jsonb column. It
// implements driver.Valuer (marshal on write) and sql.Scanner (unmarshal on
// read), which pgx honours. This is the pgx counterpart to the gorm rows'
// `serializer:json` tag.
//
// Use it for every column the gorm row tagged `serializer:json` — metadata
// maps, embedded structs (domain.Address), string slices, etc. On the write
// path apply emptyIfNil to NOT NULL metadata columns first so they store `{}`,
// never NULL.
type jsonCol[T any] struct{ V T }

func newJSON[T any](v T) jsonCol[T] { return jsonCol[T]{V: v} }

// Value implements driver.Valuer.
func (j jsonCol[T]) Value() (driver.Value, error) {
	b, err := json.Marshal(j.V)
	if err != nil {
		return nil, fmt.Errorf("jsonCol: marshal %T: %w", j.V, err)
	}
	return b, nil
}

// Scan implements sql.Scanner. A NULL column leaves T at its zero value.
func (j *jsonCol[T]) Scan(src any) error {
	if src == nil {
		var zero T
		j.V = zero
		return nil
	}
	var b []byte
	switch s := src.(type) {
	case []byte:
		b = s
	case string:
		b = []byte(s)
	default:
		return fmt.Errorf("jsonCol: cannot scan %T into %T", src, j.V)
	}
	if len(b) == 0 {
		var zero T
		j.V = zero
		return nil
	}
	if err := json.Unmarshal(b, &j.V); err != nil {
		return fmt.Errorf("jsonCol: unmarshal into %T: %w", j.V, err)
	}
	return nil
}
