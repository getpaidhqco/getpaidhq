package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
	domain "getpaidhq/internal/core/domain"
)

var customerHeaders = []string{"ID", "EMAIL", "NAME", "PHONE", "CREATED"}

func customerRow(c api.CustomerResponse) []string {
	return []string{
		c.Id,
		c.Email,
		output.Str(strings.TrimSpace(c.FirstName + " " + c.LastName)),
		output.Str(c.Phone),
		output.Time(c.CreatedAt),
	}
}

func customerTable(app *App) func([]byte) error {
	return func(raw []byte) error {
		var c api.CustomerResponse
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("decoding customer response: %w", err)
		}
		return output.Table(app.Out, customerHeaders, [][]string{customerRow(c)})
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
			body, err := bodyOrData(cmd, func() (any, error) {
				email, _ := cmd.Flags().GetString("email")
				if email == "" {
					return nil, Usagef("--email is required (or use --data)")
				}
				firstName, _ := cmd.Flags().GetString("first-name")
				lastName, _ := cmd.Flags().GetString("last-name")
				phone, _ := cmd.Flags().GetString("phone")
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return api.CreateCustomerRequest{
					Email:     email,
					FirstName: firstName,
					LastName:  lastName,
					Phone:     phone,
					Metadata:  meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/customers", nil, body)
			if err != nil {
				return err
			}
			return render(app, raw, customerTable(app))
		},
	}
	f := cmd.Flags()
	f.String("email", "", "customer email address (required)")
	f.String("first-name", "", "first name")
	f.String("last-name", "", "last name")
	f.String("phone", "", "phone number")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/customers", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, customerHeaders, customerRow)
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/customers/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return render(app, raw, customerTable(app))
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
	cmd.AddCommand(
		newPMAddCmd(app),
		newPMUpdateCmd(app),
	)
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
			body, err := bodyOrData(cmd, func() (any, error) {
				psp, _ := cmd.Flags().GetString("psp")
				name, _ := cmd.Flags().GetString("name")
				pmType, _ := cmd.Flags().GetString("type")
				token, _ := cmd.Flags().GetString("token")
				if psp == "" || name == "" || pmType == "" || token == "" {
					return nil, Usagef("--psp, --name, --type and --token are required (or use --data)")
				}
				isDefault, _ := cmd.Flags().GetBool("default")
				return api.CreatePaymentMethodRequest{
					Psp:       psp,
					Name:      name,
					Type:      domain.PaymentMethodType(pmType),
					Token:     token,
					IsDefault: isDefault,
				}, nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/customers/%s/payment-methods", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("psp", "", "payment service provider (required)")
	f.String("name", "", "display name for the payment method (required)")
	f.String("type", "", "payment method type, e.g. card (required)")
	f.String("token", "", "PSP charge token (required)")
	f.Bool("default", false, "set as default payment method")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/customers/{id}/payment-methods")
}

func newPMUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <customerId> <paymentMethodId>",
		Short:   "Update a customer payment method",
		Long:    "Update a saved payment method. Only provided flags are sent.",
		Example: "  gphq customers payment-methods update cus_1 pm_1 --name \"Updated Card\"",
		Args:    exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				pmType, _ := cmd.Flags().GetString("type")
				token, _ := cmd.Flags().GetString("token")
				isDefault, _ := cmd.Flags().GetBool("default")
				return api.UpdatePaymentMethodRequest{
					Name:      name,
					Type:      domain.PaymentMethodType(pmType),
					Token:     token,
					IsDefault: isDefault,
				}, nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/customers/%s/payment-methods/%s", args[0], args[1])
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, path, nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "display name")
	f.String("type", "", "payment method type")
	f.String("token", "", "PSP charge token")
	f.Bool("default", false, "set as default payment method")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			path := fmt.Sprintf("/api/customers/%s/dunning-history", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "GET", "/api/customers/{id}/dunning-history")
}
