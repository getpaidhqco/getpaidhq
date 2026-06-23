package cli_test

import (
	"net/url"
	"testing"
)

func TestCustomersCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// create: flags path. ogen serializes the typed CreateCustomerInput and
		// omits unset optional fields (last_name, phone, billing_address), unlike
		// the old hand-marshaled body.
		{
			name:       "create with flags",
			args:       []string{"customers", "create", "--email", "ada@example.com", "--first-name", "Ada", "--metadata", "tier=gold"},
			wantMethod: "POST",
			wantPath:   "/api/customers",
			wantBody:   `{"email":"ada@example.com","first_name":"Ada","metadata":{"tier":"gold"}}`,
			respBody:   `{"id":"cus_1","email":"ada@example.com","first_name":"Ada","created_at":"2026-06-12T09:30:00Z","updated_at":"2026-06-12T09:30:00Z"}`,
			wantOut:    []string{"cus_1", "ada@example.com", "2026-06-12 09:30"},
			wantCode:   0,
		},
		// create: --data stdin. Raw JSON is decoded into the typed input then
		// re-serialized by ogen, so the body reduces to the single set field.
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
		// API error rendering. NOTE: changed from the old test's 404 to 400.
		// The generated GetCustomer operation models only 400/500 (+ a bodyless
		// default) as typed error variants; a 404 decodes to GetCustomerDef,
		// which carries no {code,message} body and renders as a generic error. A
		// 400 hits GetCustomerBadRequest (an ApiError), exercising the same
		// {code,message} envelope-rendering path the old test intended.
		{
			name:       "get api error",
			args:       []string{"customers", "get", "cus_missing"},
			wantMethod: "GET",
			wantPath:   "/api/customers/cus_missing",
			respStatus: 400,
			respBody:   `{"code":"not_found","message":"customer not found"}`,
			wantErr:    []string{"error (not_found): customer not found"},
			wantCode:   1,
		},
		// payment-methods add. This endpoint binds the port input directly in
		// PascalCase wire format (see customer_handler.go); the CLI sends a raw
		// map, so only the set keys appear (no OrgId/CustomerId/Details/etc.).
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
			wantBody:   `{"Psp":"paystack","Name":"My Card","Type":"card","Token":"tok_abc123","IsDefault":true}`,
			respBody:   `{"id":"pm_1","name":"My Card","type":"card","psp":"paystack","created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"pm_1"},
			wantCode:   0,
		},
		// payment-methods update. Unset flags are sent as empty values (the
		// server ignores them).
		{
			name: "payment-methods update",
			args: []string{"customers", "payment-methods", "update", "cus_1", "pm_1",
				"--name", "Updated Card",
			},
			wantMethod: "PUT",
			wantPath:   "/api/customers/cus_1/payment-methods/pm_1",
			wantBody:   `{"Name":"Updated Card","Type":"","Token":"","IsDefault":false}`,
			respBody:   `{"id":"pm_1","name":"Updated Card","type":"card","psp":"paystack","created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"pm_1"},
			wantCode:   0,
		},
		// dunning-history. Rendered as raw JSON (renderValue); body must be valid
		// per the CustomerDunningHistoryResponse schema.
		{
			name:       "dunning-history",
			args:       []string{"customers", "dunning-history", "cus_1"},
			wantMethod: "GET",
			wantPath:   "/api/customers/cus_1/dunning-history",
			respBody:   `{"customer_id":"cus_1","total_dunning_campaigns":0,"successful_recoveries":0}`,
			wantOut:    []string{"cus_1", "total_dunning_campaigns"},
			wantCode:   0,
		},
	})
}
