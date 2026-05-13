package lib

import (
	"github.com/go-playground/validator/v10"
)

// NewValidator builds the shared *validator.Validate instance with the
// project's custom rules (currently the ISO 4217 currency check). The
// returned validator is wired into Fuego at server construction so every
// DTO bound through Fuego's body decoder is validated the same way.
func NewValidator(logger Logger) *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation("iso4217", ValidateCurrency); err != nil {
		logger.Errorf("register iso4217 validator: %v", err)
	}
	return v
}
