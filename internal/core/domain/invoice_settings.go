package domain

import (
	"encoding/json"
	"fmt"
)

// Setting coordinates for the per-org invoice numbering/format config.
const (
	InvoiceSettingsSettingParent = "billing"
	InvoiceSettingsSettingId     = "invoice"
)

// InvoiceSettings is the resolved per-tenant invoice reference policy: the
// prefix and zero-padding width used to render a human invoice reference.
type InvoiceSettings struct {
	Prefix  string `json:"prefix"`
	Padding int    `json:"padding"`
}

// DefaultInvoiceSettings is the fallback when an org has no invoice setting:
// "INV-" prefix, 6-digit zero padding. Tenants override via the invoice-settings
// endpoint. Mirrors DefaultReminderConfig()'s role.
func DefaultInvoiceSettings() InvoiceSettings {
	return InvoiceSettings{Prefix: "INV-", Padding: 6}
}

// Marshal renders the config to the persisted JSON string.
func (s InvoiceSettings) Marshal() (string, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

// ParseInvoiceSettings parses a persisted value; empty input returns the
// default. Empty/zero fields fall back to the defaults so a partial setting is
// always usable.
func ParseInvoiceSettings(raw string) (InvoiceSettings, error) {
	out := DefaultInvoiceSettings()
	if raw == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return DefaultInvoiceSettings(), err
	}
	if out.Prefix == "" {
		out.Prefix = "INV-"
	}
	if out.Padding <= 0 {
		out.Padding = 6
	}
	return out, nil
}

// FormatReference renders the human invoice reference, e.g. number 42 → "INV-000042".
func (s InvoiceSettings) FormatReference(number int64) string {
	return fmt.Sprintf("%s%0*d", s.Prefix, s.Padding, number)
}
