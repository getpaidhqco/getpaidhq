package cli

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

// paymentHeaders and paymentRow are the canonical table shape for
// apigen.PaymentResponse, used by both the top-level payments commands and
// subscriptions payments sub-command.
var paymentHeaders = []string{"ID", "STATUS", "CURRENCY", "AMOUNT", "REFERENCE", "CREATED"}

func paymentRow(p apigen.PaymentResponse) []string {
	return []string{
		p.ID.Or(""),
		p.Status.Or(""),
		p.Currency.Or(""),
		strconv.FormatInt(p.Amount.Or(0), 10),
		output.Str(p.Reference.Or("")),
		output.Time(p.CreatedAt.Or(time.Time{})),
	}
}

// invoiceHeaders and invoiceRow are the canonical table shape for
// apigen.InvoiceResponse, used by both the top-level invoices commands and
// subscriptions invoices sub-command.
var invoiceHeaders = []string{"ID", "STATUS", "CURRENCY", "TOTAL", "CYCLE", "PERIOD END", "CREATED"}

func invoiceRow(i apigen.InvoiceResponse) []string {
	return []string{
		i.ID.Or(""),
		string(i.Status.Or("")),
		i.Currency.Or(""),
		strconv.FormatInt(i.Total.Or(0), 10),
		strconv.Itoa(i.Cycle.Or(0)),
		output.Time(i.PeriodEnd.Or(time.Time{})),
		output.Time(i.CreatedAt.Or(time.Time{})),
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListInvoices(cmd.Context(), apigen.ListInvoicesParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, invoiceHeaders, invoiceRow)
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
			res, err := app.API.GetInvoice(cmd.Context(), apigen.GetInvoiceParams{ID: args[0]})
			inv, err := expectOK[*apigen.InvoiceResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *inv, invoiceHeaders, invoiceRow)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListPayments(cmd.Context(), apigen.ListPaymentsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, paymentHeaders, paymentRow)
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
			res, err := app.API.GetPayment(cmd.Context(), apigen.GetPaymentParams{ID: args[0]})
			pay, err := expectOK[*apigen.PaymentResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *pay, paymentHeaders, paymentRow)
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
			res, err := app.API.GetPaymentMethod(cmd.Context(), apigen.GetPaymentMethodParams{ID: args[0]})
			pm, err := expectOK[*apigen.PaymentMethodResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, pm)
		},
	}
	return annotate(cmd, "GET", "/api/payment-methods/{id}")
}
