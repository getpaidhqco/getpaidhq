package cli_test

import (
	"net/url"
	"testing"
)

func TestSubscriptionsCmd(t *testing.T) {
	// Shared subscription response fixture — ogen SubscriptionResponse uses
	// snake_case spec field names; timestamps are RFC3339.
	const subResp = `{"id":"sub_1","status":"active","currency":"NGN","billing_interval":"month","billing_interval_qty":1,"renews_at":"2026-07-12T09:00:00Z","created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z","order_id":"ord_1","start_date":"2026-06-12T09:00:00Z","billing_anchor":1,"cycles":0,"cycles_processed":0,"total_revenue":0,"retries":0}`

	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// list — ListResponse envelope
		// ----------------------------------------------------------------
		{
			name:       "list default pagination",
			args:       []string{"subscriptions", "list"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions",
			wantQuery: url.Values{
				"page":       []string{"0"},
				"limit":      []string{"10"},
				"sort_by":    []string{"created_at"},
				"sort_order": []string{"desc"},
			},
			respBody: `{"data":[` + subResp + `],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:  []string{"sub_1", "active", "NGN", "total 1 · page 0 · limit 10"},
			wantCode: 0,
		},
		{
			name:       "list -o json",
			args:       []string{"subscriptions", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions",
			respBody:   `{"data":[` + subResp + `],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{`"total": 1`},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// get
		// ----------------------------------------------------------------
		{
			name:       "get by id",
			args:       []string{"subscriptions", "get", "sub_1"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions/sub_1",
			respBody:   subResp,
			wantOut:    []string{"sub_1", "active", "NGN", "1 month"},
			wantCode:   0,
		},
		{
			name:     "get no args",
			args:     []string{"subscriptions", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
		// ----------------------------------------------------------------
		// update
		// ----------------------------------------------------------------
		{
			name: "update status and metadata",
			args: []string{
				"subscriptions", "update", "sub_1",
				"--status", "paused",
				"--metadata", "reason=test",
			},
			wantMethod: "PATCH",
			wantPath:   "/api/subscriptions/sub_1",
			// only status + metadata are set; id and default_payment_method are
			// unset optionals and omitted by ogen.
			wantBody: `{"status":"paused","metadata":{"reason":"test"}}`,
			respBody: subResp,
			wantOut:  []string{"sub_1", "active"},
			wantCode: 0,
		},
		{
			name: "update via --data",
			args: []string{"subscriptions", "update", "sub_1",
				"--data", `{"status":"active"}`},
			wantMethod: "PATCH",
			wantPath:   "/api/subscriptions/sub_1",
			wantBody:   `{"status":"active"}`,
			respBody:   subResp,
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// pause
		// ----------------------------------------------------------------
		{
			name:       "pause with reason",
			args:       []string{"subscriptions", "pause", "sub_1", "--reason", "customer request"},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/pause",
			wantBody:   `{"reason":"customer request"}`,
			respBody:   subResp,
			wantOut:    []string{"sub_1"},
			wantCode:   0,
		},
		{
			name:       "pause via --data",
			args:       []string{"subscriptions", "pause", "sub_1", "--data", `{"reason":"trial ended"}`},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/pause",
			wantBody:   `{"reason":"trial ended"}`,
			respBody:   subResp,
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// resume
		// ----------------------------------------------------------------
		{
			name:       "resume with behavior",
			args:       []string{"subscriptions", "resume", "sub_1", "--behavior", "start_new_billing_period"},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/resume",
			wantBody:   `{"resume_behavior":"start_new_billing_period"}`,
			respBody:   subResp,
			wantOut:    []string{"sub_1"},
			wantCode:   0,
		},
		{
			name:       "resume via --data",
			args:       []string{"subscriptions", "resume", "sub_1", "--data", `{"resume_behavior":"continue_existing_billing_period"}`},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/resume",
			wantBody:   `{"resume_behavior":"continue_existing_billing_period"}`,
			respBody:   subResp,
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// cancel
		// ----------------------------------------------------------------
		{
			name:       "cancel with reason",
			args:       []string{"subscriptions", "cancel", "sub_1", "--reason", "non-payment"},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/cancel",
			wantBody:   `{"reason":"non-payment"}`,
			respBody:   subResp,
			wantOut:    []string{"sub_1"},
			wantCode:   0,
		},
		{
			name:       "cancel via --data",
			args:       []string{"subscriptions", "cancel", "sub_1", "--data", `{"reason":"upgrade"}`},
			wantMethod: "PUT",
			wantPath:   "/api/subscriptions/sub_1/cancel",
			wantBody:   `{"reason":"upgrade"}`,
			respBody:   subResp,
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// billing-anchor — returns ProrationDetailsResponse
		// ----------------------------------------------------------------
		{
			name: "billing-anchor happy",
			args: []string{
				"subscriptions", "billing-anchor", "sub_1",
				"--anchor", "15",
				"--proration", "prorate",
			},
			wantMethod: "PATCH",
			wantPath:   "/api/subscriptions/sub_1/billing-anchor",
			wantBody:   `{"billing_anchor":15,"proration_mode":"prorate"}`,
			respBody:   `{"credit_amount":0,"days_credited":0,"current_period_start":"2026-06-12T09:00:00Z","current_period_end":"2026-07-12T09:00:00Z","old_billing_anchor":1,"new_billing_anchor":15,"new_period_start":"2026-06-15T09:00:00Z","new_period_end":"2026-07-15T09:00:00Z"}`,
			wantOut:    []string{"credit_amount"},
			wantCode:   0,
		},
		{
			// missing --anchor and --proration → exit 2
			name:     "billing-anchor missing flags",
			args:     []string{"subscriptions", "billing-anchor", "sub_1"},
			wantErr:  []string{"--anchor and --proration are required"},
			wantCode: 2,
		},
		{
			// only --anchor, still missing --proration → exit 2
			name:     "billing-anchor missing proration",
			args:     []string{"subscriptions", "billing-anchor", "sub_1", "--anchor", "10"},
			wantErr:  []string{"--anchor and --proration are required"},
			wantCode: 2,
		},
		{
			// --data bypasses required-flag check
			name:       "billing-anchor via --data",
			args:       []string{"subscriptions", "billing-anchor", "sub_1", "--data", `{"billing_anchor":20,"proration_mode":"none"}`},
			wantMethod: "PATCH",
			wantPath:   "/api/subscriptions/sub_1/billing-anchor",
			wantBody:   `{"billing_anchor":20,"proration_mode":"none"}`,
			respBody:   `{"credit_amount":0,"days_credited":0,"current_period_start":"2026-06-12T09:00:00Z","current_period_end":"2026-07-12T09:00:00Z","new_period_start":"2026-06-20T09:00:00Z","new_period_end":"2026-07-20T09:00:00Z"}`,
			wantOut:    []string{"credit_amount"},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// payments — ListResponse of PaymentResponse
		// ----------------------------------------------------------------
		{
			name:       "payments list",
			args:       []string{"subscriptions", "payments", "sub_1"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions/sub_1/payments",
			respBody:   `{"data":[{"id":"pay_1","status":"successful","currency":"NGN","amount":5000,"reference":"REF001","psp_id":"paystack","order_id":"ord_1","subscription_id":"sub_1","invoice_id":"inv_1","psp_fee":0,"platform_fee":0,"net_amount":5000,"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"pay_1", "successful", "NGN", "5000", "REF001", "total 1 · page 0 · limit 10"},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// invoices — ListResponse of InvoiceResponse
		// ----------------------------------------------------------------
		{
			name:       "invoices list",
			args:       []string{"subscriptions", "invoices", "sub_1"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions/sub_1/invoices",
			respBody:   `{"data":[{"id":"inv_1","subscription_id":"sub_1","customer_id":"cus_1","order_id":"ord_1","status":"paid","currency":"NGN","subtotal":5000,"total":5000,"cycle":1,"period_start":"2026-06-12T09:00:00Z","period_end":"2026-07-12T09:00:00Z","line_items":[],"created_at":"2026-06-12T09:00:00Z","updated_at":"2026-06-12T09:00:00Z"}],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"inv_1", "paid", "NGN", "5000", "total 1 · page 0 · limit 10"},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// usage — typed SubscriptionUsageResponse marshaled to JSON
		// ----------------------------------------------------------------
		{
			name:       "usage passthrough",
			args:       []string{"subscriptions", "usage", "sub_1"},
			wantMethod: "GET",
			wantPath:   "/api/subscriptions/sub_1/usage",
			respBody:   `{"subscription_id":"sub_1","current_period_start":"2026-06-12T09:00:00Z","current_period_end":"2026-07-12T09:00:00Z","meters":[]}`,
			wantOut:    []string{`"subscription_id": "sub_1"`},
			wantCode:   0,
		},
	})
}
