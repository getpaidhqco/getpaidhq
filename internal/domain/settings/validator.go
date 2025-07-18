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