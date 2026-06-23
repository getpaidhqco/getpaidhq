package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

var orderHeaders = []string{"ID", "REFERENCE", "CUSTOMER", "STATUS", "CURRENCY", "TOTAL", "CREATED"}

func orderRow(o apigen.OrderResponse) []string {
	return []string{
		o.ID.Or(""),
		output.Str(o.Reference.Or("")),
		o.CustomerID.Or(""),
		o.Status.Or(""),
		o.Currency.Or(""),
		strconv.FormatInt(o.Total.Or(0), 10),
		output.Time(o.CreatedAt.Or(time.Time{})),
	}
}

// createOrderOrderRow renders the order envelope returned by CreateOrder, which
// nests the order under a distinct generated type.
func createOrderOrderRow(o apigen.CreateOrderResponseOrder) []string {
	return []string{
		o.ID.Or(""),
		output.Str(o.Reference.Or("")),
		o.CustomerID.Or(""),
		o.Status.Or(""),
		o.Currency.Or(""),
		strconv.FormatInt(o.Total.Or(0), 10),
		output.Time(o.CreatedAt.Or(time.Time{})),
	}
}

// parseOrderItems parses repeated --item values of the form
// "product=<id>,price=<id>[,qty=<n>]".
func parseOrderItems(vals []string) ([]apigen.CreateOrderRequestCartItemsItem, error) {
	items := make([]apigen.CreateOrderRequestCartItemsItem, 0, len(vals))
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
		item := apigen.CreateOrderRequestCartItemsItem{ProductID: kv["product"], PriceID: kv["price"], Quantity: 1}
		if item.ProductID == "" || item.PriceID == "" {
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
		Example: "  gphq orders create --customer cus_1 --psp paystack --currency NGN --item product=prod_1,price=pri_1\n" +
			"  gphq orders create --data '{\"psp_id\":\"paystack\",\"customer\":{\"id\":\"cus_1\"}}'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.CreateOrderRequest) error {
				psp, _ := cmd.Flags().GetString("psp")
				if psp == "" {
					return Usagef("--psp is required (or use --data)")
				}
				customerID, _ := cmd.Flags().GetString("customer")
				email, _ := cmd.Flags().GetString("email")
				if customerID == "" && email == "" {
					return Usagef("provide --customer or --email (or use --data)")
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
					return err
				}
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}

				in.PspID = psp
				customer := apigen.CreateOrderRequestCustomer{}
				if customerID != "" {
					customer.ID = apigen.NewOptString(customerID)
				}
				if email != "" {
					customer.Email = apigen.NewOptString(email)
				}
				if firstName != "" {
					customer.FirstName = apigen.NewOptString(firstName)
				}
				if lastName != "" {
					customer.LastName = apigen.NewOptString(lastName)
				}
				if phone != "" {
					customer.Phone = apigen.NewOptString(phone)
				}
				in.Customer = customer
				if paymentMethod != "" {
					in.PaymentMethodID = apigen.NewOptString(paymentMethod)
				}
				if sessionID != "" {
					in.SessionID = apigen.NewOptString(sessionID)
				}
				cart := apigen.CreateOrderRequestCart{Items: items}
				if currency != "" {
					cart.Currency = apigen.NewOptString(currency)
				}
				in.Cart = apigen.NewOptCreateOrderRequestCart(cart)
				if meta != nil {
					in.Metadata = apigen.NewOptCreateOrderRequestMetadata(apigen.CreateOrderRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateOrder(cmd.Context(), body, apigen.CreateOrderParams{})
			env, err := expectOK[*apigen.CreateOrderResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, env.Order.Or(apigen.CreateOrderResponseOrder{}), orderHeaders, createOrderOrderRow)
		},
	}
	f := cmd.Flags()
	f.String("customer", "", "existing customer id")
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
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/orders")
}

func newOrdersCompleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "complete <orderId>",
		Short:   "Complete an order",
		Long:    "Mark an order as complete, optionally providing payment method details.",
		Example: "  gphq orders complete ord_1 --payment-method pm_1\n  gphq orders complete ord_1 --data -",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.CompleteOrderRequest) error {
				if s, _ := cmd.Flags().GetString("payment-method"); s != "" {
					in.PaymentMethodID = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CompleteOrder(cmd.Context(), body, apigen.CompleteOrderParams{ID: args[0]})
			order, err := expectOK[*apigen.OrderResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *order, orderHeaders, orderRow)
		},
	}
	f := cmd.Flags()
	f.String("payment-method", "", "payment method ID to use for completion")
	addDataFlag(cmd)
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
			res, err := app.API.GetOrder(cmd.Context(), apigen.GetOrderParams{ID: args[0]})
			order, err := expectOK[*apigen.OrderResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *order, orderHeaders, orderRow)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListOrders(cmd.Context(), apigen.ListOrdersParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, orderHeaders, orderRow)
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
			res, err := app.API.ListOrderSubscriptions(cmd.Context(), apigen.ListOrderSubscriptionsParams{ID: args[0]})
			subs, err := expectOK[*apigen.ListOrderSubscriptionsOKApplicationJSON](res, err)
			if err != nil {
				return err
			}
			if app.Output == "json" {
				return renderValue(app, subs)
			}
			rows := make([][]string, len(*subs))
			for i, s := range *subs {
				rows[i] = subRow(s)
			}
			return output.Table(app.Out, subHeaders, rows)
		},
	}
	return annotate(cmd, "GET", "/api/orders/{id}/subscriptions")
}
