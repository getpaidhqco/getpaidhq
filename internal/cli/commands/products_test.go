package commands_test

import (
	"net/url"
	"testing"
)

func TestProductsCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// ---- products create ----

		// create via --data (realistic path — the API requires >=1 variant)
		{
			name: "products create via --data",
			args: []string{"products", "create", "--data",
				`{"name":"Acme Pro","variants":[{"name":"Standard","prices":[{"category":"subscription","scheme":"fixed","currency":"USD","unit_price":999,"billing_interval":"month","billing_interval_qty":1}]}]}`},
			wantMethod: "POST",
			wantPath:   "/api/products",
			wantBody:   `{"name":"Acme Pro","variants":[{"name":"Standard","prices":[{"category":"subscription","scheme":"fixed","currency":"USD","unit_price":999,"billing_interval":"month","billing_interval_qty":1}]}]}`,
			respBody:   `{"id":"prod_1","name":"Acme Pro","status":"active","variants":[{"id":"var_1","name":"Standard","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "Acme Pro", "active", "1"},
			wantCode:   0,
		},
		// create missing --name → exit 2 (server would also reject, but CLI validates first)
		{
			name:     "products create missing name",
			args:     []string{"products", "create"},
			wantErr:  []string{"--name is required"},
			wantCode: 2,
		},

		// ---- products list ----

		{
			name:       "products list with --status archived",
			args:       []string{"products", "list", "--status", "archived"},
			wantMethod: "GET",
			wantPath:   "/api/products",
			wantQuery:  url.Values{"status": []string{"archived"}},
			respBody:   `{"data":[{"id":"prod_2","name":"Old Product","status":"archived","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"prod_2", "Old Product", "archived", "total 1"},
			wantCode:   0,
		},
		{
			name:       "products list without --status (param absent)",
			args:       []string{"products", "list"},
			wantMethod: "GET",
			wantPath:   "/api/products",
			wantQuery:  url.Values{},
			respBody:   `{"data":[{"id":"prod_3","name":"Active Product","status":"active","variants":[{"id":"var_2","name":"V1","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":5,"page":0,"limit":10}}`,
			wantOut:    []string{"prod_3", "Active Product", "active", "total 5 · page 0 · limit 10"},
			wantCode:   0,
		},

		// ---- products get ----

		{
			name:       "products get",
			args:       []string{"products", "get", "prod_1"},
			wantMethod: "GET",
			wantPath:   "/api/products/prod_1",
			respBody:   `{"id":"prod_1","name":"Acme Pro","status":"active","variants":[{"id":"var_1","name":"Standard","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "Acme Pro", "active"},
			wantCode:   0,
		},

		// ---- products update ----

		{
			name:       "products update happy path",
			args:       []string{"products", "update", "prod_1", "--name", "Acme Pro v2", "--description", "Updated desc"},
			wantMethod: "PATCH",
			wantPath:   "/api/products/prod_1",
			wantBody:   `{"name":"Acme Pro v2","description":"Updated desc","metadata":null}`,
			respBody:   `{"id":"prod_1","name":"Acme Pro v2","status":"active","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "Acme Pro v2"},
			wantCode:   0,
		},

		// ---- products delete ----

		{
			name:       "products delete",
			args:       []string{"products", "delete", "prod_1"},
			wantMethod: "DELETE",
			wantPath:   "/api/products/prod_1",
			respStatus: 204,
			respBody:   "",
			wantOut:    []string{"prod_1 deleted"},
			wantCode:   0,
		},
		{
			name:       "products delete -o json",
			args:       []string{"products", "delete", "prod_1", "-o", "json"},
			wantMethod: "DELETE",
			wantPath:   "/api/products/prod_1",
			respStatus: 204,
			respBody:   "",
			wantOut:    []string{},
			wantCode:   0,
		},

		// ---- products archive / unarchive ----

		{
			name:       "products archive",
			args:       []string{"products", "archive", "prod_1"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/archive",
			// archive sends no body
			wantBody: ``,
			respBody: `{"id":"prod_1","name":"Acme Pro","status":"archived","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:  []string{"prod_1", "archived"},
			wantCode: 0,
		},
		{
			name:       "products unarchive",
			args:       []string{"products", "unarchive", "prod_1"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/unarchive",
			wantBody:   ``,
			respBody:   `{"id":"prod_1","name":"Acme Pro","status":"active","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "active"},
			wantCode:   0,
		},

		// ---- products variants list ----
		// Server returns ListResponse ({data, meta} envelope) for ListVariants.

		{
			name:       "products variants list",
			args:       []string{"products", "variants", "list", "prod_1"},
			wantMethod: "GET",
			wantPath:   "/api/products/prod_1/variants",
			respBody:   `{"data":[{"id":"var_1","name":"Standard","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"var_1", "Standard", "0"},
			wantCode:   0,
		},

		// ---- products variants add ----

		{
			name: "products variants add",
			args: []string{"products", "variants", "add", "prod_1",
				"--name", "Premium"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/variants",
			wantBody:   `{"name":"Premium","description":"","metadata":null}`,
			respBody:   `{"id":"var_2","name":"Premium","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"var_2", "Premium"},
			wantCode:   0,
		},
	})
}

func TestVariantsCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// ---- variants get ----

		{
			name:       "variants get",
			args:       []string{"variants", "get", "var_1"},
			wantMethod: "GET",
			wantPath:   "/api/variants/var_1",
			respBody:   `{"id":"var_1","name":"Standard","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"var_1", "Standard"},
			wantCode:   0,
		},

		// ---- variants update ----

		{
			name:       "variants update",
			args:       []string{"variants", "update", "var_1", "--name", "Standard v2"},
			wantMethod: "PUT",
			wantPath:   "/api/variants/var_1",
			wantBody:   `{"name":"Standard v2","description":"","metadata":null}`,
			respBody:   `{"id":"var_1","name":"Standard v2","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"var_1", "Standard v2"},
			wantCode:   0,
		},

		// ---- variants delete ----

		{
			name:       "variants delete",
			args:       []string{"variants", "delete", "var_1"},
			wantMethod: "DELETE",
			wantPath:   "/api/variants/var_1",
			respStatus: 204,
			respBody:   "",
			wantOut:    []string{"var_1 deleted"},
			wantCode:   0,
		},

		// ---- variants prices (list prices for a variant) ----
		// Server returns ListResponse envelope for ListPrices.

		{
			name:       "variants prices",
			args:       []string{"variants", "prices", "var_1"},
			wantMethod: "GET",
			wantPath:   "/api/variants/var_1/prices",
			respBody:   `{"data":[{"id":"pri_1","variant_id":"var_1","label":"Monthly","category":"subscription","scheme":"fixed","cycles":0,"currency":"USD","unit_price":999,"unit_count":0,"min_price":0,"suggested_price":0,"billing_interval":"month","billing_interval_qty":1,"trial_interval":"","trial_interval_qty":0,"tax_code":"","billable_metric_id":"","tiers":[],"filter_field":"","filter_value":"","prorate_on_increase":false,"credit_on_decrease":false,"metadata":null,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"pri_1", "Monthly", "subscription", "fixed", "USD", "1 month"},
			wantCode:   0,
		},
	})
}

func TestPricesCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// ---- prices create happy path ----

		{
			name: "prices create happy",
			args: []string{"prices", "create",
				"--variant", "var_1",
				"--category", "subscription",
				"--scheme", "fixed",
				"--currency", "USD",
				"--unit-price", "999",
				"--label", "Monthly",
				"--interval", "month",
				"--interval-qty", "1",
			},
			wantMethod: "POST",
			wantPath:   "/api/prices",
			// All marshaled fields — including zero values (int64/int fields marshal as 0)
			wantBody: `{
				"variant_id":"var_1",
				"category":"subscription",
				"scheme":"fixed",
				"cycles":0,
				"label":"Monthly",
				"currency":"USD",
				"unit_price":999,
				"unit_count":0,
				"min_price":0,
				"suggested_price":0,
				"billing_interval":"month",
				"billing_interval_qty":1,
				"trial_interval":"",
				"trial_interval_qty":0,
				"tax_code":"",
				"billable_metric_id":"",
				"tiers":null,
				"filter_field":"",
				"filter_value":"",
				"prorate_on_increase":false,
				"credit_on_decrease":false,
				"metadata":null
			}`,
			respBody: `{"id":"pri_1","variant_id":"var_1","label":"Monthly","category":"subscription","scheme":"fixed","cycles":0,"currency":"USD","unit_price":999,"unit_count":0,"min_price":0,"suggested_price":0,"billing_interval":"month","billing_interval_qty":1,"trial_interval":"","trial_interval_qty":0,"tax_code":"","billable_metric_id":"","tiers":[],"filter_field":"","filter_value":"","prorate_on_increase":false,"credit_on_decrease":false,"metadata":null,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:  []string{"pri_1", "Monthly", "subscription", "fixed", "USD", "1 month"},
			wantCode: 0,
		},

		// create missing --currency → exit 2
		{
			name:     "prices create missing currency",
			args:     []string{"prices", "create", "--variant", "var_1", "--category", "subscription", "--scheme", "fixed"},
			wantErr:  []string{"--variant, --category, --scheme and --currency are required"},
			wantCode: 2,
		},

		// ---- prices get ----

		{
			name:       "prices get",
			args:       []string{"prices", "get", "pri_1"},
			wantMethod: "GET",
			wantPath:   "/api/prices/pri_1",
			respBody:   `{"id":"pri_1","variant_id":"var_1","label":"Monthly","category":"subscription","scheme":"fixed","cycles":0,"currency":"USD","unit_price":999,"unit_count":0,"min_price":0,"suggested_price":0,"billing_interval":"month","billing_interval_qty":1,"trial_interval":"","trial_interval_qty":0,"tax_code":"","billable_metric_id":"","tiers":[],"filter_field":"","filter_value":"","prorate_on_increase":false,"credit_on_decrease":false,"metadata":null,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"pri_1", "Monthly", "subscription", "fixed", "USD"},
			wantCode:   0,
		},

		// ---- prices update ----
		// Uses PATCH; body should NOT contain variant_id

		{
			name: "prices update",
			args: []string{"prices", "update", "pri_1",
				"--category", "subscription",
				"--scheme", "fixed",
				"--currency", "USD",
				"--unit-price", "1299",
				"--label", "Monthly Plus",
				"--interval", "month",
				"--interval-qty", "1",
			},
			wantMethod: "PATCH",
			wantPath:   "/api/prices/pri_1",
			// variant_id must be empty/absent — update does not set it
			wantBody: `{
				"variant_id":"",
				"category":"subscription",
				"scheme":"fixed",
				"cycles":0,
				"label":"Monthly Plus",
				"currency":"USD",
				"unit_price":1299,
				"unit_count":0,
				"min_price":0,
				"suggested_price":0,
				"billing_interval":"month",
				"billing_interval_qty":1,
				"trial_interval":"",
				"trial_interval_qty":0,
				"tax_code":"",
				"billable_metric_id":"",
				"tiers":null,
				"filter_field":"",
				"filter_value":"",
				"prorate_on_increase":false,
				"credit_on_decrease":false,
				"metadata":null
			}`,
			respBody: `{"id":"pri_1","variant_id":"var_1","label":"Monthly Plus","category":"subscription","scheme":"fixed","cycles":0,"currency":"USD","unit_price":1299,"unit_count":0,"min_price":0,"suggested_price":0,"billing_interval":"month","billing_interval_qty":1,"trial_interval":"","trial_interval_qty":0,"tax_code":"","billable_metric_id":"","tiers":[],"filter_field":"","filter_value":"","prorate_on_increase":false,"credit_on_decrease":false,"metadata":null,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:  []string{"pri_1", "Monthly Plus"},
			wantCode: 0,
		},

		// ---- prices delete ----

		{
			name:       "prices delete",
			args:       []string{"prices", "delete", "pri_1"},
			wantMethod: "DELETE",
			wantPath:   "/api/prices/pri_1",
			respStatus: 204,
			respBody:   "",
			wantOut:    []string{"pri_1 deleted"},
			wantCode:   0,
		},
	})
}
