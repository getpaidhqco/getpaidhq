package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

var customerHeaders = []string{"ID", "EMAIL", "NAME", "PHONE", "CREATED"}

func customerRow(c apigen.CustomerResponse) []string {
	name := strings.TrimSpace(c.FirstName.Or("") + " " + c.LastName.Or(""))
	return []string{
		c.ID.Or(""),
		c.Email.Or(""),
		output.Str(name),
		output.Str(c.Phone.Or("")),
		output.Time(c.CreatedAt.Or(time.Time{})),
	}
}

func newCustomersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customers",
		Short: "Manage customers",
		Long:  "Create, list, get, and manage payment methods for customers.",
	}
	cmd.AddCommand(
		newCustomersCreateCmd(app),
		newCustomersListCmd(app),
		newCustomersGetCmd(app),
		newCustomersPaymentMethodsCmd(app),
		newCustomersDunningHistoryCmd(app),
	)
	return cmd
}

func newCustomersCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a customer",
		Long:    "Create a new customer. Pass flags for common fields or --data for a raw JSON body.",
		Example: "  gphq customers create --email ada@example.com --first-name Ada\n  gphq customers create --data '{\"email\":\"ada@example.com\"}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.CreateCustomerInput) error {
				email, _ := cmd.Flags().GetString("email")
				if email == "" {
					return Usagef("--email is required (or use --data)")
				}
				in.Email = email
				if s, _ := cmd.Flags().GetString("first-name"); s != "" {
					in.FirstName = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("last-name"); s != "" {
					in.LastName = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("phone"); s != "" {
					in.Phone = apigen.NewOptString(s)
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptCreateCustomerInputMetadata(apigen.CreateCustomerInputMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateCustomer(cmd.Context(), body, apigen.CreateCustomerParams{})
			cust, err := expectOK[*apigen.CustomerResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *cust, customerHeaders, customerRow)
		},
	}
	f := cmd.Flags()
	f.String("email", "", "customer email address (required)")
	f.String("first-name", "", "first name")
	f.String("last-name", "", "last name")
	f.String("phone", "", "phone number")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/customers")
}

func newCustomersListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List customers",
		Long:    "List customers with optional pagination.",
		Example: "  gphq customers list\n  gphq customers list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListCustomers(cmd.Context(), apigen.ListCustomersParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, customerHeaders, customerRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/customers")
}

func newCustomersGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <customerId>",
		Short:   "Get a customer",
		Long:    "Fetch a single customer by ID.",
		Example: "  gphq customers get cus_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetCustomer(cmd.Context(), apigen.GetCustomerParams{ID: args[0]})
			cust, err := expectOK[*apigen.CustomerResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *cust, customerHeaders, customerRow)
		},
	}
	return annotate(cmd, "GET", "/api/customers/{id}")
}

func newCustomersPaymentMethodsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-methods",
		Short: "Manage customer payment methods",
		Long:  "Add and update payment methods for a customer.",
	}
	cmd.AddCommand(newPMAddCmd(app), newPMUpdateCmd(app))
	return cmd
}

func newPMAddCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <customerId>",
		Short:   "Add a payment method to a customer",
		Long:    "Attach a PSP payment token to a customer as a saved payment method.",
		Example: "  gphq customers payment-methods add cus_1 --psp paystack --name \"My Card\" --type card --token tok_abc",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := rawBodyJSON(cmd, func() (any, error) {
				psp, _ := cmd.Flags().GetString("psp")
				name, _ := cmd.Flags().GetString("name")
				pmType, _ := cmd.Flags().GetString("type")
				token, _ := cmd.Flags().GetString("token")
				if psp == "" || name == "" || pmType == "" || token == "" {
					return nil, Usagef("--psp, --name, --type and --token are required (or use --data)")
				}
				isDefault, _ := cmd.Flags().GetBool("default")
				// This endpoint binds the port input directly (PascalCase wire format,
				// unlike every other route) — see the server's customer_handler.go.
				return map[string]any{
					"Psp": psp, "Name": name, "Type": pmType, "Token": token, "IsDefault": isDefault,
				}, nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateCustomerPaymentMethod(cmd.Context(),
				apigen.CreatePaymentMethodInput(body),
				apigen.CreateCustomerPaymentMethodParams{ID: args[0]})
			pm, err := expectOK[*apigen.PaymentMethodResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, pm)
		},
	}
	f := cmd.Flags()
	f.String("psp", "", "payment service provider (required)")
	f.String("name", "", "display name for the payment method (required)")
	f.String("type", "", "payment method type, e.g. card (required)")
	f.String("token", "", "PSP charge token (required)")
	f.Bool("default", false, "set as default payment method")
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/customers/{id}/payment-methods")
}

func newPMUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <customerId> <paymentMethodId>",
		Short:   "Update a customer payment method",
		Long:    "Update a saved payment method. Unset flags are sent as empty values, which the server ignores.",
		Example: "  gphq customers payment-methods update cus_1 pm_1 --name \"Updated Card\"",
		Args:    exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := rawBodyJSON(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				pmType, _ := cmd.Flags().GetString("type")
				token, _ := cmd.Flags().GetString("token")
				isDefault, _ := cmd.Flags().GetBool("default")
				return map[string]any{
					"Name": name, "Type": pmType, "Token": token, "IsDefault": isDefault,
				}, nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateCustomerPaymentMethod(cmd.Context(),
				apigen.UpdatePaymentMethodInput(body),
				apigen.UpdateCustomerPaymentMethodParams{ID: args[0], Pmid: args[1]})
			pm, err := expectOK[*apigen.PaymentMethodResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, pm)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "display name")
	f.String("type", "", "payment method type")
	f.String("token", "", "PSP charge token")
	f.Bool("default", false, "set as default payment method")
	addDataFlag(cmd)
	return annotate(cmd, "PUT", "/api/customers/{id}/payment-methods/{pmid}")
}

func newCustomersDunningHistoryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dunning-history <customerId>",
		Short:   "Get a customer's dunning history",
		Long:    "Fetch the dunning campaign history for a customer.",
		Example: "  gphq customers dunning-history cus_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetCustomerDunningHistory(cmd.Context(), apigen.GetCustomerDunningHistoryParams{ID: args[0]})
			hist, err := expectOK[*apigen.CustomerDunningHistoryResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, hist)
		},
	}
	return annotate(cmd, "GET", "/api/customers/{id}/dunning-history")
}
