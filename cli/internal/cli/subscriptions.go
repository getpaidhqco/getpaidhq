package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

var subHeaders = []string{"ID", "STATUS", "CURRENCY", "INTERVAL", "RENEWS", "CREATED"}

func subRow(s apigen.SubscriptionResponse) []string {
	return []string{
		s.ID.Or(""),
		s.Status.Or(""),
		s.Currency.Or(""),
		fmt.Sprintf("%d %s", s.BillingIntervalQty.Or(0), s.BillingInterval.Or("")),
		output.Time(s.RenewsAt.Or(time.Time{})),
		output.Time(s.CreatedAt.Or(time.Time{})),
	}
}

func newSubscriptionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscriptions",
		Short: "Manage subscriptions",
		Long:  "List, get, update, pause, resume, cancel, and inspect subscriptions.",
	}
	cmd.AddCommand(
		newSubscriptionsListCmd(app),
		newSubscriptionsGetCmd(app),
		newSubscriptionsUpdateCmd(app),
		newSubscriptionsPauseCmd(app),
		newSubscriptionsResumeCmd(app),
		newSubscriptionsCancelCmd(app),
		newSubscriptionsBillingAnchorCmd(app),
		newSubscriptionsPaymentsCmd(app),
		newSubscriptionsInvoicesCmd(app),
		newSubscriptionsUsageCmd(app),
	)
	return cmd
}

func newSubscriptionsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List subscriptions",
		Long:    "List subscriptions with optional pagination.",
		Example: "  gphq subscriptions list\n  gphq subscriptions list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListSubscriptions(cmd.Context(), apigen.ListSubscriptionsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, subHeaders, subRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/subscriptions")
}

func newSubscriptionsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <subscriptionId>",
		Short:   "Get a subscription",
		Long:    "Fetch a single subscription by ID.",
		Example: "  gphq subscriptions get sub_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetSubscription(cmd.Context(), apigen.GetSubscriptionParams{ID: args[0]})
			sub, err := expectOK[*apigen.SubscriptionResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *sub, subHeaders, subRow)
		},
	}
	return annotate(cmd, "GET", "/api/subscriptions/{id}")
}

func newSubscriptionsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <subscriptionId>",
		Short:   "Update a subscription",
		Long:    "Update subscription status, default payment method, or metadata. Unset flags are sent as zero values.",
		Example: "  gphq subscriptions update sub_1 --status paused\n  gphq subscriptions update sub_1 --metadata key=value",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.UpdateSubscriptionRequest) error {
				if s, _ := cmd.Flags().GetString("status"); s != "" {
					in.Status = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("default-payment-method"); s != "" {
					in.DefaultPaymentMethod = apigen.NewOptString(s)
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptUpdateSubscriptionRequestMetadata(apigen.UpdateSubscriptionRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateSubscription(cmd.Context(), body, apigen.UpdateSubscriptionParams{ID: args[0]})
			sub, err := expectOK[*apigen.SubscriptionResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *sub, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("status", "", "subscription status")
	f.String("default-payment-method", "", "default payment method ID")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
	return annotate(cmd, "PATCH", "/api/subscriptions/{id}")
}

func newSubscriptionsPauseCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pause <subscriptionId>",
		Short:   "Pause a subscription",
		Long:    "Pause an active subscription, optionally providing a reason.",
		Example: "  gphq subscriptions pause sub_1 --reason \"customer request\"",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.PauseSubscriptionRequest) error {
				if s, _ := cmd.Flags().GetString("reason"); s != "" {
					in.Reason = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.PauseSubscription(cmd.Context(), body, apigen.PauseSubscriptionParams{ID: args[0]})
			sub, err := expectOK[*apigen.SubscriptionResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *sub, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("reason", "", "reason for pausing")
	addDataFlag(cmd)
	return annotate(cmd, "PUT", "/api/subscriptions/{id}/pause")
}

func newSubscriptionsResumeCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resume <subscriptionId>",
		Short:   "Resume a subscription",
		Long:    "Resume a paused subscription. Use --behavior to control billing period behavior.",
		Example: "  gphq subscriptions resume sub_1 --behavior start_new_billing_period",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.ResumeSubscriptionRequest) error {
				if s, _ := cmd.Flags().GetString("behavior"); s != "" {
					in.ResumeBehavior = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.ResumeSubscription(cmd.Context(), body, apigen.ResumeSubscriptionParams{ID: args[0]})
			sub, err := expectOK[*apigen.SubscriptionResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *sub, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("behavior", "", "resume behavior: continue_existing_billing_period or start_new_billing_period")
	addDataFlag(cmd)
	return annotate(cmd, "PUT", "/api/subscriptions/{id}/resume")
}

func newSubscriptionsCancelCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cancel <subscriptionId>",
		Short:   "Cancel a subscription",
		Long:    "Cancel an active subscription, optionally providing a reason.",
		Example: "  gphq subscriptions cancel sub_1 --reason \"non-payment\"",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.CancelSubscriptionRequest) error {
				if s, _ := cmd.Flags().GetString("reason"); s != "" {
					in.Reason = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CancelSubscription(cmd.Context(), body, apigen.CancelSubscriptionParams{ID: args[0]})
			sub, err := expectOK[*apigen.SubscriptionResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *sub, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("reason", "", "reason for cancellation")
	addDataFlag(cmd)
	return annotate(cmd, "PUT", "/api/subscriptions/{id}/cancel")
}

func newSubscriptionsBillingAnchorCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "billing-anchor <subscriptionId>",
		Short:   "Update subscription billing anchor",
		Long:    "Update the billing anchor day (1-31) for a subscription.",
		Example: "  gphq subscriptions billing-anchor sub_1 --anchor 15 --proration prorate",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.UpdateBillingAnchorRequest) error {
				anchor, _ := cmd.Flags().GetInt("anchor")
				proration, _ := cmd.Flags().GetString("proration")
				if anchor == 0 || proration == "" {
					return Usagef("--anchor and --proration are required (or use --data)")
				}
				in.BillingAnchor = anchor
				in.ProrationMode = apigen.UpdateBillingAnchorRequestProrationMode(proration)
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateSubscriptionBillingAnchor(cmd.Context(), body, apigen.UpdateSubscriptionBillingAnchorParams{ID: args[0]})
			pd, err := expectOK[*apigen.ProrationDetailsResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, pd)
		},
	}
	f := cmd.Flags()
	f.Int("anchor", 0, "billing anchor day 1-31 (required)")
	f.String("proration", "", "proration mode: none or prorate (required)")
	addDataFlag(cmd)
	return annotate(cmd, "PATCH", "/api/subscriptions/{id}/billing-anchor")
}

func newSubscriptionsPaymentsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "payments <subscriptionId>",
		Short:   "List payments for a subscription",
		Long:    "List all payments made against a subscription.",
		Example: "  gphq subscriptions payments sub_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.ListSubscriptionPayments(cmd.Context(), apigen.ListSubscriptionPaymentsParams{ID: args[0]})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, paymentHeaders, paymentRow)
		},
	}
	return annotate(cmd, "GET", "/api/subscriptions/{id}/payments")
}

func newSubscriptionsInvoicesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invoices <subscriptionId>",
		Short:   "List invoices for a subscription",
		Long:    "List all billing-cycle invoices for a subscription.",
		Example: "  gphq subscriptions invoices sub_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.ListSubscriptionInvoices(cmd.Context(), apigen.ListSubscriptionInvoicesParams{ID: args[0]})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, invoiceHeaders, invoiceRow)
		},
	}
	return annotate(cmd, "GET", "/api/subscriptions/{id}/invoices")
}

func newSubscriptionsUsageCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "usage <subscriptionId>",
		Short:   "Get current-period usage for a subscription",
		Long:    "Fetch metered usage for a subscription's current billing period.",
		Example: "  gphq subscriptions usage sub_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetSubscriptionUsage(cmd.Context(), apigen.GetSubscriptionUsageParams{ID: args[0]})
			usage, err := expectOK[*apigen.SubscriptionUsageResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, usage)
		},
	}
	return annotate(cmd, "GET", "/api/subscriptions/{id}/usage")
}
