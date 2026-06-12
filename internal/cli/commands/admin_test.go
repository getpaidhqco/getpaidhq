package commands_test

import (
	"net/url"
	"testing"
)

func TestAdminCommands(t *testing.T) {
	runCases(t, []cmdCase{
		// -----------------------------------------------------------------
		// api-keys create
		// -----------------------------------------------------------------
		{
			name:       "api-keys create happy",
			args:       []string{"api-keys", "create", "--name", "ci-deploy"},
			wantMethod: "POST",
			wantPath:   "/api/api-keys",
			wantBody:   `{"name":"ci-deploy"}`,
			respBody:   `{"id":"key_1","name":"ci-deploy","key":"sk_key_1_secret","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"key_1", "ci-deploy", "sk_key_1_secret"},
			wantErr:    []string{"store this key now"},
		},
		{
			name:       "api-keys create json mode — note still on stderr",
			args:       []string{"api-keys", "create", "-o", "json", "--name", "ci"},
			wantMethod: "POST",
			wantPath:   "/api/api-keys",
			respBody:   `{"id":"key_2","name":"ci","key":"sk_key_2_secret","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"sk_key_2_secret"},
			wantErr:    []string{"store this key now"},
		},

		// -----------------------------------------------------------------
		// api-keys list
		// -----------------------------------------------------------------
		{
			name:       "api-keys list",
			args:       []string{"api-keys", "list"},
			wantMethod: "GET",
			wantPath:   "/api/api-keys",
			respBody:   `{"data":[{"id":"key_1","name":"ci","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"key_1", "ci"},
		},

		// -----------------------------------------------------------------
		// api-keys revoke
		// -----------------------------------------------------------------
		{
			name:       "api-keys revoke",
			args:       []string{"api-keys", "revoke", "key_1"},
			wantMethod: "DELETE",
			wantPath:   "/api/api-keys/key_1",
			respStatus: 204,
			wantOut:    []string{"key_1 revoked"},
		},
		{
			name:     "api-keys revoke missing id",
			args:     []string{"api-keys", "revoke"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// orgs create
		// -----------------------------------------------------------------
		{
			name:       "orgs create happy",
			args:       []string{"orgs", "create", "--name", "Acme Corp", "--country", "NG", "--timezone", "Africa/Lagos"},
			wantMethod: "POST",
			wantPath:   "/api/organizations",
			wantBody:   `{"name":"Acme Corp","country":"NG","timezone":"Africa/Lagos","metadata":null}`,
			respBody:   `{"id":"org_1","name":"Acme Corp","country":"NG","timezone":"Africa/Lagos","status":"active","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"org_1"},
		},
		{
			name:     "orgs create missing name",
			args:     []string{"orgs", "create", "--country", "NG", "--timezone", "UTC"},
			wantCode: 2,
		},
		{
			name:     "orgs create missing country",
			args:     []string{"orgs", "create", "--name", "Test", "--timezone", "UTC"},
			wantCode: 2,
		},
		{
			name:     "orgs create missing timezone",
			args:     []string{"orgs", "create", "--name", "Test", "--country", "NG"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// gateways create — CRITICAL: credentials must pass through as
		// literal strings. Using api.CreateGatewayRequest would marshal
		// domain.Secret values as "[redacted]" and silently break creation.
		// This assertion is the regression guard against that mistake.
		// -----------------------------------------------------------------
		{
			name: "gateways create happy — literal credential passthrough",
			args: []string{
				"gateways", "create",
				"--name", "prod-paystack",
				"--psp", "paystack",
				"--credential", "secret_key=sk_live_x",
			},
			wantMethod: "POST",
			wantPath:   "/api/gateways",
			// CRITICAL: credentials must arrive as literal strings, NOT "[redacted]".
			// Using api.CreateGatewayRequest (with map[string]domain.Secret) would
			// marshal to "[redacted]" — we build a plain map to prevent that.
			wantBody: `{"name":"prod-paystack","psp":"paystack","config":null,"credentials":{"secret_key":"sk_live_x"}}`,
			respBody: `{"id":"gw_1","name":"prod-paystack","psp":"paystack","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:  []string{"gw_1"},
		},
		{
			name: "gateways create with config and multiple credentials",
			args: []string{
				"gateways", "create",
				"--name", "checkout-prod",
				"--psp", "checkout_com",
				"--config", "processing_channel_id=pc_abc",
				"--credential", "secret_key=sk_live_a",
				"--credential", "public_key=pk_live_b",
			},
			wantMethod: "POST",
			wantPath:   "/api/gateways",
			wantBody:   `{"name":"checkout-prod","psp":"checkout_com","config":{"processing_channel_id":"pc_abc"},"credentials":{"secret_key":"sk_live_a","public_key":"pk_live_b"}}`,
			respBody:   `{"id":"gw_2","name":"checkout-prod","psp":"checkout_com","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"gw_2"},
		},
		{
			name:     "gateways create missing name",
			args:     []string{"gateways", "create", "--psp", "paystack", "--credential", "k=v"},
			wantCode: 2,
		},
		{
			name:     "gateways create missing credential",
			args:     []string{"gateways", "create", "--name", "p", "--psp", "paystack"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// settings create
		// -----------------------------------------------------------------
		{
			name:       "settings create happy",
			args:       []string{"settings", "create", "--id", "theme", "--value", "dark"},
			wantMethod: "POST",
			wantPath:   "/api/settings",
			wantBody:   `{"parent_id":"","id":"theme","type":"","value":"dark"}`,
			respBody:   `{"parent_id":"","id":"theme","type":"","value":"dark","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"theme", "dark"},
		},
		{
			name:       "settings create with parent",
			args:       []string{"settings", "create", "--parent", "ui", "--id", "color", "--type", "string", "--value", "blue"},
			wantMethod: "POST",
			wantPath:   "/api/settings",
			wantBody:   `{"parent_id":"ui","id":"color","type":"string","value":"blue"}`,
			respBody:   `{"parent_id":"ui","id":"color","type":"string","value":"blue","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"ui", "color", "blue"},
		},
		{
			name:     "settings create missing id",
			args:     []string{"settings", "create", "--value", "x"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// settings list
		// -----------------------------------------------------------------
		{
			name:       "settings list no parent",
			args:       []string{"settings", "list"},
			wantMethod: "GET",
			wantPath:   "/api/settings",
			respBody:   `{"data":[{"parent_id":"ui","id":"color","type":"string","value":"blue","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"color", "blue"},
		},
		{
			name:       "settings list with parent filter",
			args:       []string{"settings", "list", "--parent", "ui"},
			wantMethod: "GET",
			wantPath:   "/api/settings",
			wantQuery:  url.Values{"parent_id": []string{"ui"}},
			respBody:   `{"data":[],"meta":{"total":0,"page":0,"limit":10}}`,
		},

		// -----------------------------------------------------------------
		// settings get
		// -----------------------------------------------------------------
		{
			name:       "settings get",
			args:       []string{"settings", "get", "ui", "color"},
			wantMethod: "GET",
			wantPath:   "/api/settings/ui/color",
			respBody:   `{"parent_id":"ui","id":"color","type":"string","value":"blue","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"ui", "color", "blue"},
		},
		{
			name:     "settings get missing args",
			args:     []string{"settings", "get", "ui"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// settings update
		// -----------------------------------------------------------------
		{
			name:       "settings update",
			args:       []string{"settings", "update", "ui", "color", "--value", "red"},
			wantMethod: "PUT",
			wantPath:   "/api/settings/ui/color",
			wantBody:   `{"type":"","value":"red"}`,
			respBody:   `{"parent_id":"ui","id":"color","type":"string","value":"red","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"red"},
		},
		{
			name:     "settings update missing args",
			args:     []string{"settings", "update", "ui"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// settings delete
		// -----------------------------------------------------------------
		{
			name:       "settings delete",
			args:       []string{"settings", "delete", "ui", "color"},
			wantMethod: "DELETE",
			wantPath:   "/api/settings/ui/color",
			respStatus: 204,
			wantOut:    []string{"ui/color deleted"},
		},
		{
			name:     "settings delete missing args",
			args:     []string{"settings", "delete", "ui"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// webhooks create
		// -----------------------------------------------------------------
		{
			name: "webhooks create happy",
			args: []string{
				"webhooks", "create",
				"--url", "https://example.com/hook",
				"--event", "subscription.created",
				"--event", "payment.succeeded",
			},
			wantMethod: "POST",
			wantPath:   "/api/webhooks",
			wantBody:   `{"url":"https://example.com/hook","events":["subscription.created","payment.succeeded"],"secret":""}`,
			respBody:   `{"id":"wh_1","url":"https://example.com/hook","events":["subscription.created","payment.succeeded"],"created_at":"2024-01-01T00:00:00Z"}`,
			wantOut:    []string{"wh_1"},
		},
		{
			name:     "webhooks create missing url",
			args:     []string{"webhooks", "create", "--event", "subscription.created"},
			wantCode: 2,
		},
		{
			name:     "webhooks create missing event",
			args:     []string{"webhooks", "create", "--url", "https://example.com/hook"},
			wantCode: 2,
		},

		// -----------------------------------------------------------------
		// webhooks list
		// -----------------------------------------------------------------
		{
			name:       "webhooks list",
			args:       []string{"webhooks", "list"},
			wantMethod: "GET",
			wantPath:   "/api/webhooks",
			respBody:   `{"data":[],"meta":{"total":0,"page":0,"limit":0}}`,
			wantOut:    []string{"data"},
		},
	})
}
