package vault

import (
	"payloop/internal/infrastructure/vault/aes_vault"

	"go.uber.org/fx"
)

// Module provides dependency injection for vault implementations
// This module now uses the AES vault implementation by default
var Module = fx.Options(
	aes_vault.Module,
)