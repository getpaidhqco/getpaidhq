# Settings Registry Implementation Specification

## Overview
Implement a Settings Registry system with field-level encryption for sensitive data using the existing TokenVault infrastructure. This focuses ONLY on application settings, not payment provider settings.

## Architecture Components

### 1. Core Interfaces

#### Settings Validator Interface
Create file: `internal/domain/settings/validator.go`

```go
package settings

import "context"

// SettingsValidator defines the interface for validating and securing settings
type SettingsValidator interface {
    // ValidateSettings validates the settings structure and values
    ValidateSettings(value interface{}) error
    
    // GetSettingsSchema returns the schema definition for UI generation
    GetSettingsSchema() SettingsSchema
    
    // GetDefaultValue returns the default settings for this type
    GetDefaultValue() interface{}
    
    // PrepareSensitiveData encrypts sensitive fields before storage
    PrepareSensitiveData(ctx context.Context, value interface{}) (interface{}, error)
    
    // RestoreSensitiveData decrypts sensitive fields after retrieval
    RestoreSensitiveData(ctx context.Context, value interface{}) (interface{}, error)
}

// SettingsSchema defines the structure of settings for documentation and UI
type SettingsSchema struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Fields      []SettingsField `json:"fields"`
}

// SettingsField defines a single field in the settings schema
type SettingsField struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"` // string, number, boolean, object
    Required    bool        `json:"required"`
    Description string      `json:"description"`
    Sensitive   bool        `json:"sensitive,omitempty"`
    Default     interface{} `json:"default,omitempty"`
    Validation  string      `json:"validation,omitempty"` // e.g., "min:0,max:30"
    Children    []SettingsField `json:"children,omitempty"` // For nested objects
}

// BaseValidator provides default implementations for non-sensitive validators
type BaseValidator struct{}

func (v *BaseValidator) PrepareSensitiveData(ctx context.Context, value interface{}) (interface{}, error) {
    return value, nil // No-op for non-sensitive data
}

func (v *BaseValidator) RestoreSensitiveData(ctx context.Context, value interface{}) (interface{}, error) {
    return value, nil // No-op for non-sensitive data
}
```

### 2. Settings Registry

Create file: `internal/application/services/settings_registry.go`

```go
package services

import (
    "context"
    "fmt"
    "payloop/internal/domain/security"
    "payloop/internal/domain/settings"
    "payloop/internal/domain/settings/validators"
    "sync"
)

// SettingsRegistry manages all setting validators and encryption
type SettingsRegistry struct {
    validators map[string]settings.SettingsValidator
    vault      security.TokenVault
    mu         sync.RWMutex
}

// SettingsRegistryInterface defines the interface for the settings registry
type SettingsRegistryInterface interface {
    Register(settingType string, validator settings.SettingsValidator)
    GetValidator(settingType string) (settings.SettingsValidator, error)
}

// NewSettingsRegistry creates a new settings registry
func NewSettingsRegistry(vault security.TokenVault) SettingsRegistryInterface {
    registry := &SettingsRegistry{
        validators: make(map[string]settings.SettingsValidator),
        vault:      vault,
    }
    
    // Register all default validators
    registry.registerDefaultValidators()
    return registry
}

func (r *SettingsRegistry) registerDefaultValidators() {
    // Core application settings validators
    r.Register("subscriptions", validators.NewSubscriptionValidator())
    r.Register("organization", validators.NewOrganizationValidator())
}

// Register adds a new validator to the registry
func (r *SettingsRegistry) Register(settingType string, validator settings.SettingsValidator) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.validators[settingType] = validator
}

// GetValidator retrieves a validator by setting type
func (r *SettingsRegistry) GetValidator(settingType string) (settings.SettingsValidator, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    validator, exists := r.validators[settingType]
    if !exists {
        return nil, fmt.Errorf("no validator registered for setting type: %s", settingType)
    }
    return validator, nil
}
```

### 3. Validator Implementations

#### Subscription Settings Validator
Create file: `internal/domain/settings/validators/subscription_validator.go`

```go
package validators

import (
    "context"
    "errors"
    "fmt"
    "payloop/internal/domain/settings"
)

// SubscriptionSettings represents subscription configuration
type SubscriptionSettings struct {
    EnableInvoicePdfs bool         `json:"enable_invoice_pdfs"`
    InvoicePrefix     string       `json:"invoice_prefix"`
    EmailReminders    bool         `json:"email_reminders"`
    ReminderDays      int          `json:"reminder_days"`
    CancelOnFailure   bool         `json:"cancel_on_failure"`
    RetryPolicy       *RetryPolicy `json:"retry_policy,omitempty"`
}

