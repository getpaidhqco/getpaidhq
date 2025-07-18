package validators

import (
    "encoding/json"
    "errors"
    "fmt"
    "payloop/internal/domain/settings"
    "regexp"
)

// OrganizationSettings represents organization configuration
type OrganizationSettings struct {
    CompanyName    string `json:"company_name"`
    CompanyEmail   string `json:"company_email"`
    CompanyWebsite string `json:"company_website,omitempty"`
    CompanyAddress struct {
        Street     string `json:"street"`
        City       string `json:"city"`
        State      string `json:"state"`
        PostalCode string `json:"postal_code"`
        Country    string `json:"country"`
    } `json:"company_address"`
    TaxSettings struct {
        TaxId        string  `json:"tax_id,omitempty"`
        VatNumber    string  `json:"vat_number,omitempty"`
        TaxExempt    bool    `json:"tax_exempt"`
        DefaultTaxRate float64 `json:"default_tax_rate,omitempty"`
    } `json:"tax_settings,omitempty"`
    Branding struct {
        LogoUrl       string `json:"logo_url,omitempty"`
        PrimaryColor  string `json:"primary_color,omitempty"`
        SecondaryColor string `json:"secondary_color,omitempty"`
    } `json:"branding,omitempty"`
}

type OrganizationValidator struct {
    settings.BaseValidator
}

func NewOrganizationValidator() *OrganizationValidator {
    return &OrganizationValidator{}
}

func (v *OrganizationValidator) ValidateSettings(value interface{}) error {
    var settings OrganizationSettings

    // Try to handle different input types
    switch val := value.(type) {
    case OrganizationSettings:
        // Direct struct type
        settings = val
    case *OrganizationSettings:
        // Pointer to struct
        if val == nil {
            return errors.New("organization settings cannot be nil")
        }
        settings = *val
    case map[string]interface{}:
        // Map from JSON unmarshaling
        // Convert map to JSON bytes
        jsonBytes, err := json.Marshal(val)
        if err != nil {
            return fmt.Errorf("failed to marshal settings map: %w", err)
        }

        // Unmarshal JSON bytes to struct
        if err := json.Unmarshal(jsonBytes, &settings); err != nil {
            return fmt.Errorf("failed to unmarshal organization settings: %w", err)
        }
    default:
        return errors.New("invalid organization settings type")
    }

    // Validate company name
    if settings.CompanyName == "" {
        return errors.New("company_name is required")
    }

    // Validate email
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !emailRegex.MatchString(settings.CompanyEmail) {
        return errors.New("company_email must be a valid email address")
    }

    // Validate website URL if provided
    if settings.CompanyWebsite != "" {
        urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
        if !urlRegex.MatchString(settings.CompanyWebsite) {
            return errors.New("company_website must be a valid URL")
        }
    }

    // Validate address
    if settings.CompanyAddress.Street == "" || settings.CompanyAddress.City == "" ||
       settings.CompanyAddress.Country == "" {
        return errors.New("company address must include street, city, and country")
    }

    // Validate tax rate if provided
    if settings.TaxSettings.DefaultTaxRate < 0 || settings.TaxSettings.DefaultTaxRate > 100 {
        return errors.New("default_tax_rate must be between 0 and 100")
    }

    // Validate colors if provided
    if settings.Branding.PrimaryColor != "" {
        colorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
        if !colorRegex.MatchString(settings.Branding.PrimaryColor) {
            return errors.New("primary_color must be a valid hex color (e.g., #FF0000)")
        }
    }

    return nil
}

func (v *OrganizationValidator) GetSettingsSchema() settings.SettingsSchema {
    return settings.SettingsSchema{
        Name:        "Organization Settings",
        Description: "Configure organization details and branding",
        Fields: []settings.SettingsField{
            {
                Name:        "company_name",
                Type:        "string",
                Required:    true,
                Description: "Legal company name",
            },
            {
                Name:        "company_email",
                Type:        "string",
                Required:    true,
                Description: "Primary contact email",
                Validation:  "email",
            },
            {
                Name:        "company_website",
                Type:        "string",
                Required:    false,
                Description: "Company website URL",
                Validation:  "url",
            },
            {
                Name:        "company_address",
                Type:        "object",
                Required:    true,
                Description: "Company address",
                Children: []settings.SettingsField{
                    {Name: "street", Type: "string", Required: true, Description: "Street address"},
                    {Name: "city", Type: "string", Required: true, Description: "City"},
                    {Name: "state", Type: "string", Required: false, Description: "State/Province"},
                    {Name: "postal_code", Type: "string", Required: false, Description: "Postal/ZIP code"},
                    {Name: "country", Type: "string", Required: true, Description: "Country code (ISO 3166)"},
                },
            },
            {
                Name:        "tax_settings",
                Type:        "object",
                Required:    false,
                Description: "Tax configuration",
                Children: []settings.SettingsField{
                    {Name: "tax_id", Type: "string", Required: false, Description: "Tax identification number"},
                    {Name: "vat_number", Type: "string", Required: false, Description: "VAT registration number"},
                    {Name: "tax_exempt", Type: "boolean", Required: false, Description: "Tax exempt status", Default: false},
                    {Name: "default_tax_rate", Type: "number", Required: false, Description: "Default tax rate percentage", Validation: "min:0,max:100"},
                },
            },
            {
                Name:        "branding",
                Type:        "object",
                Required:    false,
                Description: "Branding configuration",
                Children: []settings.SettingsField{
                    {Name: "logo_url", Type: "string", Required: false, Description: "Company logo URL"},
                    {Name: "primary_color", Type: "string", Required: false, Description: "Primary brand color", Validation: "regex:^#[0-9A-Fa-f]{6}$"},
                    {Name: "secondary_color", Type: "string", Required: false, Description: "Secondary brand color", Validation: "regex:^#[0-9A-Fa-f]{6}$"},
                },
            },
        },
    }
}

func (v *OrganizationValidator) GetDefaultValue() interface{} {
    return &OrganizationSettings{
        CompanyAddress: struct {
            Street     string `json:"street"`
            City       string `json:"city"`
            State      string `json:"state"`
            PostalCode string `json:"postal_code"`
            Country    string `json:"country"`
        }{
            Country: "US",
        },
        TaxSettings: struct {
            TaxId          string  `json:"tax_id,omitempty"`
            VatNumber      string  `json:"vat_number,omitempty"`
            TaxExempt      bool    `json:"tax_exempt"`
            DefaultTaxRate float64 `json:"default_tax_rate,omitempty"`
        }{
            TaxExempt: false,
        },
    }
}
