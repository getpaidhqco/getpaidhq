package cli

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

var productHeaders = []string{"ID", "NAME", "STATUS", "VARIANTS", "CREATED"}

func productRow(p apigen.ProductResponse) []string {
	variants, _ := p.Variants.Get()
	return []string{
		p.ID.Or(""),
		p.Name.Or(""),
		p.Status.Or(""),
		strconv.Itoa(len(variants)),
		output.Time(p.CreatedAt.Or(time.Time{})),
	}
}

var variantHeaders = []string{"ID", "NAME", "PRICES", "CREATED"}

func variantRow(v apigen.VariantResponse) []string {
	prices, _ := v.Prices.Get()
	return []string{
		v.ID.Or(""),
		v.Name.Or(""),
		strconv.Itoa(len(prices)),
		output.Time(v.CreatedAt.Or(time.Time{})),
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
			body, err := bindBody(cmd, func(in *apigen.CreateProductRequest) error {
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
					in.Metadata = apigen.NewOptCreateProductRequestMetadata(apigen.CreateProductRequestMetadata(meta))
				}
				// Variants is required by server validation; flag-only path
				// sends an empty slice and will receive a 400 from the server.
				in.Variants = []apigen.CreateProductRequestVariantsItem{}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateProduct(cmd.Context(), body, apigen.CreateProductParams{})
			prod, err := expectOK[*apigen.ProductResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *prod, productHeaders, productRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "product name")
	f.String("description", "", "product description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListProducts(cmd.Context(), apigen.ListProductsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, productHeaders, productRow)
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
			res, err := app.API.GetProduct(cmd.Context(), apigen.GetProductParams{ID: args[0]})
			prod, err := expectOK[*apigen.ProductResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *prod, productHeaders, productRow)
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
			body, err := bindBody(cmd, func(in *apigen.UpdateProductRequest) error {
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
					in.Metadata = apigen.NewOptUpdateProductRequestMetadata(apigen.UpdateProductRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateProduct(cmd.Context(), body, apigen.UpdateProductParams{ID: args[0]})
			prod, err := expectOK[*apigen.ProductResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *prod, productHeaders, productRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "product name (required)")
	f.String("description", "", "product description")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
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
			res, err := app.API.DeleteProduct(cmd.Context(), apigen.DeleteProductParams{ID: args[0]})
			if _, err := expectOK[*apigen.EmptyResponse](res, err); err != nil {
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
			res, err := app.API.ArchiveProduct(cmd.Context(), apigen.ArchiveProductParams{ID: args[0]})
			prod, err := expectOK[*apigen.ProductResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *prod, productHeaders, productRow)
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
			res, err := app.API.UnarchiveProduct(cmd.Context(), apigen.UnarchiveProductParams{ID: args[0]})
			prod, err := expectOK[*apigen.ProductResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *prod, productHeaders, productRow)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListProductVariants(cmd.Context(), apigen.ListProductVariantsParams{
				ID:        args[0],
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, variantHeaders, variantRow)
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
			body, err := bindBody(cmd, func(in *apigen.CreateVariantRequest) error {
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
					in.Metadata = apigen.NewOptCreateVariantRequestMetadata(apigen.CreateVariantRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateProductVariant(cmd.Context(), body, apigen.CreateProductVariantParams{ID: args[0]})
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
	return annotate(cmd, "POST", "/api/products/{id}/variants")
}
