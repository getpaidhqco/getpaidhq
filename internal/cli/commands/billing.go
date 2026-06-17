package commands

import (
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
)

// paymentHeaders and paymentRow are the canonical table shape for
// api.PaymentResponse, used by both the top-level payments commands and
// subscriptions payments sub-command.
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

// invoiceHeaders and invoiceRow are the canonical table shape for
// api.InvoiceResponse, used by both the top-level invoices commands and
// subscriptions invoices sub-command.
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

// ---------------------------------------------------------------------------
// invoices
// ---------------------------------------------------------------------------

func newInvoicesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoices",
		Short: "Inspect invoices",
		Long:  "List and get billing-cycle invoices.",
	}
	cmd.AddCommand(
		newInvoicesListCmd(app),
		newInvoicesGetCmd(app),
	)
	return cmd
}

func newInvoicesListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List invoices",
		Long:    "List all invoices for the organization with optional pagination.",
		Example: "  gphq invoices list\n  gphq invoices list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/invoices", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, invoiceHeaders, invoiceRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/invoices")
}

func newInvoicesGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <invoiceId>",
		Short:   "Get an invoice",
		Long:    "Fetch a single invoice by ID.",
		Example: "  gphq invoices get inv_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/invoices/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, invoiceHeaders, invoiceRow)
		},
	}
	return annotate(cmd, "GET", "/api/invoices/{id}")
}

// ---------------------------------------------------------------------------
// payments
// ---------------------------------------------------------------------------

func newPaymentsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payments",
		Short: "Inspect payments",
		Long:  "List and get payments.",
	}
	cmd.AddCommand(
		newPaymentsListCmd(app),
		newPaymentsGetCmd(app),
	)
	return cmd
}

func newPaymentsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List payments",
		Long:    "List all payments for the organization with optional pagination.",
		Example: "  gphq payments list\n  gphq payments list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/payments", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, paymentHeaders, paymentRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/payments")
}

func newPaymentsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <paymentId>",
		Short:   "Get a payment",
		Long:    "Fetch a single payment by ID.",
		Example: "  gphq payments get pay_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/payments/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, paymentHeaders, paymentRow)
		},
	}
	return annotate(cmd, "GET", "/api/payments/{id}")
}

// ---------------------------------------------------------------------------
// payment-methods
// ---------------------------------------------------------------------------

func newPaymentMethodsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-methods",
		Short: "Inspect payment methods",
		Long:  "Get saved payment methods by ID.",
	}
	cmd.AddCommand(
		newPaymentMethodsGetCmd(app),
	)
	return cmd
}

func newPaymentMethodsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <paymentMethodId>",
		Short:   "Get a payment method",
		Long:    "Fetch a single payment method by ID.",
		Example: "  gphq payment-methods get pm_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/payment-methods/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "GET", "/api/payment-methods/{id}")
}
