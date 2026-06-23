package cli_test

import (
	"testing"
)

func TestProductsCmd(t *testing.T) {
	runCases(t, []cmdCase{
		// ---- products create ----

		// create via --data (realistic path — the API requires >=1 variant).
		// The raw JSON is decoded into CreateProductRequest then re-serialized by
		// ogen; every field in the input is a set field, so the body round-trips
		// unchanged.
		{
			name: "products create via --data",
			args: []string{"products", "create", "--data",
				`{"name":"Acme Pro","variants":[{"name":"Standard","prices":[{"category":"subscription","scheme":"fixed","currency":"USD","unit_price":999,"billing_interval":"month","billing_interval_qty":1}]}]}`},
			wantMethod: "POST",
			wantPath:   "/api/products",
			wantBody:   `{"name":"Acme Pro","variants":[{"name":"Standard","prices":[{"category":"subscription","scheme":"fixed","currency":"USD","unit_price":999,"billing_interval":"month","billing_interval_qty":1}]}]}`,
			// prod_1 contains no digit "3", so "3" in wantOut uniquely identifies the VARIANTS count column.
			respBody: `{"id":"prod_1","name":"Acme Pro","status":"active","variants":[{"id":"var_1","name":"Standard","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"},{"id":"var_2","name":"Plus","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"},{"id":"var_4","name":"Pro","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:  []string{"prod_1", "Acme Pro", "active", "3"},
			wantCode: 0,
		},
		// create missing --name → exit 2 (server would also reject, but CLI validates first)
		{
			name:     "products create missing name",
			args:     []string{"products", "create"},
			wantErr:  []string{"--name is required"},
			wantCode: 2,
		},

		// ---- products list ----
		// The CLI no longer forwards --status as a query param to the generated
		// client (ListProductsParams carries only pagination), so we assert
		// pagination behaviour and rendering rather than a status query param.

		{
			name:       "products list archived (renders archived row)",
			args:       []string{"products", "list", "--status", "archived"},
			wantMethod: "GET",
			wantPath:   "/api/products",
			respBody:   `{"data":[{"id":"prod_2","name":"Old Product","status":"archived","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"prod_2", "Old Product", "archived", "total 1"},
			wantCode:   0,
		},
		{
			name:       "products list",
			args:       []string{"products", "list"},
			wantMethod: "GET",
			wantPath:   "/api/products",
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
		// ogen serializes UpdateProductRequest and omits unset optional fields
		// (metadata), so the body carries only name and description.
		{
			name:       "products update happy path",
			args:       []string{"products", "update", "prod_1", "--name", "Acme Pro v2", "--description", "Updated desc"},
			wantMethod: "PATCH",
			wantPath:   "/api/products/prod_1",
			wantBody:   `{"name":"Acme Pro v2","description":"Updated desc"}`,
			respBody:   `{"id":"prod_1","name":"Acme Pro v2","status":"active","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "Acme Pro v2"},
			wantCode:   0,
		},

		// ---- products delete ----
		// NOTE: changed from the old test's 204 to a 200 with an empty-JSON body.
		// The generated DeleteProduct decoder maps the success (EmptyResponse,
		// a jx.Raw) to status 200 and decodes a JSON value from the body; a 204
		// would fall through to the bodyless DeleteProductDef default variant.
		{
			name:       "products delete",
			args:       []string{"products", "delete", "prod_1"},
			wantMethod: "DELETE",
			wantPath:   "/api/products/prod_1",
			respStatus: 200,
			respBody:   `{}`,
			wantOut:    []string{"prod_1 deleted"},
			wantCode:   0,
		},
		{
			name:       "products delete -o json",
			args:       []string{"products", "delete", "prod_1", "-o", "json"},
			wantMethod: "DELETE",
			wantPath:   "/api/products/prod_1",
			respStatus: 200,
			respBody:   `{}`,
			wantOut:    []string{},
			wantCode:   0,
		},

		// ---- products archive / unarchive ----

		{
			name:       "products archive",
			args:       []string{"products", "archive", "prod_1"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/archive",
			wantNoBody: true,
			respBody:   `{"id":"prod_1","name":"Acme Pro","status":"archived","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "archived"},
			wantCode:   0,
		},
		{
			name:       "products unarchive",
			args:       []string{"products", "unarchive", "prod_1"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/unarchive",
			wantNoBody: true,
			respBody:   `{"id":"prod_1","name":"Acme Pro","status":"active","variants":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"prod_1", "active"},
			wantCode:   0,
		},

		// ---- products variants list ----
		// Server returns the {data, meta} envelope for ListProductVariants.

		{
			name:       "products variants list",
			args:       []string{"products", "variants", "list", "prod_1"},
			wantMethod: "GET",
			wantPath:   "/api/products/prod_1/variants",
			// var_b has 3 prices; "var_b" contains no digit, so "3" uniquely identifies the PRICES count column.
			respBody: `{"data":[{"id":"var_b","name":"Standard","prices":[{"id":"pri_a"},{"id":"pri_b"},{"id":"pri_c"}],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:  []string{"var_b", "Standard", "3"},
			wantCode: 0,
		},

		// ---- products variants add ----
		// ogen omits unset optional fields (description, metadata).
		{
			name: "products variants add",
			args: []string{"products", "variants", "add", "prod_1",
				"--name", "Premium"},
			wantMethod: "POST",
			wantPath:   "/api/products/prod_1/variants",
			wantBody:   `{"name":"Premium"}`,
			respBody:   `{"id":"var_2","name":"Premium","prices":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`,
			wantOut:    []string{"var_2", "Premium"},
			wantCode:   0,
		},
	})
}
