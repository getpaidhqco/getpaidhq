package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
)

var orderHeaders = []string{"ID", "REFERENCE", "CUSTOMER", "STATUS", "CURRENCY", "TOTAL", "CREATED"}

func orderRow(o api.OrderResponse) []string {
	return []string{
		o.Id,
		output.Str(o.Reference),
		o.CustomerId,
		o.Status,
		o.Currency,
		strconv.FormatInt(o.Total, 10),
		output.Time(o.CreatedAt),
	}
}

// subscriptionHeaders and subscriptionRow are defined here pending a later
// task that collects them into subscriptions.go.
var subscriptionHeaders = []string{"ID", "STATUS", "CURRENCY", "INTERVAL", "RENEWS", "CREATED"}

// domainSubscription mirrors the JSON emitted by ListSubscriptions, which
// returns []domain.Subscription — a struct with no json tags, so all field
// names are capitalized in the wire format.
type domainSubscription struct {
	Id                 string    `json:"Id"`
	Status             string    `json:"Status"`
	Currency           string    `json:"Currency"`
	BillingInterval    string    `json:"BillingInterval"`
	BillingIntervalQty int       `json:"BillingIntervalQty"`
	RenewsAt           time.Time `json:"RenewsAt"`
	CreatedAt          time.Time `json:"CreatedAt"`
}

func subscriptionRow(s domainSubscription) []string {
	return []string{
		s.Id,
		s.Status,
		s.Currency,
		fmt.Sprintf("%d %s", s.BillingIntervalQty, s.BillingInterval),
		output.Time(s.RenewsAt),
		output.Time(s.CreatedAt),
	}
}

