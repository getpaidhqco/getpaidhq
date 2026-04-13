package handler

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateCurrency validates a currency code against ISO 4217.
func ValidateCurrency(fl validator.FieldLevel) bool {
	currency := fl.Field().String()
	validCurrencies := map[string]bool{
		"AED": true, "AFN": true, "ALL": true, "AMD": true, "ANG": true, "AOA": true, "ARS": true, "AUD": true,
		"AWG": true, "AZN": true, "BAM": true, "BBD": true, "BDT": true, "BGN": true, "BHD": true, "BIF": true,
		"BMD": true, "BND": true, "BOB": true, "BRL": true, "BSD": true, "BTN": true, "BWP": true, "BYN": true,
		"BZD": true, "CAD": true, "CDF": true, "CHF": true, "CLP": true, "CNY": true, "COP": true, "CRC": true,
		"CUC": true, "CUP": true, "CVE": true, "CZK": true, "DJF": true, "DKK": true, "DOP": true, "DZD": true,
		"EGP": true, "ERN": true, "ETB": true, "EUR": true, "FJD": true, "FKP": true, "FOK": true, "GBP": true,
		"GEL": true, "GGP": true, "GHS": true, "GIP": true, "GMD": true, "GNF": true, "GTQ": true, "GYD": true,
		"HKD": true, "HNL": true, "HRK": true, "HTG": true, "HUF": true, "IDR": true, "ILS": true, "IMP": true,
		"INR": true, "IQD": true, "IRR": true, "ISK": true, "JEP": true, "JMD": true, "JOD": true, "JPY": true,
		"KES": true, "KGS": true, "KHR": true, "KID": true, "KMF": true, "KRW": true, "KWD": true, "KYD": true,
		"KZT": true, "LAK": true, "LBP": true, "LKR": true, "LRD": true, "LSL": true, "LYD": true, "MAD": true,
		"MDL": true, "MGA": true, "MKD": true, "MMK": true, "MNT": true, "MOP": true, "MRU": true, "MUR": true,
		"MVR": true, "MWK": true, "MXN": true, "MYR": true, "MZN": true, "NAD": true, "NGN": true, "NIO": true,
		"NOK": true, "NPR": true, "NZD": true, "OMR": true, "PAB": true, "PEN": true, "PGK": true, "PHP": true,
		"PKR": true, "PLN": true, "PYG": true, "QAR": true, "RON": true, "RSD": true, "RUB": true, "RWF": true,
		"SAR": true, "SBD": true, "SCR": true, "SDG": true, "SEK": true, "SGD": true, "SHP": true, "SLE": true,
		"SLL": true, "SOS": true, "SRD": true, "SSP": true, "STN": true, "SYP": true, "SZL": true, "THB": true,
		"TJS": true, "TMT": true, "TND": true, "TOP": true, "TRY": true, "TTD": true, "TVD": true, "TWD": true,
		"TZS": true, "UAH": true, "UGX": true, "USD": true, "UYU": true, "UZS": true, "VES": true, "VND": true,
		"VUV": true, "WST": true, "XAF": true, "XCD": true, "XDR": true, "XOF": true, "XPF": true, "YER": true,
		"ZAR": true, "ZMW": true, "ZWL": true,
	}
	return validCurrencies[currency]
}

// FormatValidationErrors formats validator.ValidationErrors into a user-friendly list.
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

// getJSONFieldName extracts the JSON field name from the struct field.
func getJSONFieldName(fe validator.FieldError) string {
	fieldName := fe.Field()

	structType := fe.StructNamespace()
	if structType == "" {
		return toSnakeCase(fieldName)
	}

	return toSnakeCase(fieldName)
}

// toSnakeCase converts PascalCase to snake_case.
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
