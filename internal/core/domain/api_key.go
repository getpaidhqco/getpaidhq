package domain

import "time"

// ApiKey is the stored record for an API key. The raw secret is NEVER
// persisted — only the HMAC of it (computed with a server-side pepper)
// is. Callers receive the raw secret exactly once when the key is
// created (via port.CreatedApiKey returned from ApiKeyService.Create);
// after that, only KeyHash is available.
type ApiKey struct {
	OrgId string
	Id    string
	// Name is an optional human-readable label set at creation time
	// (e.g. "ci-deploy"). Purely metadata — authentication uses the
	// hash only.
	Name      string
	KeyHash   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
