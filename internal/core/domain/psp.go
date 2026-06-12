package domain

import "time"

// PspConfig represents a payment service provider configuration for an organization.
// Named PspConfig (not Gateway) to avoid collision with the Gateway string type.
type PspConfig struct {
	OrgId  string
	Id     string
	PspId  Gateway
	Name   string
	Active bool
	// Config holds the non-secret, UI-readable gateway settings (themes,
	// processing channel ids). Safe to echo in API responses.
	Config map[string]string
	// EncryptedCredentials is the AES-GCM envelope of the gateway's secret
	// credentials, sealed by port.SecretCipher with AAD (OrgId, Id). Opaque
	// here; decrypted only by the GatewayFactory at the point of use and
	// never returned by any endpoint.
	EncryptedCredentials string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
