package cli_test

import (
	"testing"
)

func TestMetersCmd(t *testing.T) {
	meterResp := `{
		"id": "met_1",
		"code": "api_calls",
		"name": "API Calls",
		"aggregation": "count",
		"field_name": "",
		"carry_over": false,
		"rounding_mode": "",
		"rounding_scale": 0,
		"filters": [],
		"group_by": ["region"],
		"metadata": {"env": "prod"},
		"created_at": "2026-06-12T10:00:00Z",
		"updated_at": "2026-06-12T10:00:00Z"
	}`

	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// meters create — full body via --data. bindBody decodes the raw JSON
		// into CreateMeterRequest then re-encodes it via the ogen encoder, so
		// the wire body is the round-tripped form (same fields, JSON-equal).
		// ----------------------------------------------------------------
		{
			name:       "create full body via --data",
			args:       []string{"meters", "create", "--data", `{"code":"api_calls","name":"API Calls","aggregation":"count","group_by":["region"],"metadata":{"env":"prod"}}`},
			wantMethod: "POST",
			wantPath:   "/api/meters",
			wantBody:   `{"code":"api_calls","name":"API Calls","aggregation":"count","group_by":["region"],"metadata":{"env":"prod"}}`,
			respBody:   meterResp,
			wantOut:    []string{"met_1", "api_calls", "API Calls", "count"},
			wantCode:   0,
		},
		// create via flags — ogen omits unset optionals (filters, group_by,
		// rounding_*, metadata) entirely; required code/name/aggregation always
		// emit, and carry_over/field_name emit because they were set.
		{
			name:       "create with flags",
			args:       []string{"meters", "create", "--code", "bytes_used", "--name", "Bytes Used", "--aggregation", "sum", "--field", "bytes", "--carry-over"},
			wantMethod: "POST",
			wantPath:   "/api/meters",
			wantBody:   `{"code":"bytes_used","name":"Bytes Used","aggregation":"sum","field_name":"bytes","carry_over":true}`,
			respBody:   `{"id":"met_2","code":"bytes_used","name":"Bytes Used","aggregation":"sum","field_name":"bytes","carry_over":true,"rounding_mode":"","rounding_scale":0,"filters":[],"group_by":[],"metadata":{},"created_at":"2026-06-12T10:00:00Z","updated_at":"2026-06-12T10:00:00Z"}`,
			wantOut:    []string{"met_2", "bytes_used", "Bytes Used", "sum"},
			wantCode:   0,
		},
		// create -o json
		{
			name:       "create json output",
			args:       []string{"meters", "create", "-o", "json", "--data", `{"code":"c","name":"n","aggregation":"count"}`},
			wantMethod: "POST",
			wantPath:   "/api/meters",
			respBody:   meterResp,
			wantOut:    []string{`"id": "met_1"`, `"code": "api_calls"`},
			wantCode:   0,
		},
		// create missing required flags
		{
			name:     "create missing aggregation",
			args:     []string{"meters", "create", "--code", "api_calls", "--name", "API Calls"},
			wantErr:  []string{"--code, --name and --aggregation are required"},
			wantCode: 2,
		},
		{
			name:     "create missing code",
			args:     []string{"meters", "create", "--name", "API Calls", "--aggregation", "count"},
			wantErr:  []string{"--code, --name and --aggregation are required"},
			wantCode: 2,
		},
		// ----------------------------------------------------------------
		// meters list
		// ----------------------------------------------------------------
		{
			name:       "list",
			args:       []string{"meters", "list"},
			wantMethod: "GET",
			wantPath:   "/api/meters",
			respBody:   `{"data":[` + meterResp + `],"meta":{"total":1,"page":0,"limit":10}}`,
			wantOut:    []string{"met_1", "api_calls", "API Calls", "count", "total 1 · page 0 · limit 10"},
			wantCode:   0,
		},
		{
			name:       "list json output",
			args:       []string{"meters", "list", "-o", "json"},
			wantMethod: "GET",
			wantPath:   "/api/meters",
			respBody:   `{"data":[],"meta":{"total":0,"page":0,"limit":10}}`,
			wantOut:    []string{`"total": 0`},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// meters get
		// ----------------------------------------------------------------
		{
			name:       "get",
			args:       []string{"meters", "get", "met_1"},
			wantMethod: "GET",
			wantPath:   "/api/meters/met_1",
			respBody:   meterResp,
			wantOut:    []string{"met_1", "api_calls", "API Calls", "count"},
			wantCode:   0,
		},
		{
			name:     "get no args",
			args:     []string{"meters", "get"},
			wantErr:  []string{"expects 1 argument"},
			wantCode: 2,
		},
	})
}

