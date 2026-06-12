package commands

import (
	"net/http"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
)

func newCartsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "carts",
		Short: "Manage carts",
		Long:  "Add and remove items from shopping carts.",
	}
	cmd.AddCommand(
		newCartsAddCmd(app),
		newCartsRemoveCmd(app),
	)
	return cmd
}

func newCartsAddCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <cartId>",
		Short:   "Add a product to a cart",
		Long:    "Add a product variant/price to an existing cart. Pass --product and --price, or --data for a raw JSON body.",
		Example: "  gphq carts add cart_1 --product prod_1 --price pri_1 --qty 2\n  gphq carts add cart_1 --data '{\"product_id\":\"prod_1\",\"price_id\":\"pri_1\",\"quantity\":1}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				product, _ := cmd.Flags().GetString("product")
				price, _ := cmd.Flags().GetString("price")
				if product == "" || price == "" {
					return nil, Usagef("--product and --price are required (or use --data)")
				}
				qty, _ := cmd.Flags().GetInt("qty")
				return api.AddItemRequest{
					ProductId: product,
					PriceId:   price,
					Quantity:  qty,
				}, nil
			})
			if err != nil {
				return err
			}
			path := "/api/carts/" + args[0] + "/add"
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("product", "", "product ID (required)")
	f.String("price", "", "price ID (required)")
	f.Int("qty", 1, "quantity")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/carts/{id}/add")
}

func newCartsRemoveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <cartId>",
		Short:   "Remove an item from a cart",
		Long:    "Remove a line item from a cart by its item ID.",
		Example: "  gphq carts remove cart_1 --item-id item_abc\n  gphq carts remove cart_1 --data '{\"id\":\"item_abc\"}'",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				itemID, _ := cmd.Flags().GetString("item-id")
				if itemID == "" {
					return nil, Usagef("--item-id is required (or use --data)")
				}
				return api.RemoveItemRequest{
					Id: itemID,
				}, nil
			})
			if err != nil {
				return err
			}
			path := "/api/carts/" + args[0] + "/remove"
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("item-id", "", "cart item ID to remove (required)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/carts/{id}/remove")
}
