package services

import (
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
