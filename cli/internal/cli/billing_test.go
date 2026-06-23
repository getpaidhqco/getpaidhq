package cli_test

import (
	"net/url"
	"testing"
)

func TestBillingCmd(t *testing.T) {
	// Realistic apigen.InvoiceResponse fixture (snake_case json tags). All
	// fields are optional in the generated schema, but "status" must be a valid
	// InvoiceResponseStatus enum value for the ogen decoder.
	const invResp = `{"id":"inv_1","subscription_id":"sub_1","customer_id":"cus_1","order_id":"ord_1","status":"paid","currency":"NGN","subtotal":5000,"total":5000,"cycle":1,"period_start":"2026-06-12T09:00:00Z","period_end":"2026-07-12T09:00:00Z","line_items":[],"metadata":{},"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`

	// Realistic apigen.PaymentResponse fixture (snake_case json tags).
	const payResp = `{"id":"pay_1","psp_id":"paystack","reference":"REF001","order_id":"ord_1","subscription_id":"sub_1","invoice_id":"inv_1","status":"successful","currency":"NGN","amount":5000,"psp_fee":50,"platform_fee":0,"net_amount":4950,"metadata":{},"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`

	// Realistic apigen.PaymentMethodResponse fixture.
	const pmResp = `{"id":"pm_1","status":"active","psp":"paystack","name":"My Card","customer_id":"cus_1","billing_address":{},"type":"card","details":null,"metadata":null,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}`

	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// invoices list — {data,meta} envelope
		// ----------------------------------------------------------------
		{
			name:       "invoices list default pagination",
			args:       []string{"invoices", "list"},
			wantMethod: "GET",
			wantPath:   "/api/invoices",
			wantQuery: url.Values{
				"page":       []string{"0"},
				"limit":      []string{"10"},
				"sort_by":    []string{"created_at"},
				"sort_order": []string{"desc"},
			},
			respBody: `{"data":[` + invResp + `],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:  []string{"inv_1", "paid", "NGN", "5000", "total 1 · page 0 · limit 10"},
			wantCode: 0,
		},
		{
			name:       "invoices list -o json",
			args:       []string{"invoices", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/invoices",
			respBody:   `{"data":[` + invResp + `],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{`"total": 1`},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// invoices get — single InvoiceResponse
		// ----------------------------------------------------------------
		{
			name:       "invoices get by id",
			args:       []string{"invoices", "get", "inv_1"},
			wantMethod: "GET",
			wantPath:   "/api/invoices/inv_1",
			respBody:   invResp,
			wantOut:    []string{"inv_1", "paid", "NGN", "5000"},
			wantCode:   0,
		},
		{
			name:     "invoices get no args",
			args:     []string{"invoices", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		// ----------------------------------------------------------------
		// payments list — {data,meta} envelope
		// ----------------------------------------------------------------
		{
			name:       "payments list default pagination",
			args:       []string{"payments", "list"},
			wantMethod: "GET",
			wantPath:   "/api/payments",
			wantQuery: url.Values{
				"page":       []string{"0"},
				"limit":      []string{"10"},
				"sort_by":    []string{"created_at"},
				"sort_order": []string{"desc"},
			},
			respBody: `{"data":[` + payResp + `],"meta":{"total":2,"page":0,"limit":10}}`,
			wantOut:  []string{"pay_1", "successful", "NGN", "5000", "REF001", "total 2 · page 0 · limit 10"},
			wantCode: 0,
		},
		{
			name:       "payments list -o json",
			args:       []string{"payments", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/payments",
			respBody:   `{"data":[` + payResp + `],"meta":{"total":2,"page":0,"limit":10}}`,
			wantOut:    []string{`"total": 2`},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// payments get — single PaymentResponse
		// ----------------------------------------------------------------
		{
			name:       "payments get by id",
			args:       []string{"payments", "get", "pay_1"},
			wantMethod: "GET",
			wantPath:   "/api/payments/pay_1",
			respBody:   payResp,
			wantOut:    []string{"pay_1", "successful", "NGN", "5000", "REF001"},
			wantCode:   0,
		},
		{
			name:     "payments get no args",
			args:     []string{"payments", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		// ----------------------------------------------------------------
		// payment-methods get — JSON passthrough
		// ----------------------------------------------------------------
		{
			name:       "payment-methods get by id",
			args:       []string{"payment-methods", "get", "pm_1"},
			wantMethod: "GET",
			wantPath:   "/api/payment-methods/pm_1",
			respBody:   pmResp,
			wantOut:    []string{`"id": "pm_1"`, `"psp": "paystack"`, `"type": "card"`},
			wantCode:   0,
		},
		{
			name:     "payment-methods get no args",
			args:     []string{"payment-methods", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
	})
}
