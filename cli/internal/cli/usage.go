package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

// ---------------------------------------------------------------------------
// meters table
// ---------------------------------------------------------------------------

var meterHeaders = []string{"ID", "CODE", "NAME", "AGGREGATION", "CREATED"}

func meterRow(m apigen.MeterResponse) []string {
	return []string{
		m.ID.Or(""),
		m.Code.Or(""),
		m.Name.Or(""),
		m.Aggregation.Or(""),
		output.Time(m.CreatedAt.Or(time.Time{})),
	}
}

// ---------------------------------------------------------------------------
// meters parent
// ---------------------------------------------------------------------------

func newMetersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "meters",
		Short: "Manage billable meters",
		Long:  "Create, list, and get billable metrics (meters) for usage-based billing.",
	}
	cmd.AddCommand(
		newMetersCreateCmd(app),
		newMetersListCmd(app),
		newMetersGetCmd(app),
	)
	return cmd
}

func newMetersCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a meter",
		Long: `Create a new billable meter.

A meter defines what usage events to measure and how to aggregate them. The
aggregation type determines how events are combined into a billable quantity.

Aggregation types: count sum max latest weighted_sum unique_count

Meter filters (rate dimensions per price) can only be specified via --data.
Example --data payload:
  {
    "code": "api_calls",
    "name": "API Calls",
    "aggregation": "count",
    "group_by": ["region"],
    "filters": [{"field": "tier", "values": ["pro", "enterprise"]}]
  }`,
		Example: "  gphq meters create --code api_calls --name \"API Calls\" --aggregation count\n  gphq meters create --code bytes --name \"Bytes\" --aggregation sum --field bytes_used --carry-over\n  gphq meters create --data '{\"code\":\"api_calls\",\"name\":\"API Calls\",\"aggregation\":\"count\"}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.CreateMeterRequest) error {
				code, _ := cmd.Flags().GetString("code")
				name, _ := cmd.Flags().GetString("name")
				agg, _ := cmd.Flags().GetString("aggregation")
				if code == "" || name == "" || agg == "" {
					return Usagef("--code, --name and --aggregation are required (or use --data)")
				}
				in.Code = code
				in.Name = name
				in.Aggregation = apigen.CreateMeterRequestAggregation(agg)
				if s, _ := cmd.Flags().GetString("field"); s != "" {
					in.FieldName = apigen.NewOptString(s)
				}
				if carryOver, _ := cmd.Flags().GetBool("carry-over"); carryOver {
					in.CarryOver = apigen.NewOptBool(carryOver)
				}
				if s, _ := cmd.Flags().GetString("rounding-mode"); s != "" {
					in.RoundingMode = apigen.NewOptCreateMeterRequestRoundingMode(apigen.CreateMeterRequestRoundingMode(s))
				}
				if scale, _ := cmd.Flags().GetInt("rounding-scale"); scale != 0 {
					in.RoundingScale = apigen.NewOptInt(scale)
				}
				if groupBy, _ := cmd.Flags().GetStringArray("group-by"); len(groupBy) > 0 {
					in.GroupBy = groupBy
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptCreateMeterRequestMetadata(apigen.CreateMeterRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateMeter(cmd.Context(), body, apigen.CreateMeterParams{})
			meter, err := expectOK[*apigen.MeterResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *meter, meterHeaders, meterRow)
		},
	}
	f := cmd.Flags()
	f.String("code", "", "meter code — referenced by usage events (required)")
	f.String("name", "", "human-readable meter name (required)")
	f.String("aggregation", "", "aggregation type: count sum max latest weighted_sum unique_count (required)")
	f.String("field", "", "event metadata field to aggregate (field_name); required for sum/max/latest/weighted_sum")
	f.Bool("carry-over", false, "carry over unused quota to the next billing period")
	f.String("rounding-mode", "", "rounding mode: round ceil floor")
	f.Int("rounding-scale", 0, "rounding decimal scale (0–18)")
	f.StringArray("group-by", nil, "metadata keys to group usage by (repeatable; v1 honours one key)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline); required for filter definitions")
	return annotate(cmd, "POST", "/api/meters")
}

func newMetersListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List meters",
		Long:    "List all billable meters for the organization.",
		Example: "  gphq meters list\n  gphq meters list --page 1 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListMeters(cmd.Context(), apigen.ListMetersParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, meterHeaders, meterRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/meters")
}

func newMetersGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a meter",
		Long:    "Fetch a single billable meter by ID.",
		Example: "  gphq meters get met_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetMeter(cmd.Context(), apigen.GetMeterParams{ID: args[0]})
			meter, err := expectOK[*apigen.MeterResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *meter, meterHeaders, meterRow)
		},
	}
	return annotate(cmd, "GET", "/api/meters/{id}")
}

// ---------------------------------------------------------------------------
// usage parent
// ---------------------------------------------------------------------------

func newUsageCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Ingest usage events",
		Long:  "Ingest metered usage events for billing.",
	}
	cmd.AddCommand(
		newUsageIngestCmd(app),
	)
	return cmd
}

func newUsageIngestCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a usage event",
		Long: `Ingest a single usage event (wrapped as a one-element batch).

Exactly one of --customer or --external-customer should identify the customer.
The event timestamp defaults to the ingestion time when omitted; pass an RFC3339
value to set an explicit time (e.g. 2026-06-12T10:00:00Z).

To ingest multiple events in one request pass --data with a full
{"events":[...]} body.`,
		Example: "  gphq usage ingest --metric api_calls --customer cus_1\n  gphq usage ingest --metric bytes --customer cus_1 --metadata bytes=1024 --timestamp 2026-06-12T10:00:00Z\n  gphq usage ingest --data '{\"events\":[{\"metric_code\":\"api_calls\",\"customer_id\":\"cus_1\"}]}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.IngestEventsRequest) error {
				metric, _ := cmd.Flags().GetString("metric")
				if metric == "" {
					return Usagef("--metric is required (or use --data)")
				}
				event := apigen.IngestEventsRequestEventsItem{MetricCode: metric}
				if s, _ := cmd.Flags().GetString("customer"); s != "" {
					event.CustomerID = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("external-customer"); s != "" {
					event.ExternalCustomerID = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("subscription"); s != "" {
					event.SubscriptionID = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("external-id"); s != "" {
					event.ExternalID = apigen.NewOptString(s)
				}
				if tsStr, _ := cmd.Flags().GetString("timestamp"); tsStr != "" {
					ts, err := time.Parse(time.RFC3339, tsStr)
					if err != nil {
						return Usagef("--timestamp must be RFC3339, e.g. 2026-06-12T10:00:00Z: %v", err)
					}
					event.Timestamp = apigen.NewOptDateTime(ts)
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					event.Metadata = apigen.NewOptIngestEventsRequestEventsItemMetadata(apigen.IngestEventsRequestEventsItemMetadata(meta))
				}
				in.Events = []apigen.IngestEventsRequestEventsItem{event}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.IngestUsageEvents(cmd.Context(), body, apigen.IngestUsageEventsParams{})
			resp, err := expectOK[*apigen.IngestEventsResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, resp)
		},
	}
	f := cmd.Flags()
	f.String("metric", "", "meter code the event counts against (metric_code; required)")
	f.String("customer", "", "customer ID (customer_id)")
	f.String("external-customer", "", "external customer ID (external_customer_id)")
	f.String("subscription", "", "subscription ID (subscription_id)")
	f.String("external-id", "", "idempotency key for the event (external_id)")
	f.String("timestamp", "", "event time, RFC3339 (defaults to ingestion time)")
	f.StringArray("metadata", nil, "event metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body — full {\"events\":[...]} batch (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/usage/ingest")
}

// ---------------------------------------------------------------------------
// reminders parent
// ---------------------------------------------------------------------------

func newRemindersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reminders",
		Short: "Manage renewal reminder config",
		Long:  "Get and set the renewal reminder configuration for the organization.",
	}
	cmd.AddCommand(
		newRemindersGetCmd(app),
		newRemindersSetCmd(app),
	)
	return cmd
}

func newRemindersGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get reminder config",
		Long:    "Fetch the renewal reminder configuration for the organization.",
		Example: "  gphq reminders get",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := app.API.GetReminderConfig(cmd.Context(), apigen.GetReminderConfigParams{})
			cfg, err := expectOK[*apigen.ReminderConfigDTO](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, cfg)
		},
	}
	return annotate(cmd, "GET", "/api/billing/reminder-config")
}

func newRemindersSetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set reminder config",
		Long: `Set the renewal reminder configuration for the organization.

Offsets are Go duration strings relative to the renewal date (e.g. "168h" = 7
days before renewal, "24h" = 1 day before). Pass --offset multiple times to
configure several reminder points.`,
		Example: "  gphq reminders set --enabled --offset 168h --offset 24h\n  gphq reminders set --data '{\"enabled\":true,\"offsets\":[\"168h\",\"24h\"]}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.ReminderConfigDTO) error {
				enabled, _ := cmd.Flags().GetBool("enabled")
				offsets, _ := cmd.Flags().GetStringArray("offset")
				in.Enabled = apigen.NewOptBool(enabled)
				in.Offsets = offsets
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateReminderConfig(cmd.Context(), body, apigen.UpdateReminderConfigParams{})
			cfg, err := expectOK[*apigen.ReminderConfigDTO](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, cfg)
		},
	}
	f := cmd.Flags()
	f.Bool("enabled", false, "enable renewal reminders")
	f.StringArray("offset", nil, "reminder offset before renewal, e.g. 168h (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "PUT", "/api/billing/reminder-config")
}
