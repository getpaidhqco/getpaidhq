package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

// ---------------------------------------------------------------------------
// dunning list envelope — DunningList carries {data: raw, total: int}
// ---------------------------------------------------------------------------

func renderDunningList[T any](app *App, lr *apigen.DunningList, headers []string, row func(T) []string) error {
	if app.Output == "json" {
		b, err := json.Marshal(lr)
		if err != nil {
			return err
		}
		return output.JSON(app.Out, b)
	}
	var items []T
	if len(lr.Data) > 0 {
		if err := json.Unmarshal(lr.Data, &items); err != nil {
			return fmt.Errorf("decoding list data: %w", err)
		}
	}
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = row(it)
	}
	if err := output.Table(app.Out, headers, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintf(app.Out, "\ntotal %d\n", lr.Total.Or(0))
	return err
}

// ---------------------------------------------------------------------------
// campaign table
// ---------------------------------------------------------------------------

var campaignHeaders = []string{"ID", "SUBSCRIPTION", "STATUS", "FAILED", "ATTEMPTS", "NEXT ATTEMPT", "CREATED"}

func campaignRow(c apigen.DunningCampaignResponse) []string {
	return []string{
		c.ID.Or(""),
		c.SubscriptionID.Or(""),
		c.Status.Or(""),
		fmt.Sprintf("%d", c.FailedAmount.Or(0)),
		fmt.Sprintf("%d", c.TotalAttempts.Or(0)),
		output.Time(c.NextAttemptAt.Or(time.Time{})),
		output.Time(c.CreatedAt.Or(time.Time{})),
	}
}

// ---------------------------------------------------------------------------
// config table
// ---------------------------------------------------------------------------

var dunningConfigHeaders = []string{"ID", "NAME", "STATUS", "PRIORITY", "APPLIES TO", "CREATED"}

