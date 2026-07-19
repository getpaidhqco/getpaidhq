package validators

import (
	"fmt"
	"getpaidhq/internal/lib/currencies"

	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation("iso4217", currencies.Validate); err != nil {
		panic(fmt.Errorf("register iso4217 validator: %w", err))
	}
	return v
}
