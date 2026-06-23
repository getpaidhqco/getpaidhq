package cli_test

import (
	"net/url"
	"testing"
)

func TestOrdersCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// orders create — flags path
		// ----------------------------------------------------------------
		{
			name: "orders create flags",
			args: []string{
				"orders", "create",
				"--customer", "cus_1",
				"--psp", "paystack",
				"--currency", "NGN",
				"--item", "product=prod_1,price=pri_1",
				"--item", "product=prod_2,price=pri_2,qty=3",
				"--metadata", "src=web",
			},
			wantMethod: "POST",
			wantPath:   "/api/orders",
			// ogen serializes typed inputs with spec JSON field names and omits
			// unset optionals; only psp_id + customer are required-or-set here.
			wantBody: `{
				"psp_id":"paystack",
				"customer":{"id":"cus_1"},
				"cart":{"currency":"NGN","items":[
					{"product_id":"prod_1","price_id":"pri_1","quantity":1},
					{"product_id":"prod_2","price_id":"pri_2","quantity":3}
				]},
				"metadata":{"src":"web"}
			}`,
			respBody: `{"order":{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"pending","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"},"psp":null}`,
			wantOut:  []string{"ord_1", "cus_1", "REF001", "pending", "NGN", "5000"},
			wantCode: 0,
		},
		// orders create — missing --psp → exit 2
		{
			name:     "orders create missing psp",
			args:     []string{"orders", "create", "--customer", "cus_1"},
			wantErr:  []string{"--psp is required"},
			wantCode: 2,
		},
		// orders create — missing customer → exit 2
		{
			name:     "orders create missing customer",
			args:     []string{"orders", "create", "--psp", "paystack"},
			wantErr:  []string{"provide --customer or --email"},
			wantCode: 2,
		},
		// orders create — bad --item (no price) → exit 2
		{
			name:     "orders create bad item no price",
			args:     []string{"orders", "create", "--customer", "cus_1", "--psp", "paystack", "--item", "product=prod_1"},
			wantErr:  []string{"--item needs product=<id>,price=<id>"},
			wantCode: 2,
		},
		// orders create — bad --item qty not integer → exit 2
		{
			name:     "orders create bad item qty",
			args:     []string{"orders", "create", "--customer", "cus_1", "--psp", "paystack", "--item", "product=prod_1,price=pri_1,qty=abc"},
			wantErr:  []string{"--item qty must be a positive integer"},
			wantCode: 2,
		},
		// orders create — unknown --item key (typo) → exit 2
		{
			name:     "orders create unknown item key",
			args:     []string{"orders", "create", "--customer", "cus_1", "--psp", "paystack", "--item", "product=prod_1,price=pri_1,quanity=5"},
			wantErr:  []string{"unknown key"},
			wantCode: 2,
		},
		// orders create -o json — the command renders only the order envelope's
		// nested order object (not the psp) as pretty JSON via renderOne.
		{
			name:       "orders create json mode",
			args:       []string{"-o", "json", "orders", "create", "--customer", "cus_1", "--psp", "paystack", "--currency", "NGN", "--item", "product=prod_1,price=pri_1"},
			wantMethod: "POST",
			wantPath:   "/api/orders",
			// Fully-populated order: ogen's per-field MarshalJSON emits nothing
			// for unset optionals, so every optional field is set here.
			respBody: `{"order":{"id":"ord_3","cart_id":"cart_1","session_id":"sess_1","customer_id":"cus_1","reference":"REF003","status":"pending","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z","metadata":{"k":"v"},"items":[],"customer":{"id":"cus_1","email":"a@b.c","created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z","billing_address":{},"first_name":null,"last_name":null,"name":null,"phone":null,"metadata":null}},"psp":{"id":"paystack"}}`,
			wantOut:  []string{`"id": "ord_3"`, `"reference": "REF003"`, `"status": "pending"`},
			wantCode: 0,
		},
		// orders create — via --data inline JSON (decoded into the typed body,
		// re-encoded by ogen → same key set as the verbatim input here).
		{
			name:       "orders create via data",
			args:       []string{"orders", "create", "--data", `{"psp_id":"paystack","customer":{"id":"cus_1"}}`},
			wantMethod: "POST",
			wantPath:   "/api/orders",
			wantBody:   `{"psp_id":"paystack","customer":{"id":"cus_1"}}`,
			respBody:   `{"order":{"id":"ord_2","customer_id":"cus_1","reference":"REF002","status":"pending","currency":"USD","total":0,"created_at":"2026-06-12T09:01:00Z"},"psp":null}`,
			wantOut:    []string{"ord_2"},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// orders complete
		// ----------------------------------------------------------------
		{
			name:       "orders complete with payment-method",
			args:       []string{"orders", "complete", "ord_1", "--payment-method", "pm_1"},
			wantMethod: "POST",
			wantPath:   "/api/orders/ord_1/complete",
			// only payment_method_id is set; all other fields are unset optionals.
			wantBody: `{"payment_method_id":"pm_1"}`,
			respBody: `{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"completed","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:  []string{"ord_1", "completed"},
			wantCode: 0,
		},
		// orders complete — via --data stdin
		{
			name:       "orders complete via data stdin",
			args:       []string{"orders", "complete", "ord_1", "--data", "-"},
			stdin:      `{"payment_method_id":"pm_2"}`,
			wantMethod: "POST",
			wantPath:   "/api/orders/ord_1/complete",
			wantBody:   `{"payment_method_id":"pm_2"}`,
			respBody:   `{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"completed","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"ord_1"},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// orders get
		// ----------------------------------------------------------------
		{
			name:       "orders get",
			args:       []string{"orders", "get", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1",
			respBody:   `{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"pending","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"ord_1", "cus_1", "REF001", "pending"},
			wantCode:   0,
		},
		// orders get — no args → exit 2
		{
			name:     "orders get no args",
			args:     []string{"orders", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		// ----------------------------------------------------------------
		// orders list
		// ----------------------------------------------------------------
		{
			name:       "orders list with pagination",
			args:       []string{"orders", "list", "--page", "1", "--limit", "5"},
			wantMethod: "GET",
			wantPath:   "/api/orders",
			wantQuery: url.Values{
				"page":  []string{"1"},
				"limit": []string{"5"},
			},
			respBody: `{"data":[{"id":"ord_9","customer_id":"cus_1","reference":"REF009","status":"pending","currency":"USD","total":1000,"created_at":"2026-06-12T09:00:00Z"}],"meta":{"total":9,"page":1,"limit":5}}`,
			wantOut:  []string{"ord_9", "total 9 · page 1 · limit 5"},
			wantCode: 0,
		},
		// ----------------------------------------------------------------
		// orders subscriptions — plain array of SubscriptionResponse
		// ----------------------------------------------------------------
		{
			name:       "orders subscriptions",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			// ogen SubscriptionResponse: snake_case JSON field names.
			respBody: `[{"id":"sub_1","status":"active","currency":"NGN","billing_interval":"month","billing_interval_qty":1,"renews_at":"2026-07-12T09:00:00Z","created_at":"2026-06-12T09:00:00Z"}]`,
			wantOut:  []string{"sub_1", "active", "NGN"},
			wantCode: 0,
		},
		// orders subscriptions — timestamps render in "2006-01-02 15:04" form
		{
			name:       "orders subscriptions timestamp format",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"id":"sub_2","status":"active","currency":"USD","billing_interval":"month","billing_interval_qty":1,"renews_at":"2026-07-12T09:00:00Z","created_at":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{"2026-07-12 09:00", "2026-06-12 09:00"},
			wantCode:   0,
		},
		// orders subscriptions — unset renews_at renders as "-"
		{
			name:       "orders subscriptions zero renews-at",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"id":"sub_3","status":"active","currency":"USD","billing_interval":"month","billing_interval_qty":1,"created_at":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{"sub_3", "-"},
			wantCode:   0,
		},
		// orders subscriptions -o json — typed array marshaled back to JSON
		{
			name:       "orders subscriptions json mode",
			args:       []string{"-o", "json", "orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"id":"sub_4","status":"active","currency":"NGN","billing_interval":"month","billing_interval_qty":1,"renews_at":"2026-07-12T09:00:00Z","created_at":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{`"id": "sub_4"`, `"status": "active"`},
			wantCode:   0,
		},
		// orders subscriptions — no args → exit 2
		{
			name:     "orders subscriptions no args",
			args:     []string{"orders", "subscriptions"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
	})
}