type RetryPolicy struct {
    RetryAttempts int    `json:"attempts"`
    RetryPeriod   int    `json:"retry_period"`
    FailureAction string `json:"failure_action"` // cancel, mark_unpaid, past_due
}

type SubscriptionValidator struct {
    settings.BaseValidator
}

func NewSubscriptionValidator() *SubscriptionValidator {
    return &SubscriptionValidator{}
}

func (v *SubscriptionValidator) ValidateSettings(value interface{}) error {
    settings, ok := value.(*SubscriptionSettings)
    if !ok {
        return errors.New("invalid subscription settings type")
    }
    
    // Validate reminder days
    if settings.ReminderDays < 0 || settings.ReminderDays > 30 {
        return fmt.Errorf("reminder_days must be between 0 and 30, got %d", settings.ReminderDays)
    }
    
    // Validate invoice prefix
    if len(settings.InvoicePrefix) > 10 {
        return errors.New("invoice_prefix must be 10 characters or less")
    }
    
    // Validate retry policy if provided
    if settings.RetryPolicy != nil {
        if settings.RetryPolicy.RetryAttempts < 0 || settings.RetryPolicy.RetryAttempts > 10 {
            return errors.New("retry_attempts must be between 0 and 10")
        }
        
        if settings.RetryPolicy.RetryPeriod < 1 || settings.RetryPolicy.RetryPeriod > 30 {
            return errors.New("retry_period must be between 1 and 30 days")
        }
        
        validActions := map[string]bool{
            "cancel":      true,
            "mark_unpaid": true,
            "past_due":    true,
        }
        if !validActions[settings.RetryPolicy.FailureAction] {
            return fmt.Errorf("invalid failure_action: %s", settings.RetryPolicy.FailureAction)
        }
    }
    
    return nil
}

func (v *SubscriptionValidator) GetSettingsSchema() settings.SettingsSchema {
    return settings.SettingsSchema{
        Name:        "Subscription Settings",
        Description: "Configure subscription billing behavior and retry policies",
        Fields: []settings.SettingsField{
            {
                Name:        "enable_invoice_pdfs",
                Type:        "boolean",
                Required:    true,
                Description: "Enable PDF generation for invoices",
                Default:     true,
            },
            {
                Name:        "invoice_prefix",
                Type:        "string",
                Required:    false,
                Description: "Prefix for invoice numbers (max 10 chars)",
                Validation:  "max:10",
            },
            {
                Name:        "email_reminders",
                Type:        "boolean",
                Required:    true,
                Description: "Send email reminders for upcoming charges",
                Default:     true,
            },
            {
                Name:        "reminder_days",
                Type:        "number",
                Required:    true,
                Description: "Days before charge to send reminder",
                Default:     3,
                Validation:  "min:0,max:30",
            },
            {
                Name:        "cancel_on_failure",
                Type:        "boolean",
                Required:    true,
                Description: "Automatically cancel subscription on payment failure",
                Default:     false,
            },
            {
                Name:        "retry_policy",
                Type:        "object",
                Required:    false,
                Description: "Payment retry configuration",
                Children: []settings.SettingsField{
                    {
                        Name:        "attempts",
                        Type:        "number",
                        Required:    true,
                        Description: "Number of retry attempts",
                        Default:     3,
                        Validation:  "min:0,max:10",
                    },
                    {
                        Name:        "retry_period",
                        Type:        "number",
                        Required:    true,
                        Description: "Days between retry attempts",
                        Default:     3,
                        Validation:  "min:1,max:30",
                    },
                    {
                        Name:        "failure_action",
                        Type:        "string",
                        Required:    true,
                        Description: "Action after all retries fail",
                        Default:     "past_due",
                        Validation:  "in:cancel,mark_unpaid,past_due",
                    },
                },
            },
        },
    }
}

func (v *SubscriptionValidator) GetDefaultValue() interface{} {
    return &SubscriptionSettings{
        EnableInvoicePdfs: true,
        InvoicePrefix:     "",
        EmailReminders:    true,
        ReminderDays:      3,
        CancelOnFailure:   false,
        RetryPolicy: &RetryPolicy{
            RetryAttempts: 3,
            RetryPeriod:   3,
            FailureAction: "past_due",
        },
    }
}
```


#### Organization Settings Validator
Create file: `internal/domain/settings/validators/organization_validator.go`

```go
package validators

