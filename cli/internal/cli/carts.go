package cli

import (
	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
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
			body, err := bindBody(cmd, func(in *apigen.AddItemRequest) error {
				product, _ := cmd.Flags().GetString("product")
				price, _ := cmd.Flags().GetString("price")
				if product == "" || price == "" {
					return Usagef("--product and --price are required (or use --data)")
				}
				qty, _ := cmd.Flags().GetInt("qty")
				if qty < 1 {
					return Usagef("--qty must be a positive integer")
				}
				in.ProductID = product
				in.PriceID = price
				in.Quantity = apigen.NewOptInt(qty)
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.AddProductToCart(cmd.Context(), body, apigen.AddProductToCartParams{ID: args[0]})
			cart, err := expectOK[*apigen.CartResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, cart)
		},
	}
	f := cmd.Flags()
	f.String("product", "", "product ID (required)")
	f.String("price", "", "price ID (required)")
	f.Int("qty", 1, "quantity")
	addDataFlag(cmd)
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
			body, err := bindBody(cmd, func(in *apigen.RemoveItemRequest) error {
				itemID, _ := cmd.Flags().GetString("item-id")
				if itemID == "" {
					return Usagef("--item-id is required (or use --data)")
				}
				in.ID = apigen.NewOptString(itemID)
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.RemoveItemFromCart(cmd.Context(), body, apigen.RemoveItemFromCartParams{ID: args[0]})
			cart, err := expectOK[*apigen.CartResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, cart)
		},
	}
	f := cmd.Flags()
	f.String("item-id", "", "cart item ID to remove (required)")
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/carts/{id}/remove")
}
