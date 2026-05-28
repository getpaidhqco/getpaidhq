package lib

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// NewValidator builds the shared *validator.Validate instance with the
// project's custom rules (currently the ISO 4217 currency check). The
// returned validator is wired into Fuego at server construction so every
// DTO bound through Fuego's body decoder is validated the same way.
//
// Registration failure panics. A degraded validator silently turns every
// `validate:"iso4217"` tag into a no-op, which would let invalid currency
// codes through unnoticed — a fail-fast at startup is much louder than a
// production validation bypass.
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation("iso4217", ValidateCurrency); err != nil {
		panic(fmt.Errorf("register iso4217 validator: %w", err))
	}
	return v
}

// ValidateCurrency validates a currency code against ISO 4217.
func ValidateCurrency(fl validator.FieldLevel) bool {
	currency := fl.Field().String()
	validCurrencies := map[string]bool{
		"AED": true, "AFN": true, "ALL": true, "AMD": true, "ANG": true, "AOA": true, "ARS": true, "AUD": true,
		"AWG": true, "AZN": true, "BAM": true, "BBD": true, "BGN": true, "BHD": true, "BIF": true,
		"BMD": true, "BND": true, "BOB": true, "BRL": true, "BSD": true, "BTN": true, "BWP": true, "BYN": true,
		"BZD": true, "CAD": true, "CDF": true, "CHF": true, "CLP": true, "CNY": true, "COP": true, "CRC": true,
		"CUC": true, "CUP": true, "CVE": true, "CZK": true, "DJF": true, "DKK": true, "DOP": true, "DZD": true,
		"EGP": true, "ERN": true, "ETB": true, "EUR": true, "FJD": true, "FKP": true, "GBP": true,
		"GEL": true, "GHS": true, "GIP": true, "GMD": true, "GNF": true, "GTQ": true, "GYD": true,
		"HKD": true, "HNL": true, "HTG": true, "HUF": true, "IDR": true, "ILS": true,
		"INR": true, "IQD": true, "IRR": true, "ISK": true, "JMD": true, "JOD": true, "JPY": true,
		"KES": true, "KGS": true, "KHR": true, "KMF": true, "KRW": true, "KWD": true, "KYD": true,
		"KZT": true, "LAK": true, "LBP": true, "LKR": true, "LRD": true, "LSL": true, "LYD": true, "MAD": true,
		"MDL": true, "MGA": true, "MKD": true, "MMK": true, "MNT": true, "MOP": true, "MRU": true, "MUR": true,
		"MVR": true, "MWK": true, "MXN": true, "MYR": true, "MZN": true, "NAD": true, "NGN": true, "NIO": true,
		"NOK": true, "NPR": true, "NZD": true, "OMR": true, "PAB": true, "PEN": true, "PGK": true, "PHP": true,
		"PKR": true, "PLN": true, "PYG": true, "QAR": true, "RON": true, "RSD": true, "RUB": true, "RWF": true,
		"SAR": true, "SBD": true, "SCR": true, "SDG": true, "SEK": true, "SGD": true, "SHP": true,
		"SOS": true, "SRD": true, "SSP": true, "STN": true, "SYP": true, "SZL": true, "THB": true,
		"TJS": true, "TMT": true, "TND": true, "TOP": true, "TRY": true, "TTD": true, "TWD": true,
		"TZS": true, "UAH": true, "UGX": true, "USD": true, "UYU": true, "UZS": true, "VES": true, "VND": true,
		"VUV": true, "WST": true, "XAF": true, "XCD": true, "XOF": true, "XPF": true, "YER": true,
		"ZAR": true, "ZMW": true, "ZWL": true,
	}
	return validCurrencies[currency]
}
