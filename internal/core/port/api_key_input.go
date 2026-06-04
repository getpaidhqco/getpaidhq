package port

import "getpaidhq/internal/core/domain"

// CreatedApiKey is the result of ApiKeyService.Create. It pairs the persisted
// ApiKey aggregate with the plaintext raw secret that was minted at creation.
// The raw secret is surfaced ONCE at creation — there is no recovery flow —
// and never persisted, never logged, never re-derivable from the row.
type CreatedApiKey struct {
	ApiKey domain.ApiKey
	// Key is the plaintext token (sk_<id>_<secret>).
	Key string
}
