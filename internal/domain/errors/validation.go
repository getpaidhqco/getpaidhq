package errors

import "fmt"

type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

func NewValidationError(field, message string) ValidationError {
    return ValidationError{Field: field, Message: message}
}

// Specific errors
var (
    ErrMissingOrgId             = NewValidationError("orgId", "organization ID is required")
    ErrMissingVariantId         = NewValidationError("variantId", "variant ID is required")
    ErrInvalidPriceCategory     = NewValidationError("category", "invalid price category")
    ErrInvalidUsageConfiguration = NewValidationError("usage", "invalid usage configuration")
    ErrMissingCurrency          = NewValidationError("currency", "currency is required")
)