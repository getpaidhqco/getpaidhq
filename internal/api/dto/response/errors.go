package response

import (
	"github.com/go-playground/validator/v10"
	"strings"
)

func FormatValidationErrors(errs validator.ValidationErrors) []map[string]string {
	errors := make([]map[string]string, len(errs))
	for i, fe := range errs {
		errors[i] = map[string]string{
			"field":   getJSONFieldName(fe),
			"message": validationErrorToText(fe),
		}
	}
	return errors
}

// getJSONFieldName extracts the JSON field name from the struct field
func getJSONFieldName(fe validator.FieldError) string {
	// Get the struct field
	fieldName := fe.Field()

	// Get the struct type
	structType := fe.StructNamespace()
	if structType == "" {
		return toSnakeCase(fieldName)
	}

	// Try to get the actual field from reflection
	// This is a simplified approach - you might need to adjust based on your specific needs
	return toSnakeCase(fieldName)
}

// toSnakeCase converts PascalCase to snake_case
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