// renderCreateOrder unwraps the CreateOrderResponse envelope and renders the
// inner OrderResponse as a single-row table.
func renderCreateOrder(app *App, raw []byte) error {
	if app.Output == "json" {
		return output.JSON(app.Out, raw)
	}
	var env struct {
		Order api.OrderResponse `json:"order"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("decoding create-order response: %w", err)
	}
	return output.Table(app.Out, orderHeaders, [][]string{orderRow(env.Order)})
}

// parseOrderItems parses repeated --item values of the form
// "product=<id>,price=<id>[,qty=<n>]".
func parseOrderItems(vals []string) ([]api.CartItem, error) {
	items := make([]api.CartItem, 0, len(vals))
	for _, v := range vals {
		kv, err := parseKV(strings.Split(v, ","), "item")
		if err != nil {
			return nil, err
		}
		for k := range kv {
			switch k {
			case "product", "price", "qty":
			default:
				return nil, Usagef("--item has unknown key %q (want product=,price=[,qty=])", k)
			}
		}
		item := api.CartItem{ProductId: kv["product"], PriceId: kv["price"], Quantity: 1}
		if item.ProductId == "" || item.PriceId == "" {
			return nil, Usagef("--item needs product=<id>,price=<id>[,qty=<n>], got %q", v)
		}
		if q, ok := kv["qty"]; ok {
			item.Quantity, err = strconv.Atoi(q)
			if err != nil || item.Quantity < 1 {
				return nil, Usagef("--item qty must be a positive integer, got %q", q)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func newOrdersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Manage orders",
		Long:  "Create, complete, list, and inspect orders and their subscriptions.",
	}
	cmd.AddCommand(
		newOrdersCreateCmd(app),
		newOrdersCompleteCmd(app),
		newOrdersGetCmd(app),
		newOrdersListCmd(app),
		newOrdersSubscriptionsCmd(app),
	)
	return cmd
}

func newOrdersCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an order",
		Long:  "Create a new order. Pass flags for common fields or --data for a raw JSON body.",
		Example: "  gphq orders create --customer-id cus_1 --psp paystack --currency NGN --item product=prod_1,price=pri_1\n" +
			"  gphq orders create --data '{\"psp_id\":\"paystack\",\"customer\":{\"id\":\"cus_1\"}}'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				psp, _ := cmd.Flags().GetString("psp")
				if psp == "" {
					return nil, Usagef("--psp is required (or use --data)")
				}
				customerID, _ := cmd.Flags().GetString("customer-id")
				email, _ := cmd.Flags().GetString("email")
				if customerID == "" && email == "" {
					return nil, Usagef("provide --customer-id or --email (or use --data)")
				}
				firstName, _ := cmd.Flags().GetString("first-name")
				lastName, _ := cmd.Flags().GetString("last-name")
				phone, _ := cmd.Flags().GetString("phone")
				paymentMethod, _ := cmd.Flags().GetString("payment-method")
				sessionID, _ := cmd.Flags().GetString("session")
				currency, _ := cmd.Flags().GetString("currency")
				itemVals, _ := cmd.Flags().GetStringArray("item")
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")

				items, err := parseOrderItems(itemVals)
				if err != nil {
					return nil, err
				}
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return api.CreateOrderRequest{
					Customer: api.CreateOrderRequestCustomer{
						ID:        customerID,
						Email:     email,
						FirstName: firstName,
						LastName:  lastName,
						Phone:     phone,
					},
					PaymentMethodId: paymentMethod,
					SessionId:       sessionID,
					PspId:           psp,
					Cart: api.CartInput{
						Currency: currency,
						Items:    items,
					},
					Metadata: meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/orders", nil, body)
			if err != nil {
				return err
			}
			return renderCreateOrder(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("customer-id", "", "customer ID")
	f.String("email", "", "customer email")
	f.String("first-name", "", "customer first name")
	f.String("last-name", "", "customer last name")
	f.String("phone", "", "customer phone")
	f.String("psp", "", "payment service provider ID (required)")
	f.String("payment-method", "", "payment method ID")
	f.String("session", "", "session ID")
	f.String("currency", "", "cart currency")
	f.StringArray("item", nil, "cart item: product=<id>,price=<id>[,qty=<n>] (repeatable)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/orders")
}

func newOrdersCompleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "complete <orderId>",
		Short:   "Complete an order",
		Long:    "Mark an order as complete, optionally providing payment method details.",
		Example: "  gphq orders complete ord_1 --payment-method-id pm_1\n  gphq orders complete ord_1 --data -",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				pmID, _ := cmd.Flags().GetString("payment-method-id")
				return api.CompleteOrderRequest{
					PaymentMethodId: pmID,
				}, nil
			})
			if err != nil {
				return err
			}
			path := "/api/orders/" + args[0] + "/complete"
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, orderHeaders, orderRow)
		},
	}
	f := cmd.Flags()
	f.String("payment-method-id", "", "payment method ID to use for completion")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/orders/{id}/complete")
}

func newOrdersGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <orderId>",
		Short:   "Get an order",
		Long:    "Fetch a single order by ID.",
		Example: "  gphq orders get ord_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/orders/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, orderHeaders, orderRow)
		},
	}
	return annotate(cmd, "GET", "/api/orders/{id}")
}

func newOrdersListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List orders",
		Long:    "List orders with optional pagination.",
		Example: "  gphq orders list\n  gphq orders list --page 2 --limit 5",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/orders", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, orderHeaders, orderRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/orders")
}

func newOrdersSubscriptionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subscriptions <orderId>",
		Short:   "List subscriptions for an order",
		Long:    "Fetch all subscriptions attached to an order. The response is a plain JSON array.",
		Example: "  gphq orders subscriptions ord_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/orders/" + args[0] + "/subscriptions"
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			if app.Output == "json" {
				return output.JSON(app.Out, raw)
			}
			var subs []domainSubscription
			if err := json.Unmarshal(raw, &subs); err != nil {
				return fmt.Errorf("decoding subscriptions response: %w", err)
			}
			rows := make([][]string, len(subs))
			for i, s := range subs {
				rows[i] = subscriptionRow(s)
			}
			return output.Table(app.Out, subscriptionHeaders, rows)
		},
	}
	return annotate(cmd, "GET", "/api/orders/{id}/subscriptions")
}