func TestUsageCmd(t *testing.T) {
	ingestResp := `{"results":[{"index":0,"id":"evt_1","status":"recorded"}]}`

	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// usage ingest — happy path. ogen omits unset optional event fields,
		// so only the supplied fields appear on the wire.
		// ----------------------------------------------------------------
		{
			name: "ingest with flags",
			args: []string{
				"usage", "ingest",
				"--metric", "api_calls",
				"--customer", "cus_1",
				"--metadata", "region=eu",
				"--timestamp", "2026-06-12T10:00:00Z",
			},
			wantMethod: "POST",
			wantPath:   "/api/usage/ingest",
			wantBody:   `{"events":[{"customer_id":"cus_1","metric_code":"api_calls","timestamp":"2026-06-12T10:00:00Z","metadata":{"region":"eu"}}]}`,
			respBody:   ingestResp,
			wantOut:    []string{`"status": "recorded"`},
			wantCode:   0,
		},
		// ingest with external_customer_id — timestamp omitted (defaults
		// server-side), metadata omitted.
		{
			name: "ingest external customer",
			args: []string{
				"usage", "ingest",
				"--metric", "bytes_used",
				"--external-customer", "ext_cus_42",
				"--external-id", "dedup_key_1",
			},
			wantMethod: "POST",
			wantPath:   "/api/usage/ingest",
			wantBody:   `{"events":[{"external_customer_id":"ext_cus_42","metric_code":"bytes_used","external_id":"dedup_key_1"}]}`,
			respBody:   ingestResp,
			wantOut:    []string{`"results"`},
			wantCode:   0,
		},
		// ingest missing metric
		{
			name:     "ingest missing metric",
			args:     []string{"usage", "ingest", "--customer", "cus_1"},
			wantErr:  []string{"--metric is required"},
			wantCode: 2,
		},
		// ingest bad timestamp
		{
			name:     "ingest bad timestamp",
			args:     []string{"usage", "ingest", "--metric", "api_calls", "--timestamp", "not-a-date"},
			wantErr:  []string{"--timestamp must be RFC3339"},
			wantCode: 2,
		},
		// ingest via --data (batch passthrough). bindBody round-trips the body
		// through IngestEventsRequest; only metric_code is required so the two
		// supplied events re-encode identically.
		{
			name:       "ingest batch via --data",
			args:       []string{"usage", "ingest", "--data", `-`},
			stdin:      `{"events":[{"metric_code":"api_calls","customer_id":"cus_1"},{"metric_code":"api_calls","customer_id":"cus_2"}]}`,
			wantMethod: "POST",
			wantPath:   "/api/usage/ingest",
			wantBody:   `{"events":[{"metric_code":"api_calls","customer_id":"cus_1"},{"metric_code":"api_calls","customer_id":"cus_2"}]}`,
			respBody:   `{"results":[{"index":0,"id":"evt_1","status":"recorded"},{"index":1,"id":"evt_2","status":"recorded"}]}`,
			wantOut:    []string{`"results"`},
			wantCode:   0,
		},
	})
}

func TestRemindersCmd(t *testing.T) {
	configResp := `{"enabled":true,"offsets":["168h0m0s","24h0m0s"]}`

	runCases(t, []cmdCase{
		// ----------------------------------------------------------------
		// reminders get
		// ----------------------------------------------------------------
		{
			name:       "get",
			args:       []string{"reminders", "get"},
			wantMethod: "GET",
			wantPath:   "/api/billing/reminder-config",
			respBody:   configResp,
			wantOut:    []string{`"enabled": true`, `"offsets"`},
			wantCode:   0,
		},
		// ----------------------------------------------------------------
		// reminders set — body: {"enabled":true,"offsets":["168h","24h"]}
		// ----------------------------------------------------------------
		{
			name:       "set with flags",
			args:       []string{"reminders", "set", "--enabled", "--offset", "168h", "--offset", "24h"},
			wantMethod: "PUT",
			wantPath:   "/api/billing/reminder-config",
			wantBody:   `{"enabled":true,"offsets":["168h","24h"]}`,
			respBody:   configResp,
			wantOut:    []string{`"enabled": true`},
			wantCode:   0,
		},
		// set via --data
		{
			name:       "set via --data",
			args:       []string{"reminders", "set", "--data", `{"enabled":true,"offsets":["168h","24h"]}`},
			wantMethod: "PUT",
			wantPath:   "/api/billing/reminder-config",
			wantBody:   `{"enabled":true,"offsets":["168h","24h"]}`,
			respBody:   configResp,
			wantOut:    []string{`"enabled"`},
			wantCode:   0,
		},
	})
}
