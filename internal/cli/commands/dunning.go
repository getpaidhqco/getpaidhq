package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
	"getpaidhq/internal/core/domain"
)

// ---------------------------------------------------------------------------
// dunningList envelope — {data: any, total: int}
// ---------------------------------------------------------------------------

type dunningEnvelope[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
}

func renderDunningList[T any](app *App, raw []byte, headers []string, row func(T) []string) error {
	if app.Output == "json" {
		return output.JSON(app.Out, raw)
	}
	var page dunningEnvelope[T]
	if err := json.Unmarshal(raw, &page); err != nil {
		return fmt.Errorf("decoding list response: %w", err)
	}
	rows := make([][]string, len(page.Data))
	for i, item := range page.Data {
		rows[i] = row(item)
	}
	if err := output.Table(app.Out, headers, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintf(app.Out, "\ntotal %d\n", page.Total)
	return err
}

// ---------------------------------------------------------------------------
// campaign table
// ---------------------------------------------------------------------------

var campaignHeaders = []string{"ID", "SUBSCRIPTION", "STATUS", "FAILED", "ATTEMPTS", "NEXT ATTEMPT", "CREATED"}

func campaignRow(c api.DunningCampaignResponse) []string {
	return []string{
		c.ID,
		c.SubscriptionID,
		c.Status,
		strconv.FormatInt(c.FailedAmount, 10),
		strconv.Itoa(c.TotalAttempts),
		output.Time(c.NextAttemptAt),
		output.Time(c.CreatedAt),
	}
}

// ---------------------------------------------------------------------------
// config table
// ---------------------------------------------------------------------------

var dunningConfigHeaders = []string{"ID", "NAME", "STATUS", "PRIORITY", "APPLIES TO", "CREATED"}

func dunningConfigRow(c api.DunningConfigurationResponse) []string {
	return []string{
		c.ID,
		c.Name,
		c.Status,
		strconv.Itoa(c.Priority),
		string(c.AppliesTo),
		output.Time(c.CreatedAt),
	}
}

// ---------------------------------------------------------------------------
// dunning parent
// ---------------------------------------------------------------------------

func newDunningCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dunning",
		Short: "Manage dunning campaigns and configurations",
		Long:  "Manage dunning campaigns (failed-payment recovery runners) and dunning configurations (retry policies).",
	}
	cmd.AddCommand(
		newDunningCampaignsCmd(app),
		newDunningConfigsCmd(app),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// dunning campaigns sub-parent
// ---------------------------------------------------------------------------

func newDunningCampaignsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaigns",
		Short: "Manage dunning campaigns",
		Long:  "List, inspect, update, and act on dunning campaigns.",
	}
	cmd.AddCommand(
		newCampaignsListCmd(app),
		newCampaignsGetCmd(app),
		newCampaignsUpdateCmd(app),
		newCampaignsAttemptsCmd(app),
		newCampaignsRetryCmd(app),
		newCampaignsCommunicationsCmd(app),
	)
	return cmd
}

func newCampaignsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List dunning campaigns",
		Long:    "List all dunning campaigns for the organization.",
		Example: "  gphq dunning campaigns list\n  gphq dunning campaigns list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/campaigns", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderDunningList(app, raw, campaignHeaders, campaignRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/dunning/campaigns")
}

func newCampaignsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a dunning campaign",
		Long:    "Fetch a single dunning campaign by ID.",
		Example: "  gphq dunning campaigns get dc_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/campaigns/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, campaignHeaders, campaignRow)
		},
	}
	return annotate(cmd, "GET", "/api/dunning/campaigns/{id}")
}

func newCampaignsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a dunning campaign",
		Long: `Update the status of a dunning campaign.

Status must be one of: active, paused, cancelled.
Pass --status to change the campaign state and optionally --reason to record why.
Use --data to send a raw JSON body instead of flags.`,
		Example: "  gphq dunning campaigns update dc_1 --status paused --reason \"investigating payment issue\"\n  gphq dunning campaigns update dc_1 --data '{\"status\":\"cancelled\",\"reason\":\"customer churned\"}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				status, _ := cmd.Flags().GetString("status")
				if status == "" {
					return nil, Usagef("--status is required (active, paused, or cancelled) — or use --data")
				}
				reason, _ := cmd.Flags().GetString("reason")
				return api.UpdateDunningCampaignRequest{
					Status: status,
					Reason: reason,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/dunning/campaigns/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, campaignHeaders, campaignRow)
		},
	}
	f := cmd.Flags()
	f.String("status", "", "new campaign status: active, paused, or cancelled (required)")
	f.String("reason", "", "reason for the status change")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "PATCH", "/api/dunning/campaigns/{id}")
}

func newCampaignsAttemptsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attempts <id>",
		Short:   "List attempts for a dunning campaign",
		Long:    "List all retry attempts for a dunning campaign.",
		Example: "  gphq dunning campaigns attempts dc_1\n  gphq dunning campaigns attempts dc_1 --limit 50",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/campaigns/"+args[0]+"/attempts", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/dunning/campaigns/{id}/attempts")
}

func newCampaignsRetryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <id>",
		Short: "Trigger a manual retry attempt",
		Long: `Trigger a manual dunning retry attempt for a campaign.

Optionally pass --payment-method to specify the payment method ID to charge.
Use --data to send a raw JSON body instead of flags.`,
		Example: "  gphq dunning campaigns retry dc_1\n  gphq dunning campaigns retry dc_1 --payment-method pm_abc\n  gphq dunning campaigns retry dc_1 --data '{\"payment_method_id\":\"pm_abc\"}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				pmID, _ := cmd.Flags().GetString("payment-method")
				return api.TriggerManualAttemptRequest{
					PaymentMethodID: pmID,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/dunning/campaigns/"+args[0]+"/attempts", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("payment-method", "", "payment method ID to charge (optional)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/dunning/campaigns/{id}/attempts")
}

func newCampaignsCommunicationsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "communications <id>",
		Short:   "List communications for a dunning campaign",
		Long:    "List all customer communications (emails, SMS, etc.) sent for a dunning campaign.",
		Example: "  gphq dunning campaigns communications dc_1\n  gphq dunning campaigns communications dc_1 --limit 50",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/campaigns/"+args[0]+"/communications", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/dunning/campaigns/{id}/communications")
}

// ---------------------------------------------------------------------------
// dunning configs sub-parent
// ---------------------------------------------------------------------------

func newDunningConfigsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configs",
		Short: "Manage dunning configurations",
		Long:  "List, inspect, create, and update dunning retry configurations.",
	}
	cmd.AddCommand(
		newConfigsListCmd(app),
		newConfigsGetCmd(app),
		newConfigsCreateCmd(app),
		newConfigsUpdateCmd(app),
	)
	return cmd
}

func newConfigsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List dunning configurations",
		Long:    "List all dunning retry configurations for the organization.",
		Example: "  gphq dunning configs list\n  gphq dunning configs list --page 0 --limit 20",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/configurations", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderDunningList(app, raw, dunningConfigHeaders, dunningConfigRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/dunning/configurations")
}

func newConfigsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a dunning configuration",
		Long:    "Fetch a single dunning configuration by ID.",
		Example: "  gphq dunning configs get dcfg_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/dunning/configurations/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, dunningConfigHeaders, dunningConfigRow)
		},
	}
	return annotate(cmd, "GET", "/api/dunning/configurations/{id}")
}

func newConfigsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a dunning configuration",
		Long: `Create a new dunning retry configuration.

The API requires a nested "config" object (retry schedule, escalation policy, etc.)
that cannot be expressed with flags alone — flag-only creates will be rejected by the
server unless you also supply the full config via --data. Use --data for complete
configurations; flags are provided as a convenience for simple cases.

Example --data payload:
  {
    "name": "Standard retry",
    "applies_to": "all",
    "config": {
      "immediate_attempts": 1,
      "progressive_attempts": [{"delay_hours": 24}, {"delay_hours": 72}],
      "escalation_policy": "cancel"
    }
  }`,
		Example: "  gphq dunning configs create --data @config.json\n  gphq dunning configs create --name \"Standard\" --applies-to all --data '{\"config\":{\"immediate_attempts\":1,\"escalation_policy\":\"cancel\"}}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				if name == "" {
					return nil, Usagef("--name is required (or use --data)")
				}
				appliesTo, _ := cmd.Flags().GetString("applies-to")
				if appliesTo == "" {
					return nil, Usagef("--applies-to is required (or use --data)")
				}
				description, _ := cmd.Flags().GetString("description")
				priority, _ := cmd.Flags().GetInt("priority")
				// Config left zero — server will validate and reject if incomplete.
				// Callers who need a full config must use --data.
				return api.CreateDunningConfigurationRequest{
					Name:        name,
					Description: description,
					Priority:    priority,
					AppliesTo:   domain.DunningConfigScope(appliesTo),
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/dunning/configurations", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, dunningConfigHeaders, dunningConfigRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "configuration name (required unless --data is used)")
	f.String("description", "", "optional description")
	f.Int("priority", 0, "priority (lower = higher precedence)")
	f.String("applies-to", "", "scope: all, product, customer, etc. (required unless --data is used)")
	f.String("data", "", "raw JSON body (@file, -, or inline); the API requires a nested config object so complex configurations must use this flag")
	return annotate(cmd, "POST", "/api/dunning/configurations")
}

func newConfigsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a dunning configuration",
		Long:    "Update fields on a dunning configuration. Unset flags are sent as empty values, which the server ignores. Note: priority 0 cannot be set via flags (the server treats 0 as unset); use --data.",
		Example: "  gphq dunning configs update dcfg_1 --name \"Updated name\" --status active\n  gphq dunning configs update dcfg_1 --data '{\"name\":\"New name\",\"status\":\"active\"}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				description, _ := cmd.Flags().GetString("description")
				priority, _ := cmd.Flags().GetInt("priority")
				status, _ := cmd.Flags().GetString("status")
				return api.UpdateDunningConfigurationRequest{
					Name:        name,
					Description: description,
					Priority:    priority,
					Status:      domain.ConfigStatus(status),
					// Config, IsAbTest, AbTestPercentage: pointer fields — left nil
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/dunning/configurations/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, dunningConfigHeaders, dunningConfigRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "new configuration name")
	f.String("description", "", "updated description")
	f.Int("priority", 0, "updated priority")
	f.String("status", "", "new status, e.g. active, inactive")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "PATCH", "/api/dunning/configurations/{id}")
}

// ---------------------------------------------------------------------------
// payment-tokens
// ---------------------------------------------------------------------------

func newPaymentTokensCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-tokens",
		Short: "Manage payment update tokens",
		Long:  "Verify, activate, and create payment update tokens used in dunning recovery flows.",
	}
	cmd.AddCommand(
		newPaymentTokensVerifyCmd(app),
		newPaymentTokensActivateCmd(app),
		newPaymentTokensCreateCmd(app),
	)
	return cmd
}

func newPaymentTokensVerifyCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "verify <tokenId>",
		Short:   "Verify a payment update token",
		Long:    "Verify that a payment update token is valid and retrieve its metadata.",
		Example: "  gphq payment-tokens verify tok_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := api.VerifyPaymentTokenRequest{TokenID: args[0]}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/payment-tokens/verify", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "POST", "/api/payment-tokens/verify")
}

func newPaymentTokensActivateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "activate <tokenId>",
		Short:   "Activate a payment update token",
		Long:    "Activate a payment update token (marks it as used for a payment method update).",
		Example: "  gphq payment-tokens activate tok_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := api.ActivatePaymentTokenRequest{TokenID: args[0]}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/payment-tokens/activate", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "POST", "/api/payment-tokens/activate")
}

func newPaymentTokensCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <subscriptionId>",
		Short: "Create a payment update token (admin)",
		Long: `Admin: create a payment update token for a subscription.

The token can be sent to customers as part of a dunning recovery flow to allow
them to update their payment method without logging in.`,
		Example: "  gphq payment-tokens create sub_1 --max-uses 3 --expiry-hours 48 --reason \"proactive retry\"\n  gphq payment-tokens create sub_1 --data '{\"max_uses\":1,\"expiry_hours\":24}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				maxUses, _ := cmd.Flags().GetInt("max-uses")
				expiryHours, _ := cmd.Flags().GetInt("expiry-hours")
				reason, _ := cmd.Flags().GetString("reason")
				notes, _ := cmd.Flags().GetString("notes")
				return api.CreatePaymentTokenRequest{
					MaxUses:     maxUses,
					ExpiryHours: expiryHours,
					AdminReason: reason,
					AdminNotes:  notes,
					// AllowedActions only settable via --data
				}, nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/admin/subscriptions/%s/payment-tokens", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.Int("max-uses", 0, "maximum number of times the token can be used (0 = unlimited)")
	f.Int("expiry-hours", 0, "hours until token expiry (0 = default server expiry)")
	f.String("reason", "", "admin reason for creating the token (admin_reason)")
	f.String("notes", "", "admin notes (admin_notes)")
	f.String("data", "", "raw JSON body (@file, -, or inline); use this to set allowed_actions")
	return annotate(cmd, "POST", "/api/admin/subscriptions/{id}/payment-tokens")
}
