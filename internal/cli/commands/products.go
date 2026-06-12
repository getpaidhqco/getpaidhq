package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
)

var productHeaders = []string{"ID", "NAME", "STATUS", "VARIANTS", "CREATED"}

func productRow(p api.ProductResponse) []string {
	return []string{
		p.Id,
		p.Name,
		string(p.Status),
		strconv.Itoa(len(p.Variants)),
		output.Time(p.CreatedAt),
	}
}

var variantHeaders = []string{"ID", "NAME", "PRICES", "CREATED"}

func variantRow(v api.VariantResponse) []string {
	return []string{
		v.Id,
		v.Name,
		strconv.Itoa(len(v.Prices)),
		output.Time(v.CreatedAt),
	}
}

func newProductsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "products",
		Short: "Manage products",
		Long:  "Create, list, get, update, and archive products, variants, and their prices.",
	}
	cmd.AddCommand(
		newProductsCreateCmd(app),
		newProductsListCmd(app),
		newProductsGetCmd(app),
		newProductsUpdateCmd(app),
		newProductsDeleteCmd(app),
		newProductsArchiveCmd(app),
		newProductsUnarchiveCmd(app),
		newProductsVariantsCmd(app),
	)
	return cmd
}

func newProductsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a product",
		Long: "Create a new product. The API requires at least one variant — use --data with a complete JSON body.\n" +
			"Flag-only creates (--name only) will be rejected by the server unless you provide variants via --data.",
		Example: "  gphq products create --name \"Acme Pro\" --description \"My product\"\n  gphq products create --data @product.json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
				return api.CreateProductRequest{
					Name:        name,
					Description: desc,
					Metadata:    meta,
					// Variants is required by server validation; flag-only path
					// sends an empty slice and will receive a 400 from the server.
					Variants: []api.CreateProductVariantRequest{},
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/products", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, productHeaders, productRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "product name")
	f.String("description", "", "product description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/products")
}

func newProductsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List products",
		Long:    "List products with optional pagination and status filter.",
		Example: "  gphq products list\n  gphq products list --status archived\n  gphq products list --status all --page 2",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			q := listQuery(cmd)
			status, _ := cmd.Flags().GetString("status")
			if status != "" {
				if q == nil {
					q = url.Values{}
				}
				q.Set("status", status)
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/products", q, nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, productHeaders, productRow)
		},
	}
	addListFlags(cmd)
	cmd.Flags().String("status", "", "filter by status: active, archived, or all (default: active)")
	return annotate(cmd, "GET", "/api/products")
}

func newProductsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a product",
		Long:    "Fetch a single product by ID.",
		Example: "  gphq products get prod_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/products/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, productHeaders, productRow)
		},
	}
	return annotate(cmd, "GET", "/api/products/{id}")
}

func newProductsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a product",
		Long:    "Update a product's name, description, or metadata.",
		Example: "  gphq products update prod_1 --name \"New Name\"\n  gphq products update prod_1 --data '{\"name\":\"New Name\"}'",
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
				return api.UpdateProductRequest{
					Name:        name,
					Description: desc,
					Metadata:    meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/products/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, productHeaders, productRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "product name (required)")
	f.String("description", "", "product description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "PATCH", "/api/products/{id}")
}

func newProductsDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete a product",
		Long:    "Permanently delete a product by ID.",
		Example: "  gphq products delete prod_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := app.Client.Do(cmd.Context(), http.MethodDelete, "/api/products/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderDeleted(app, args[0])
		},
	}
	return annotate(cmd, "DELETE", "/api/products/{id}")
}

func newProductsArchiveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "archive <id>",
		Short:   "Archive a product",
		Long:    "Archive a product, hiding it from default listings and preventing new sales.",
		Example: "  gphq products archive prod_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/products/"+args[0]+"/archive", nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, productHeaders, productRow)
		},
	}
	return annotate(cmd, "POST", "/api/products/{id}/archive")
}

func newProductsUnarchiveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unarchive <id>",
		Short:   "Unarchive a product",
		Long:    "Return an archived product to active status.",
		Example: "  gphq products unarchive prod_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/products/"+args[0]+"/unarchive", nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, productHeaders, productRow)
		},
	}
	return annotate(cmd, "POST", "/api/products/{id}/unarchive")
}

// newProductsVariantsCmd is the "products variants" sub-group.
func newProductsVariantsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variants",
		Short: "Manage product variants",
		Long:  "List or add variants for a product.",
	}
	cmd.AddCommand(
		newProductsVariantsListCmd(app),
		newProductsVariantsAddCmd(app),
	)
	return cmd
}

func newProductsVariantsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <productId>",
		Short:   "List variants of a product",
		Long:    "List all variants belonging to a product. Returns a paginated {data,meta} envelope.",
		Example: "  gphq products variants list prod_1",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/api/products/%s/variants", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, variantHeaders, variantRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/products/{id}/variants")
}

func newProductsVariantsAddCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <productId>",
		Short:   "Add a variant to a product",
		Long:    "Create a new variant under an existing product.",
		Example: "  gphq products variants add prod_1 --name Premium\n  gphq products variants add prod_1 --data @variant.json",
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
				return api.CreateVariantRequest{
					Name:        name,
					Description: desc,
					Metadata:    meta,
				}, nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/products/%s/variants", args[0])
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, path, nil, body)
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
	return annotate(cmd, "POST", "/api/products/{id}/variants")
}
