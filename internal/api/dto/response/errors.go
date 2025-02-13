package response

import (
	"github.com/go-playground/validator/v10"
)

func FormatValidationErrors(errs validator.ValidationErrors) []map[string]string {
	errors := make([]map[string]string, len(errs))
	for i, fe := range errs {
		errors[i] = map[string]string{
			"field":   fe.Field(),
			"error":   fe.Tag(),
			"message": validationErrorToText(fe),
		}
	}
	return errors
}

func validationErrorToText(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address"
	}
	return "Unknown error"
}
