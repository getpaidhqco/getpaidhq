package commands_test

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
				"--customer-id", "cus_1",
				"--psp", "paystack",
				"--currency", "NGN",
				"--item", "product=prod_1,price=pri_1",
				"--item", "product=prod_2,price=pri_2,qty=3",
				"--metadata", "src=web",
			},
			wantMethod: "POST",
			wantPath:   "/api/orders",
			wantBody: `{
				"customer":{"id":"cus_1","email":"","first_name":"","last_name":"","phone":"","metadata":null},
				"payment_method_id":"",
				"session_id":"",
				"psp_id":"paystack",
				"cart":{"currency":"NGN","items":[
					{"product_id":"prod_1","price_id":"pri_1","quantity":1},
					{"product_id":"prod_2","price_id":"pri_2","quantity":3}
				]},
				"metadata":{"src":"web"},
				"options":null
			}`,
			respBody: `{"order":{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"pending","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"},"psp":null}`,
			wantOut:  []string{"ord_1", "cus_1", "REF001", "pending", "NGN", "5000"},
			wantCode: 0,
		},
		// orders create — missing --psp → exit 2
		{
			name:     "orders create missing psp",
			args:     []string{"orders", "create", "--customer-id", "cus_1"},
			wantErr:  []string{"--psp is required"},
			wantCode: 2,
		},
		// orders create — missing customer → exit 2
		{
			name:     "orders create missing customer",
			args:     []string{"orders", "create", "--psp", "paystack"},
			wantErr:  []string{"provide --customer-id or --email"},
			wantCode: 2,
		},
		// orders create — bad --item (no price) → exit 2
		{
			name:     "orders create bad item no price",
			args:     []string{"orders", "create", "--customer-id", "cus_1", "--psp", "paystack", "--item", "product=prod_1"},
			wantErr:  []string{"--item needs product=<id>,price=<id>"},
			wantCode: 2,
		},
		// orders create — bad --item qty not integer → exit 2
		{
			name:     "orders create bad item qty",
			args:     []string{"orders", "create", "--customer-id", "cus_1", "--psp", "paystack", "--item", "product=prod_1,price=pri_1,qty=abc"},
			wantErr:  []string{"--item qty must be a positive integer"},
			wantCode: 2,
		},
		// orders create — unknown --item key (typo) → exit 2
		{
			name:     "orders create unknown item key",
			args:     []string{"orders", "create", "--customer-id", "cus_1", "--psp", "paystack", "--item", "product=prod_1,price=pri_1,quanity=5"},
			wantErr:  []string{"unknown key"},
			wantCode: 2,
		},
		// orders create -o json — raw envelope passes through (incl. "psp" key)
		{
			name:       "orders create json mode",
			args:       []string{"-o", "json", "orders", "create", "--customer-id", "cus_1", "--psp", "paystack", "--currency", "NGN", "--item", "product=prod_1,price=pri_1"},
			wantMethod: "POST",
			wantPath:   "/api/orders",
			respBody:   `{"order":{"id":"ord_3","customer_id":"cus_1","reference":"REF003","status":"pending","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"},"psp":{"id":"paystack"}}`,
			wantOut:    []string{`"psp"`, `"id": "ord_3"`},
			wantCode:   0,
		},
		// orders create — via --data inline JSON (verbatim passthrough)
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
			name:       "orders complete with payment-method-id",
			args:       []string{"orders", "complete", "ord_1", "--payment-method-id", "pm_1"},
			wantMethod: "POST",
			wantPath:   "/api/orders/ord_1/complete",
			wantBody:   `{"payment_method_id":"pm_1","payment_method":{"psp":"","name":"","is_default":false,"billing_address":{"first_name":"","last_name":"","email":"","phone":"","line1":"","line2":"","city":"","state":"","postal_code":"","country":""},"type":"","details":null,"token":"","metadata":null},"payment":{"psp_id":"","reference":"","amount":0,"completed_at":"","metadata":null,"currency":""},"metadata":null}`,
			respBody:   `{"id":"ord_1","customer_id":"cus_1","reference":"REF001","status":"completed","currency":"NGN","total":5000,"created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"ord_1", "completed"},
			wantCode:   0,
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
		// orders subscriptions — plain array (not enveloped)
		// ----------------------------------------------------------------
		{
			name:       "orders subscriptions",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			// Handler returns []domain.Subscription — no JSON tags, field names are capitalized
			respBody: `[{"Id":"sub_1","Status":"active","Currency":"NGN","BillingInterval":"month","BillingIntervalQty":1,"RenewsAt":"2026-07-12T09:00:00Z","CreatedAt":"2026-06-12T09:00:00Z"}]`,
			wantOut:  []string{"sub_1", "active", "NGN"},
			wantCode: 0,
		},
		// orders subscriptions — timestamps render in "2006-01-02 15:04" form
		{
			name:       "orders subscriptions timestamp format",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"Id":"sub_2","Status":"active","Currency":"USD","BillingInterval":"month","BillingIntervalQty":1,"RenewsAt":"2026-07-12T09:00:00Z","CreatedAt":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{"2026-07-12 09:00", "2026-06-12 09:00"},
			wantCode:   0,
		},
		// orders subscriptions — zero RenewsAt renders as "-"
		{
			name:       "orders subscriptions zero renews-at",
			args:       []string{"orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"Id":"sub_3","Status":"active","Currency":"USD","BillingInterval":"month","BillingIntervalQty":1,"RenewsAt":"0001-01-01T00:00:00Z","CreatedAt":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{"sub_3", "-"},
			wantCode:   0,
		},
		// orders subscriptions -o json — raw array passthrough
		{
			name:       "orders subscriptions json mode",
			args:       []string{"-o", "json", "orders", "subscriptions", "ord_1"},
			wantMethod: "GET",
			wantPath:   "/api/orders/ord_1/subscriptions",
			respBody:   `[{"Id":"sub_4","Status":"active","Currency":"NGN","BillingInterval":"month","BillingIntervalQty":1,"RenewsAt":"2026-07-12T09:00:00Z","CreatedAt":"2026-06-12T09:00:00Z"}]`,
			wantOut:    []string{`"Id": "sub_4"`, `"Status": "active"`},
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

func TestCartsCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// carts add — happy path
		{
			name:       "carts add happy",
			args:       []string{"carts", "add", "cart_1", "--product", "prod_1", "--price", "pri_1", "--qty", "2"},
			wantMethod: "POST",
			wantPath:   "/api/carts/cart_1/add",
			wantBody:   `{"product_id":"prod_1","price_id":"pri_1","quantity":2}`,
			respBody:   `{"id":"cart_1","items":[]}`,
			wantOut:    []string{"cart_1"},
			wantCode:   0,
		},
		// carts add — default qty=1
		{
			name:       "carts add default qty",
			args:       []string{"carts", "add", "cart_1", "--product", "prod_1", "--price", "pri_1"},
			wantMethod: "POST",
			wantPath:   "/api/carts/cart_1/add",
			wantBody:   `{"product_id":"prod_1","price_id":"pri_1","quantity":1}`,
			respBody:   `{"id":"cart_1","items":[]}`,
			wantCode:   0,
		},
		// carts add — missing product → exit 2
		{
			name:     "carts add missing product",
			args:     []string{"carts", "add", "cart_1", "--price", "pri_1"},
			wantErr:  []string{"--product and --price are required"},
			wantCode: 2,
		},
		// carts add — missing price → exit 2
		{
			name:     "carts add missing price",
			args:     []string{"carts", "add", "cart_1", "--product", "prod_1"},
			wantErr:  []string{"--product and --price are required"},
			wantCode: 2,
		},
		// carts add — --qty 0 → exit 2
		{
			name:     "carts add qty zero",
			args:     []string{"carts", "add", "cart_1", "--product", "prod_1", "--price", "pri_1", "--qty", "0"},
			wantErr:  []string{"--qty must be a positive integer"},
			wantCode: 2,
		},
		// carts remove — happy path
		{
			name:       "carts remove happy",
			args:       []string{"carts", "remove", "cart_1", "--item-id", "item_1"},
			wantMethod: "POST",
			wantPath:   "/api/carts/cart_1/remove",
			wantBody:   `{"org_id":"","id":"item_1"}`,
			respBody:   `{"id":"cart_1","items":[]}`,
			wantOut:    []string{"cart_1"},
			wantCode:   0,
		},
		// carts remove — missing item-id → exit 2
		{
			name:     "carts remove missing item-id",
			args:     []string{"carts", "remove", "cart_1"},
			wantErr:  []string{"--item-id is required"},
			wantCode: 2,
		},
	})
}

func TestSessionsCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// sessions create — happy path
		{
			name:       "sessions create happy",
			args:       []string{"sessions", "create", "--currency", "USD", "--country", "US", "--metadata", "src=api"},
			wantMethod: "POST",
			wantPath:   "/api/sessions",
			wantBody:   `{"currency":"USD","country":"US","metadata":{"src":"api"}}`,
			respBody:   `{"id":"sess_1","currency":"USD","country":"US","created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"sess_1"},
			wantCode:   0,
		},
		// sessions create — missing currency → exit 2
		{
			name:     "sessions create missing currency",
			args:     []string{"sessions", "create", "--country", "US"},
			wantErr:  []string{"--currency and --country are required"},
			wantCode: 2,
		},
		// sessions create — missing country → exit 2
		{
			name:     "sessions create missing country",
			args:     []string{"sessions", "create", "--currency", "USD"},
			wantErr:  []string{"--currency and --country are required"},
			wantCode: 2,
		},
		// sessions create — via --data inline
		{
			name:       "sessions create via data",
			args:       []string{"sessions", "create", "--data", `{"currency":"EUR","country":"DE"}`},
			wantMethod: "POST",
			wantPath:   "/api/sessions",
			wantBody:   `{"currency":"EUR","country":"DE"}`,
			respBody:   `{"id":"sess_2","currency":"EUR","country":"DE","created_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"sess_2"},
			wantCode:   0,
		},
	})
}
