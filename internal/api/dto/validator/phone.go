package validator

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

// ValidatePhone validates that a string is a valid E.164 phone number
// E.164 format: +[country code][number] e.g., +14155552671
func ValidatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // Empty is valid for omitempty
	}
	
	// E.164 format regex: + followed by 1-15 digits
	e164Regex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return e164Regex.MatchString(phone)
}