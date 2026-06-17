package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/variants/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, variantHeaders, variantRow)
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
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				if name == "" {
					return nil, Usagef("--name is required (or use --data)")
				}
				desc, _ := cmd.Flags().GetString("description")
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return api.UpdateVariantRequest{
					Name:        name,
					Description: desc,
					Metadata:    meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, "/api/variants/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, variantHeaders, variantRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "variant name (required)")
	f.String("description", "", "variant description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
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
			_, err := app.Client.Do(cmd.Context(), http.MethodDelete, "/api/variants/"+args[0], nil, nil)
			if err != nil {
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
			path := fmt.Sprintf("/api/variants/%s/prices", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, priceHeaders, priceRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/variants/{variantId}/prices")
}
