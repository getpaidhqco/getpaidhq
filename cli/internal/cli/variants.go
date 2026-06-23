package cli

import (
	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
)

func newVariantsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variants",
		Short: "Manage variants",
		Long:  "Get, update, delete, and list prices for variants.",
	}
	cmd.AddCommand(
		newVariantsGetCmd(app),
		newVariantsUpdateCmd(app),
		newVariantsDeleteCmd(app),
		newVariantsPricesCmd(app),
	)
	return cmd
}

func newVariantsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <variantId>",
		Short:   "Get a variant",
		Long:    "Fetch a single variant by ID.",
		Example: "  gphq variants get var_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetVariant(cmd.Context(), apigen.GetVariantParams{VariantId: args[0]})
			variant, err := expectOK[*apigen.VariantResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *variant, variantHeaders, variantRow)
		},
	}
	return annotate(cmd, "GET", "/api/variants/{variantId}")
}

func newVariantsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <variantId>",
		Short:   "Update a variant",
		Long:    "Update a variant's name, description, or metadata.",
		Example: "  gphq variants update var_1 --name \"Premium v2\"\n  gphq variants update var_1 --data @variant.json",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.UpdateVariantRequest) error {
				name, _ := cmd.Flags().GetString("name")
				if name == "" {
					return Usagef("--name is required (or use --data)")
				}
				in.Name = name
				if s, _ := cmd.Flags().GetString("description"); s != "" {
					in.Description = apigen.NewOptString(s)
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptUpdateVariantRequestMetadata(apigen.UpdateVariantRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateVariant(cmd.Context(), body, apigen.UpdateVariantParams{VariantId: args[0]})
			variant, err := expectOK[*apigen.VariantResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *variant, variantHeaders, variantRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "variant name (required)")
	f.String("description", "", "variant description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
	return annotate(cmd, "PUT", "/api/variants/{variantId}")
}

func newVariantsDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <variantId>",
		Short:   "Delete a variant",
		Long:    "Permanently delete a variant by ID.",
		Example: "  gphq variants delete var_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.DeleteVariant(cmd.Context(), apigen.DeleteVariantParams{VariantId: args[0]})
			if _, err := expectOK[*apigen.EmptyResponse](res, err); err != nil {
				return err
			}
			return renderDeleted(app, args[0])
		},
	}
	return annotate(cmd, "DELETE", "/api/variants/{variantId}")
}

func newVariantsPricesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "prices <variantId>",
		Short:   "List prices of a variant",
		Long:    "List all prices for a variant. Returns a paginated {data,meta} envelope.",
		Example: "  gphq variants prices var_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListVariantPrices(cmd.Context(), apigen.ListVariantPricesParams{
				VariantId: args[0],
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, priceHeaders, priceRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/variants/{variantId}/prices")
}
