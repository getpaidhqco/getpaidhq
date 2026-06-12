package commands_test

import (
	"net/url"
	"testing"
)

func TestHealthCmd(t *testing.T) {
	runCases(t, []cmdCase{
		{
			name:       "health ok",
			args:       []string{"health"},
			wantMethod: "GET",
			wantPath:   "/api/health",
			respBody:   `{"status":"ok"}`,
			wantOut:    []string{`"status": "ok"`},
			wantCode:   0,
		},
	})
}

func TestCustomersCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// create: flags path
		{
			name:       "create with flags",
			args:       []string{"customers", "create", "--email", "ada@example.com", "--first-name", "Ada", "--metadata", "tier=gold"},
			wantMethod: "POST",
			wantPath:   "/api/customers",
			wantBody:   `{"email":"ada@example.com","first_name":"Ada","last_name":"","billing_address":{},"phone":"","metadata":{"tier":"gold"}}`,
			respBody:   `{"id":"cus_1","email":"ada@example.com","first_name":"Ada","created_at":"2026-06-12T09:30:00Z","updated_at":"2026-06-12T09:30:00Z"}`,
			wantOut:    []string{"cus_1", "ada@example.com", "2026-06-12 09:30"},
			wantCode:   0,
		},
		// create: --data stdin
		{
			name:       "create via --data stdin",
			args:       []string{"customers", "create", "--data", "-"},
			stdin:      `{"email":"bob@example.com"}`,
			wantMethod: "POST",
			wantPath:   "/api/customers",
			wantBody:   `{"email":"bob@example.com"}`,
			respBody:   `{"id":"cus_2","email":"bob@example.com","created_at":"2026-06-12T09:31:00Z","updated_at":"2026-06-12T09:31:00Z"}`,
			wantOut:    []string{"cus_2"},
			wantCode:   0,
		},
		// create: --data + flag conflict → exit 2
		{
			name:     "create data+flags conflict",
			args:     []string{"customers", "create", "--data", `{"email":"x@x.com"}`, "--email", "x@x.com"},
			wantErr:  []string{"--data cannot be combined with --email"},
			wantCode: 2,
		},
		// create: missing email → exit 2
		{
			name:     "create missing email",
			args:     []string{"customers", "create"},
			wantErr:  []string{"--email is required"},
			wantCode: 2,
		},
		// list: pagination flags and query params
		{
			name:       "list with pagination",
			args:       []string{"customers", "list", "--page", "2", "--limit", "5", "--sort-by", "email", "--sort-order", "asc"},
			wantMethod: "GET",
			wantPath:   "/api/customers",
			wantQuery: url.Values{
				"page":       []string{"2"},
				"limit":      []string{"5"},
				"sort_by":    []string{"email"},
				"sort_order": []string{"asc"},
			},
			respBody: `{"data":[{"id":"cus_3","email":"eve@example.com","created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":11,"page":2,"limit":5}}`,
			wantOut:  []string{"cus_3", "eve@example.com", "total 11 · page 2 · limit 5"},
			wantCode: 0,
		},
		// list: -o json
		{
			name:       "list json output",
			args:       []string{"customers", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/customers",
			respBody:   `{"data":[],"meta":{"total":0,"page":0,"limit":10}}`,
			wantOut:    []string{`"total": 0`},
			wantCode:   0,
		},
		// get: success
		{
			name:       "get by id",
			args:       []string{"customers", "get", "cus_9"},
			wantMethod: "GET",
			wantPath:   "/api/customers/cus_9",
			respBody:   `{"id":"cus_9","email":"z@example.com","created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"cus_9", "z@example.com"},
			wantCode:   0,
		},
		// get: no args → exit 2
		{
			name:     "get no args",
			args:     []string{"customers", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		// API error rendering
		{
			name:       "get 404 api error",
			args:       []string{"customers", "get", "cus_missing"},
			wantMethod: "GET",
			wantPath:   "/api/customers/cus_missing",
			respStatus: 404,
			respBody:   `{"code":"not_found","message":"customer not found"}`,
			wantErr:    []string{"error (not_found): customer not found"},
			wantCode:   1,
		},
		// payment-methods add
		{
			name: "payment-methods add",
			args: []string{"customers", "payment-methods", "add", "cus_1",
				"--psp", "paystack",
				"--name", "My Card",
				"--type", "card",
				"--token", "tok_abc123",
				"--default",
			},
			wantMethod: "POST",
			wantPath:   "/api/customers/cus_1/payment-methods",
			wantBody:   `{"psp":"paystack","name":"My Card","type":"card","token":"tok_abc123","is_default":true,"billing_address":{},"details":null,"metadata":null}`,
			respBody:   `{"id":"pm_1","name":"My Card","type":"card","psp":"paystack","created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"pm_1"},
			wantCode:   0,
		},
		// payment-methods update
		{
			name: "payment-methods update",
			args: []string{"customers", "payment-methods", "update", "cus_1", "pm_1",
				"--name", "Updated Card",
			},
			wantMethod: "PUT",
			wantPath:   "/api/customers/cus_1/payment-methods/pm_1",
			respBody:   `{"id":"pm_1","name":"Updated Card","type":"card","psp":"paystack","created_at":"2026-06-12T09:00:00Z"}`,
			wantCode:   0,
		},
		// dunning-history
		{
			name:       "dunning-history",
			args:       []string{"customers", "dunning-history", "cus_1"},
			wantMethod: "GET",
			wantPath:   "/api/customers/cus_1/dunning-history",
			respBody:   `{"data":[],"total":0}`,
			wantOut:    []string{`"data"`},
			wantCode:   0,
		},
	})
}