import (
    "context"
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
        TaxId        string `json:"tax_id,omitempty"`
        VatNumber    string `json:"vat_number,omitempty"`
        TaxExempt    bool   `json:"tax_exempt"`
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
    settings, ok := value.(*OrganizationSettings)
    if !ok {
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
```

### 4. Updated Settings Service

Update file: `internal/application/services/settings_service.go`

Add these fields and methods to the existing SettingsService:

```go
// Add to struct
type SettingsService struct {
    settingRepository repositories.SettingRepository
    registry          services.SettingsRegistryInterface
    logger            logger.Logger
}

// Update constructor
func NewSettingsService(
    settingRepository repositories.SettingRepository,
    registry services.SettingsRegistryInterface,
    logger logger.Logger,
) interfaces.SettingsService {
    return &SettingsService{
        settingRepository: settingRepository,
        registry:          registry,
        logger:            logger,
    }
}

// Update UpsertSetting method to include validation and encryption
func (s *SettingsService) UpsertSetting(ctx context.Context, orgId, parentId, id string, value interface{}) (entities.Setting, error) {
    // Get validator from registry
    validator, err := s.registry.GetValidator(id)
    if err != nil {
        return entities.Setting{}, fmt.Errorf("no validator found for setting type %s: %w", id, err)
    }
    
    // Validate the settings
    if err := validator.ValidateSettings(value); err != nil {
        return entities.Setting{}, fmt.Errorf("validation failed: %w", err)
    }
    
    // Prepare sensitive data for storage
    secureValue, err := validator.PrepareSensitiveData(ctx, value)
    if err != nil {
        return entities.Setting{}, fmt.Errorf("failed to secure sensitive data: %w", err)
    }
    
    // Marshal the secure value
    jsonValue, err := json.Marshal(secureValue)
    if err != nil {
        return entities.Setting{}, fmt.Errorf("failed to marshal settings: %w", err)
    }
    
    // Create or update the setting
    setting := entities.Setting{
        OrgId:    orgId,
        ParentId: parentId,
        Id:       id,
        Type:     s.determineSettingType(id),
        Value:    string(jsonValue),
    }
    
    return s.settingRepository.Upsert(ctx, setting)
}

// Update GetSetting to include decryption
func (s *SettingsService) GetSetting(ctx context.Context, orgId, parentId, id string, result interface{}) error {
    // Retrieve from database
    setting, err := s.settingRepository.FindById(ctx, orgId, parentId, id)
    if err != nil {
        return err
    }
    
    // Get validator from registry
    validator, err := s.registry.GetValidator(id)
    if err != nil {
        return fmt.Errorf("no validator found for setting type %s: %w", id, err)
    }
    
    // Determine the secure type based on the validator
    secureValue := s.getSecureTypeForValidator(id)
    if err := json.Unmarshal([]byte(setting.Value), secureValue); err != nil {
        return fmt.Errorf("failed to unmarshal secure settings: %w", err)
    }
    
    // Restore sensitive data
    decryptedValue, err := validator.RestoreSensitiveData(ctx, secureValue)
    if err != nil {
        return fmt.Errorf("failed to restore sensitive data: %w", err)
    }
    
    // Marshal and unmarshal to populate the result
    jsonData, err := json.Marshal(decryptedValue)
    if err != nil {
        return fmt.Errorf("failed to marshal decrypted data: %w", err)
    }
    
    return json.Unmarshal(jsonData, result)
}

func (s *SettingsService) getSecureTypeForValidator(settingType string) interface{} {
    switch settingType {
    case "subscriptions":
        return &validators.SubscriptionSettings{}
    case "organization":
        return &validators.OrganizationSettings{}
    default:
        return &map[string]interface{}{}
    }
}

func (s *SettingsService) determineSettingType(id string) string {
    return id
}
```

### 5. Module Registration

Update file: `internal/application/services/services.go`

Add the settings registry to the dependency injection where other services are registered:

```go
// Add the settings registry provider
fx.Provide(
    func(vault security.TokenVault) services.SettingsRegistryInterface {
        return services.NewSettingsRegistry(vault)
    },
),

// Update the SettingsService provider to include the registry:
fx.Provide(
    func(repo repositories.SettingRepository, registry services.SettingsRegistryInterface, logger logger.Logger) interfaces.SettingsService {
        return services.NewSettingsService(repo, registry, logger)
    },
),
```

### 6. Updated Settings Controller

Update file: `internal/api/controllers/settings_controller.go`

Remove the hardcoded logic in the Update method:

```go
// Update updates an existing setting by merging values
func (s SettingsController) Update(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId
    parentId := c.Param("parent_id")
    id := c.Param("id")

    var input interface{}
    if err := c.ShouldBindJSON(&input); err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    setting, err := s.settingsService.UpsertSetting(c.Request.Context(), orgId, parentId, id, input)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(http.StatusOK, setting.Value)
}
```