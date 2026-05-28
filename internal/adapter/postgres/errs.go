package postgres

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"getpaidhq/internal/core/port"
)

// translateErr wraps gorm-specific errors as domain-level sentinels so
// callers can do `errors.Is(err, port.ErrNotFound)` without importing
// gorm. Every repo method that ends with First/Take/Last (i.e. expects
// exactly one row) MUST run its returned error through this helper.
//
// nil maps to nil — the helper is safe to call unconditionally on the
// error of a query.
//
// The original error is wrapped (%w) so callers that need the raw
// driver error still have access via errors.Unwrap / errors.As.
func translateErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", port.ErrNotFound, err)
	}
	return err
}
