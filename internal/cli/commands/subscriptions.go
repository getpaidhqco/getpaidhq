package commands

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
	"getpaidhq/internal/core/domain"
)

// subHeaders and subRow are for the api.SubscriptionResponse type (tagged JSON).
// Do not confuse with subscriptionHeaders/subscriptionRow in orders.go which
// decode the untagged []domain.Subscription shape returned by orders/{id}/subscriptions.
var subHeaders = []string{"ID", "STATUS", "CURRENCY", "INTERVAL", "RENEWS", "CREATED"}

func subRow(s api.SubscriptionResponse) []string {
	return []string{
		s.Id,
		string(s.Status),
		s.Currency,
		fmt.Sprintf("%d %s", s.BillingIntervalQty, s.BillingInterval),
		output.Time(s.RenewsAt),
		output.Time(s.CreatedAt),
	}
}

// paymentHeaders and paymentRow are defined here (used by subscriptions payments);
// Task 8 (invoices/payments) will reuse these.
var paymentHeaders = []string{"ID", "STATUS", "CURRENCY", "AMOUNT", "REFERENCE", "CREATED"}

func paymentRow(p api.PaymentResponse) []string {
	return []string{
		p.Id,
		string(p.Status),
		p.Currency,
		strconv.FormatInt(p.Amount, 10),
		output.Str(p.Reference),
		output.Time(p.CreatedAt),
	}
}

// invoiceHeaders and invoiceRow are defined here (used by subscriptions invoices);
// Task 8 (invoices/payments) will reuse these.
var invoiceHeaders = []string{"ID", "STATUS", "CURRENCY", "TOTAL", "CYCLE", "PERIOD END", "CREATED"}

func invoiceRow(i api.InvoiceResponse) []string {
	return []string{
		i.Id,
		i.Status,
		i.Currency,
		strconv.FormatInt(i.Total, 10),
		strconv.Itoa(i.Cycle),
		output.Time(i.PeriodEnd),
		output.Time(i.CreatedAt),
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/subscriptions", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, subHeaders, subRow)
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/subscriptions/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, subHeaders, subRow)
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
			body, err := bodyOrData(cmd, func() (any, error) {
				status, _ := cmd.Flags().GetString("status")
				defaultPM, _ := cmd.Flags().GetString("default-payment-method")
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return domain.UpdateSubscriptionRequest{
					Status:               domain.SubscriptionStatus(status),
					DefaultPaymentMethod: defaultPM,
					Metadata:             meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/subscriptions/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("status", "", "subscription status")
	f.String("default-payment-method", "", "default payment method ID")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			body, err := bodyOrData(cmd, func() (any, error) {
				reason, _ := cmd.Flags().GetString("reason")
				return api.PauseSubscriptionRequest{Reason: reason}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, "/api/subscriptions/"+args[0]+"/pause", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("reason", "", "reason for pausing")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			body, err := bodyOrData(cmd, func() (any, error) {
				behavior, _ := cmd.Flags().GetString("behavior")
				return api.ResumeSubscriptionRequest{
					ResumeBehavior: domain.SubscriptionResumeBehavior(behavior),
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, "/api/subscriptions/"+args[0]+"/resume", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("behavior", "", "resume behavior: continue_existing_billing_period or start_new_billing_period")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			body, err := bodyOrData(cmd, func() (any, error) {
				reason, _ := cmd.Flags().GetString("reason")
				return api.PauseSubscriptionRequest{Reason: reason}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, "/api/subscriptions/"+args[0]+"/cancel", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, subHeaders, subRow)
		},
	}
	f := cmd.Flags()
	f.String("reason", "", "reason for cancellation")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			body, err := bodyOrData(cmd, func() (any, error) {
				anchor, _ := cmd.Flags().GetInt("anchor")
				proration, _ := cmd.Flags().GetString("proration")
				if anchor == 0 || proration == "" {
					return nil, Usagef("--anchor and --proration are required (or use --data)")
				}
				return api.UpdateBillingAnchorRequest{
					BillingAnchor: anchor,
					ProrationMode: domain.ProrationMode(proration),
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/subscriptions/"+args[0]+"/billing-anchor", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.Int("anchor", 0, "billing anchor day 1-31 (required)")
	f.String("proration", "", "proration mode: none or prorate (required)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			path := "/api/subscriptions/" + args[0] + "/payments"
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, paymentHeaders, paymentRow)
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
			path := "/api/subscriptions/" + args[0] + "/invoices"
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, invoiceHeaders, invoiceRow)
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
			path := "/api/subscriptions/" + args[0] + "/usage"
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "GET", "/api/subscriptions/{id}/usage")
}
