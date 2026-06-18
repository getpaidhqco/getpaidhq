package postgresgorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("nulltime", ZeroTimeNullSerializer{})
}

// ZeroTimeNullSerializer maps a value-type time.Time field to a nullable SQL
// timestamp: the Go zero time persists as NULL, and a NULL column scans back to
// the zero time. The domain already uses the zero time.Time as its "unset"
// sentinel (omitzero JSON tags, explicit time.Time{} resets), so this preserves
// that convention while storing NULL instead of 0001-01-01.
//
// Tag nullable DateTime columns whose Go field is time.Time (not *time.Time)
// with `serializer:nulltime`. Do not tag NOT NULL columns (created_at, etc.).
type ZeroTimeNullSerializer struct{}

// Scan implements schema.SerializerInterface. NULL → zero time.
func (ZeroTimeNullSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	var t time.Time
	if dbValue != nil {
		nt := sql.NullTime{}
		if err := nt.Scan(dbValue); err != nil {
			return fmt.Errorf("nulltime: scan column %q: %w", field.DBName, err)
		}
		t = nt.Time
	}
	return field.Set(ctx, dst, t)
}

// Value implements schema.SerializerInterface. Zero time → NULL.
func (ZeroTimeNullSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	t, ok := fieldValue.(time.Time)
	if !ok {
		return nil, fmt.Errorf("nulltime: column %q is %T, want time.Time", field.DBName, fieldValue)
	}
	if t.IsZero() {
		return nil, nil
	}
	return t, nil
}
