package cli_test

import (
	"net/url"
	"testing"
)

func TestDunningCmd(t *testing.T) {
	campaignResp := `{
		"id": "dc_1",
		"subscription_id": "sub_1",
		"customer_id": "cus_1",
		"status": "active",
		"currency": "USD",
		"failed_amount": 9900,
		"total_attempts": 2,
		"next_attempt_at": "0001-01-01T00:00:00Z",
		"started_at": "2026-06-01T10:00:00Z",
		"created_at": "2026-06-01T10:00:00Z",
		"updated_at": "2026-06-01T10:00:00Z"
	}`

	configResp := `{
		"id": "dcfg_1",
		"name": "Standard",
		"status": "active",
		"priority": 10,
		"applies_to": "organization",
		"config": {},
		"is_ab_test": false,
		"created_at": "2026-06-01T10:00:00Z",
		"updated_at": "2026-06-01T10:00:00Z"
	}`

	runCases(t, []cmdCase{
		{
			name:       "campaigns list",
			args:       []string{"dunning", "campaigns", "list"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns",
			respBody: `{
				"data": [` + campaignResp + `],
				"total": 1
			}`,
			wantOut:  []string{"dc_1", "sub_1", "active", "9900", "2", "-", "total 1"},
			wantCode: 0,
		},
		{
			name:       "campaigns list json output",
			args:       []string{"dunning", "campaigns", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns",
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"total": 0`},
			wantCode:   0,
		},
		{
			name:       "campaigns get",
			args:       []string{"dunning", "campaigns", "get", "dc_1"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns/dc_1",
			respBody:   campaignResp,
			wantOut:    []string{"dc_1", "sub_1", "active"},
			wantCode:   0,
		},
		{
			name:     "campaigns get no args",
			args:     []string{"dunning", "campaigns", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		{
			name:       "campaigns update with status flag",
			args:       []string{"dunning", "campaigns", "update", "dc_1", "--status", "paused", "--reason", "investigating"},
			wantMethod: "PATCH",
			wantPath:   "/api/dunning/campaigns/dc_1",
			wantBody:   `{"status":"paused","reason":"investigating"}`,
			respBody:   campaignResp,
			wantOut:    []string{"dc_1"},
			wantCode:   0,
		},
		{
			name:       "campaigns update via --data",
			args:       []string{"dunning", "campaigns", "update", "dc_2", "--data", `{"status":"cancelled","reason":"churned"}`},
			wantMethod: "PATCH",
			wantPath:   "/api/dunning/campaigns/dc_2",
			wantBody:   `{"status":"cancelled","reason":"churned"}`,
			respBody:   campaignResp,
			wantCode:   0,
		},
		{
			name:     "campaigns update missing status",
			args:     []string{"dunning", "campaigns", "update", "dc_1"},
			wantErr:  []string{"--status is required"},
			wantCode: 2,
		},
		{
			name:       "campaigns attempts",
			args:       []string{"dunning", "campaigns", "attempts", "dc_1"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns/dc_1/attempts",
			wantQuery:  url.Values{"limit": {"10"}, "page": {"0"}},
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"data"`},
			wantCode:   0,
		},
		{
			name:       "campaigns attempts --limit 50",
			args:       []string{"dunning", "campaigns", "attempts", "dc_1", "--limit", "50"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns/dc_1/attempts",
			wantQuery:  url.Values{"limit": {"50"}},
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"data"`},
			wantCode:   0,
		},
		{
			name:       "campaigns retry with payment method",
			args:       []string{"dunning", "campaigns", "retry", "dc_1", "--payment-method", "pm_abc"},
			wantMethod: "POST",
			wantPath:   "/api/dunning/campaigns/dc_1/attempts",
			wantBody:   `{"payment_method_id":"pm_abc"}`,
			respBody:   `{"id":"da_1","dunning_campaign_id":"dc_1","status":"pending","amount":9900,"attempt_number":3,"currency":"USD","attempt_type":"manual","attempted_at":"2026-06-12T10:00:00Z","created_at":"2026-06-12T10:00:00Z"}`,
			wantOut:    []string{`"id": "da_1"`},
			wantCode:   0,
		},
		// retry without payment method — optional unset, ogen emits empty object
		{
			name:       "campaigns retry no payment method",
			args:       []string{"dunning", "campaigns", "retry", "dc_1"},
			wantMethod: "POST",
			wantPath:   "/api/dunning/campaigns/dc_1/attempts",
			wantBody:   `{}`,
			respBody:   `{"id":"da_2","dunning_campaign_id":"dc_1","status":"pending","amount":9900,"attempt_number":3,"currency":"USD","attempt_type":"manual","attempted_at":"2026-06-12T10:00:00Z","created_at":"2026-06-12T10:00:00Z"}`,
			wantCode:   0,
		},
		{
			name:       "campaigns communications",
			args:       []string{"dunning", "campaigns", "communications", "dc_1"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns/dc_1/communications",
			wantQuery:  url.Values{"limit": {"10"}, "page": {"0"}},
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"data"`},
			wantCode:   0,
		},
		{
			name:       "campaigns communications --limit 50",
			args:       []string{"dunning", "campaigns", "communications", "dc_1", "--limit", "50"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/campaigns/dc_1/communications",
			wantQuery:  url.Values{"limit": {"50"}},
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"data"`},
			wantCode:   0,
		},
		{
			name:       "configs list",
			args:       []string{"dunning", "configs", "list"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/configurations",
			respBody: `{
				"data": [` + configResp + `],
				"total": 1
			}`,
			wantOut:  []string{"dcfg_1", "Standard", "active", "10", "organization", "total 1"},
			wantCode: 0,
		},
		{
			name:       "configs get",
			args:       []string{"dunning", "configs", "get", "dcfg_1"},
			wantMethod: "GET",
			wantPath:   "/api/dunning/configurations/dcfg_1",
			respBody:   configResp,
			wantOut:    []string{"dcfg_1", "Standard"},
			wantCode:   0,
		},
		// --data round-trips through the generated request type, so it must use the
		// real spec config schema (immediate_retries / progressive_retries /
		// escalation_rules); unknown fields are silently dropped by the typed decoder.
		{
			name: "configs create via --data",
			args: []string{"dunning", "configs", "create", "--data", `{
				"name": "Standard retry",
				"applies_to": "organization",
				"config": {
					"immediate_retries": {"enabled": true, "max_attempts": 1},
					"progressive_retries": {"enabled": true, "max_attempts": 2, "intervals": ["24h", "72h"]},
					"escalation_rules": {"cancel_after_attempt": 3}
				}
			}`},
			wantMethod: "POST",
			wantPath:   "/api/dunning/configurations",
			wantBody: `{
				"name": "Standard retry",
				"applies_to": "organization",
				"config": {
					"immediate_retries": {"enabled": true, "max_attempts": 1},
					"progressive_retries": {"enabled": true, "max_attempts": 2, "intervals": ["24h", "72h"]},
					"escalation_rules": {"cancel_after_attempt": 3}
				}
			}`,
			respBody: configResp,
			wantOut:  []string{"dcfg_1"},
			wantCode: 0,
		},
		{
			name:     "configs create missing name flag",
			args:     []string{"dunning", "configs", "create", "--applies-to", "organization"},
			wantErr:  []string{"--name is required"},
			wantCode: 2,
		},
		{
			name:     "configs create missing applies-to",
			args:     []string{"dunning", "configs", "create", "--name", "test"},
			wantErr:  []string{"--applies-to is required"},
			wantCode: 2,
		},
		// only set flags are sent; priority 0 stays unset.
		{
			name:       "configs update name and status",
			args:       []string{"dunning", "configs", "update", "dcfg_1", "--name", "Updated", "--status", "inactive"},
			wantMethod: "PATCH",
			wantPath:   "/api/dunning/configurations/dcfg_1",
			wantBody:   `{"name":"Updated","status":"inactive"}`,
			respBody:   configResp,
			wantOut:    []string{"dcfg_1", "Standard"},
			wantCode:   0,
		},
	})
}

func TestPaymentTokensCmd(t *testing.T) {
	tokenResp := `{
		"token_id": "tok_abc",
		"subscription_id": "sub_1",
		"customer_id": "cus_1",
		"status": "active",
		"expires_at": "2026-06-13T10:00:00Z",
		"created_at": "2026-06-12T10:00:00Z",
		"max_uses": 3,
		"used_count": 0,
		"admin_generated": true
	}`

	runCases(t, []cmdCase{
		{
			name:       "verify token",
			args:       []string{"payment-tokens", "verify", "tok_abc"},
			wantMethod: "POST",
			wantPath:   "/api/payment-tokens/verify",
			wantBody:   `{"token_id":"tok_abc"}`,
			respBody:   tokenResp,
			wantOut:    []string{`"token_id": "tok_abc"`},
			wantCode:   0,
		},
		{
			name:     "verify no args",
			args:     []string{"payment-tokens", "verify"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		{
			name:       "activate token",
			args:       []string{"payment-tokens", "activate", "tok_abc"},
			wantMethod: "POST",
			wantPath:   "/api/payment-tokens/activate",
			wantBody:   `{"token_id":"tok_abc"}`,
			respBody:   tokenResp,
			wantOut:    []string{`"token_id": "tok_abc"`},
			wantCode:   0,
		},
		{
			name:       "create token with flags",
			args:       []string{"payment-tokens", "create", "sub_1", "--max-uses", "3", "--expiry-hours", "48", "--reason", "proactive retry", "--notes", "see ticket #123"},
			wantMethod: "POST",
			wantPath:   "/api/admin/subscriptions/sub_1/payment-tokens",
			wantBody:   `{"max_uses":3,"expiry_hours":48,"admin_reason":"proactive retry","admin_notes":"see ticket #123"}`,
			respBody:   tokenResp,
			wantOut:    []string{`"token_id": "tok_abc"`, `"max_uses": 3`},
			wantCode:   0,
		},
		{
			name:       "create token via --data",
			args:       []string{"payment-tokens", "create", "sub_2", "--data", `{"max_uses":1,"expiry_hours":24,"admin_reason":"test"}`},
			wantMethod: "POST",
			wantPath:   "/api/admin/subscriptions/sub_2/payment-tokens",
			wantBody:   `{"max_uses":1,"expiry_hours":24,"admin_reason":"test"}`,
			respBody:   tokenResp,
			wantCode:   0,
		},
		// no flags, ogen emits empty object body
		{
			name:       "create token json output",
			args:       []string{"payment-tokens", "create", "sub_1", "-o", "json"},
			wantMethod: "POST",
			wantPath:   "/api/admin/subscriptions/sub_1/payment-tokens",
			wantBody:   `{}`,
			respBody:   tokenResp,
			wantOut:    []string{`"admin_generated": true`},
			wantCode:   0,
		},
	})
}
