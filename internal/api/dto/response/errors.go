package response

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func FormatValidationErrors(errs validator.ValidationErrors) []gin.H {
	errors := make([]gin.H, len(errs))
	for i, fe := range errs {
		errors[i] = gin.H{
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
