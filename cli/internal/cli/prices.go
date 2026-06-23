package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

var priceHeaders = []string{"ID", "LABEL", "CATEGORY", "SCHEME", "CURRENCY", "UNIT PRICE", "INTERVAL", "CREATED"}

func priceRow(p apigen.PriceResponse) []string {
	interval := "-"
	if bi := p.BillingInterval.Or(""); bi != "" && bi != "none" {
		interval = fmt.Sprintf("%d %s", p.BillingIntervalQty.Or(0), bi)
	}
	return []string{
		p.ID.Or(""),
		output.Str(p.Label.Or("")),
		p.Category.Or(""),
		p.Scheme.Or(""),
		p.Currency.Or(""),
		strconv.FormatInt(p.UnitPrice.Or(0), 10),
		interval,
		output.Time(p.CreatedAt.Or(time.Time{})),
	}
}

func newPricesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices",
		Short: "Manage prices",
		Long:  "Create, get, update, and delete prices.",
	}
	cmd.AddCommand(
		newPricesCreateCmd(app),
		newPricesGetCmd(app),
		newPricesUpdateCmd(app),
		newPricesDeleteCmd(app),
	)
	return cmd
}

// priceRequestFromFlags populates a CreatePriceRequest from price flags.
// --variant, --category, --scheme, and --currency are always required.
func priceRequestFromFlags(cmd *cobra.Command, in *apigen.CreatePriceRequest) error {
	variantID, _ := cmd.Flags().GetString("variant")
	category, _ := cmd.Flags().GetString("category")
	scheme, _ := cmd.Flags().GetString("scheme")
	currency, _ := cmd.Flags().GetString("currency")
	if variantID == "" || category == "" || scheme == "" || currency == "" {
		return Usagef("--variant, --category, --scheme and --currency are required (or use --data)")
	}
	in.VariantID = variantID
	in.Category = apigen.CreatePriceRequestCategory(category)
	in.Scheme = apigen.CreatePriceRequestScheme(scheme)
	in.Currency = currency
	if s, _ := cmd.Flags().GetString("label"); s != "" {
		in.Label = apigen.NewOptString(s)
	}
	if v, _ := cmd.Flags().GetInt64("unit-price"); v != 0 {
		in.UnitPrice = apigen.NewOptInt64(v)
	}
	if s, _ := cmd.Flags().GetString("interval"); s != "" {
		in.BillingInterval = apigen.NewOptCreatePriceRequestBillingInterval(apigen.CreatePriceRequestBillingInterval(s))
	}
	if v, _ := cmd.Flags().GetInt("interval-qty"); v != 0 {
		in.BillingIntervalQty = apigen.NewOptInt(v)
	}
	if s, _ := cmd.Flags().GetString("trial-interval"); s != "" {
		in.TrialInterval = apigen.NewOptCreatePriceRequestTrialInterval(apigen.CreatePriceRequestTrialInterval(s))
	}
	if v, _ := cmd.Flags().GetInt("trial-qty"); v != 0 {
		in.TrialIntervalQty = apigen.NewOptInt(v)
	}
	if v, _ := cmd.Flags().GetInt("cycles"); v != 0 {
		in.Cycles = apigen.NewOptInt(v)
	}
	metaPairs, _ := cmd.Flags().GetStringArray("metadata")
	meta, err := parseKV(metaPairs, "metadata")
	if err != nil {
		return err
	}
	if meta != nil {
		in.Metadata = apigen.NewOptCreatePriceRequestMetadata(apigen.CreatePriceRequestMetadata(meta))
	}
	return nil
}

func newPricesCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a price",
		Long:  "Create a new price for a variant. Tiers and filter fields can only be set via --data.",
		Example: "  gphq prices create --variant var_1 --category subscription --scheme fixed --currency USD --unit-price 999 --interval month --interval-qty 1\n" +
			"  gphq prices create --data @price.json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bindBody(cmd, func(in *apigen.CreatePriceRequest) error {
				return priceRequestFromFlags(cmd, in)
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreatePrice(cmd.Context(), body, apigen.CreatePriceParams{})
			price, err := expectOK[*apigen.PriceResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *price, priceHeaders, priceRow)
		},
	}
	addPriceFlags(cmd)
	return annotate(cmd, "POST", "/api/prices")
}

func newPricesGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <priceId>",
		Short:   "Get a price",
		Long:    "Fetch a single price by ID.",
		Example: "  gphq prices get pri_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.GetPrice(cmd.Context(), apigen.GetPriceParams{PriceId: args[0]})
			price, err := expectOK[*apigen.PriceResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *price, priceHeaders, priceRow)
		},
	}
	return annotate(cmd, "GET", "/api/prices/{priceId}")
}

func newPricesUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <priceId>",
		Short: "Update a price",
		Long: "Update a price. The server reuses the create schema, so --variant (the variant the price belongs to) " +
			"must be re-supplied on every flag-based update.",
		Example: "  gphq prices update pri_1 --variant var_1 --category subscription --scheme fixed --currency USD --unit-price 1299\n" +
			"  gphq prices update pri_1 --data @price.json",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bindBody(cmd, func(in *apigen.CreatePriceRequest) error {
				return priceRequestFromFlags(cmd, in)
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdatePrice(cmd.Context(), body, apigen.UpdatePriceParams{PriceId: args[0]})
			price, err := expectOK[*apigen.PriceResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *price, priceHeaders, priceRow)
		},
	}
	addPriceFlags(cmd)
	return annotate(cmd, "PATCH", "/api/prices/{priceId}")
}

func newPricesDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <priceId>",
		Short:   "Delete a price",
		Long:    "Permanently delete a price by ID.",
		Example: "  gphq prices delete pri_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := app.API.DeletePrice(cmd.Context(), apigen.DeletePriceParams{PriceId: args[0]})
			if _, err := expectOK[*apigen.EmptyResponse](res, err); err != nil {
				return err
			}
			return renderDeleted(app, args[0])
		},
	}
	return annotate(cmd, "DELETE", "/api/prices/{priceId}")
}

// addPriceFlags adds shared flags to a prices create/update command.
func addPriceFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("variant", "", "variant ID (required)")
	f.String("category", "", "price category: one_time, subscription, free, variable (required)")
	f.String("scheme", "", "price scheme: fixed, tiered, volume, graduated, package (required)")
	f.String("currency", "", "ISO 4217 currency code, e.g. USD (required)")
	f.String("label", "", "display label")
	f.Int64("unit-price", 0, "unit price in smallest currency unit (e.g. cents)")
	f.String("interval", "", "billing interval: none, minute, hour, day, week, month, year")
	f.Int("interval-qty", 0, "billing interval quantity")
	f.String("trial-interval", "", "trial period interval: none, minute, hour, day, week, month, year")
	f.Int("trial-qty", 0, "trial period quantity")
	f.Int("cycles", 0, "number of billing cycles (0 = unlimited)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
}