func dunningConfigRow(c apigen.DunningConfigurationResponse) []string {
	return []string{
		c.ID.Or(""),
		c.Name.Or(""),
		c.Status.Or(""),
		fmt.Sprintf("%d", c.Priority.Or(0)),
		c.AppliesTo.Or(""),
		output.Time(c.CreatedAt.Or(time.Time{})),
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListDunningCampaigns(cmd.Context(), apigen.ListDunningCampaignsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.DunningList](res, err)
			if err != nil {
				return err
			}
			return renderDunningList(app, lr, campaignHeaders, campaignRow)
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
			res, err := app.API.GetDunningCampaign(cmd.Context(), apigen.GetDunningCampaignParams{ID: args[0]})
			c, err := expectOK[*apigen.DunningCampaignResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *c, campaignHeaders, campaignRow)
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
			body, err := bindBody(cmd, func(in *apigen.UpdateDunningCampaignRequest) error {
				status, _ := cmd.Flags().GetString("status")
				if status == "" {
					return Usagef("--status is required (active, paused, or cancelled) — or use --data")
				}
				in.Status = apigen.UpdateDunningCampaignRequestStatus(status)
				if reason, _ := cmd.Flags().GetString("reason"); reason != "" {
					in.Reason = apigen.NewOptString(reason)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateDunningCampaign(cmd.Context(), body, apigen.UpdateDunningCampaignParams{ID: args[0]})
			c, err := expectOK[*apigen.DunningCampaignResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *c, campaignHeaders, campaignRow)
		},
	}
	f := cmd.Flags()
	f.String("status", "", "new campaign status: active, paused, or cancelled (required)")
	f.String("reason", "", "reason for the status change")
	addDataFlag(cmd)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListDunningCampaignAttempts(cmd.Context(), apigen.ListDunningCampaignAttemptsParams{
				ID:        args[0],
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.DunningList](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, lr)
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
			body, err := bindBody(cmd, func(in *apigen.TriggerManualAttemptRequest) error {
				if pmID, _ := cmd.Flags().GetString("payment-method"); pmID != "" {
					in.PaymentMethodID = apigen.NewOptString(pmID)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.TriggerDunningManualAttempt(cmd.Context(), body, apigen.TriggerDunningManualAttemptParams{ID: args[0]})
			att, err := expectOK[*apigen.DunningAttemptResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, att)
		},
	}
	f := cmd.Flags()
	f.String("payment-method", "", "payment method ID to charge (optional)")
	addDataFlag(cmd)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListDunningCampaignCommunications(cmd.Context(), apigen.ListDunningCampaignCommunicationsParams{
				ID:        args[0],
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.DunningList](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, lr)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListDunningConfigurations(cmd.Context(), apigen.ListDunningConfigurationsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.DunningList](res, err)
			if err != nil {
				return err
			}
			return renderDunningList(app, lr, dunningConfigHeaders, dunningConfigRow)
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
			res, err := app.API.GetDunningConfiguration(cmd.Context(), apigen.GetDunningConfigurationParams{ID: args[0]})
			c, err := expectOK[*apigen.DunningConfigurationResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *c, dunningConfigHeaders, dunningConfigRow)
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
			body, err := bindBody(cmd, func(in *apigen.CreateDunningConfigurationRequest) error {
				name, _ := cmd.Flags().GetString("name")
				if name == "" {
					return Usagef("--name is required (or use --data)")
				}
				appliesTo, _ := cmd.Flags().GetString("applies-to")
				if appliesTo == "" {
					return Usagef("--applies-to is required (or use --data)")
				}
				in.Name = name
				in.AppliesTo = appliesTo
				if description, _ := cmd.Flags().GetString("description"); description != "" {
					in.Description = apigen.NewOptString(description)
				}
				priority, _ := cmd.Flags().GetInt("priority")
				in.Priority = apigen.NewOptInt(priority)
				// Config left zero — server will validate and reject if incomplete.
				// Callers who need a full config must use --data.
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateDunningConfiguration(cmd.Context(), body, apigen.CreateDunningConfigurationParams{})
			c, err := expectOK[*apigen.DunningConfigurationResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *c, dunningConfigHeaders, dunningConfigRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "configuration name (required)")
	f.String("description", "", "optional description")
	f.Int("priority", 0, "priority (lower = higher precedence)")
	f.String("applies-to", "", "scope: all, product, customer, etc. (required)")
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
			body, err := bindBody(cmd, func(in *apigen.UpdateDunningConfigurationRequest) error {
				if name, _ := cmd.Flags().GetString("name"); name != "" {
					in.Name = apigen.NewOptString(name)
				}
				if description, _ := cmd.Flags().GetString("description"); description != "" {
					in.Description = apigen.NewOptString(description)
				}
				if priority, _ := cmd.Flags().GetInt("priority"); priority != 0 {
					in.Priority = apigen.NewOptInt(priority)
				}
				if status, _ := cmd.Flags().GetString("status"); status != "" {
					in.Status = apigen.NewOptString(status)
				}
				// Config, IsAbTest, AbTestPercentage: only settable via --data
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateDunningConfiguration(cmd.Context(), body, apigen.UpdateDunningConfigurationParams{ID: args[0]})
			c, err := expectOK[*apigen.DunningConfigurationResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *c, dunningConfigHeaders, dunningConfigRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "new configuration name")
	f.String("description", "", "updated description")
	f.Int("priority", 0, "updated priority")
	f.String("status", "", "new status, e.g. active, inactive")
	addDataFlag(cmd)
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
			res, err := app.API.VerifyPaymentToken(cmd.Context(),
				&apigen.VerifyPaymentTokenRequest{TokenID: args[0]},
				apigen.VerifyPaymentTokenParams{})
			tok, err := expectOK[*apigen.PaymentUpdateTokenResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, tok)
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
			res, err := app.API.ActivatePaymentToken(cmd.Context(),
				&apigen.ActivatePaymentTokenRequest{TokenID: args[0]},
				apigen.ActivatePaymentTokenParams{})
			tok, err := expectOK[*apigen.PaymentUpdateTokenResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, tok)
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
			body, err := bindBody(cmd, func(in *apigen.CreatePaymentTokenRequest) error {
				if maxUses, _ := cmd.Flags().GetInt("max-uses"); maxUses != 0 {
					in.MaxUses = apigen.NewOptInt(maxUses)
				}
				if expiryHours, _ := cmd.Flags().GetInt("expiry-hours"); expiryHours != 0 {
					in.ExpiryHours = apigen.NewOptInt(expiryHours)
				}
				if reason, _ := cmd.Flags().GetString("reason"); reason != "" {
					in.AdminReason = apigen.NewOptString(reason)
				}
				if notes, _ := cmd.Flags().GetString("notes"); notes != "" {
					in.AdminNotes = apigen.NewOptString(notes)
				}
				// AllowedActions only settable via --data
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreatePaymentToken(cmd.Context(), body, apigen.CreatePaymentTokenParams{ID: args[0]})
			tok, err := expectOK[*apigen.PaymentUpdateTokenResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, tok)
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
