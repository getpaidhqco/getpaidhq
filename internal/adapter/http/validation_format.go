package handler

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// FormatValidationErrors formats validator.ValidationErrors into a
// user-friendly list of {field, message} maps. Used by both the error
// serializer and direct callers.
func FormatValidationErrors(errs validator.ValidationErrors) []map[string]string {
	errors := make([]map[string]string, len(errs))
	for i, fe := range errs {
		errors[i] = map[string]string{
			"field":   toSnakeCase(fe.Field()),
			"message": validationErrorToText(fe),
		}
	}
	return errors
}

func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func validationErrorToText(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address"
	case "oneof":
		return "Must be one of the allowed values: " + fe.Param()
	case "gt":
		return "Must be greater than " + fe.Param()
	case "gte":
		return "Must be greater or equal than " + fe.Param()
	case "lte":
		return "Must be less than or equal to " + fe.Param()
	case "lt":
		return "Must be less than " + fe.Param()
	case "iso4217":
		return "Must be a valid currency code"
	default:
		return "Invalid value"
	}
}
