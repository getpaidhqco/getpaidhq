package commands

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
	"getpaidhq/internal/core/domain"
)

var priceHeaders = []string{"ID", "LABEL", "CATEGORY", "SCHEME", "CURRENCY", "UNIT PRICE", "INTERVAL", "CREATED"}

func priceRow(p api.PriceResponse) []string {
	interval := "-"
	if p.BillingInterval != "" && p.BillingInterval != domain.BillingIntervalNone {
		interval = fmt.Sprintf("%d %s", p.BillingIntervalQty, p.BillingInterval)
	}
	return []string{
		p.Id,
		output.Str(p.Label),
		string(p.Category),
		string(p.Scheme),
		string(p.Currency),
		strconv.FormatInt(p.UnitPrice, 10),
		interval,
		output.Time(p.CreatedAt),
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

// priceRequestFromFlags reads price flags from cmd and returns a populated
// CreatePriceRequest. --variant, --category, --scheme, and --currency are
// always required.
func priceRequestFromFlags(cmd *cobra.Command) (any, error) {
	variantID, _ := cmd.Flags().GetString("variant")
	category, _ := cmd.Flags().GetString("category")
	scheme, _ := cmd.Flags().GetString("scheme")
	currency, _ := cmd.Flags().GetString("currency")
	if variantID == "" || category == "" || scheme == "" || currency == "" {
		return nil, Usagef("--variant, --category, --scheme and --currency are required (or use --data)")
	}
	label, _ := cmd.Flags().GetString("label")
	unitPrice, _ := cmd.Flags().GetInt64("unit-price")
	interval, _ := cmd.Flags().GetString("interval")
	intervalQty, _ := cmd.Flags().GetInt("interval-qty")
	trialInterval, _ := cmd.Flags().GetString("trial-interval")
	trialQty, _ := cmd.Flags().GetInt("trial-qty")
	cycles, _ := cmd.Flags().GetInt("cycles")
	metaPairs, _ := cmd.Flags().GetStringArray("metadata")
	meta, err := parseKV(metaPairs, "metadata")
	if err != nil {
		return nil, err
	}
	return api.CreatePriceRequest{
		VariantId:          variantID,
		Category:           domain.PriceCategory(category),
		Scheme:             domain.PriceScheme(scheme),
		Currency:           currency,
		Label:              label,
		UnitPrice:          unitPrice,
		BillingInterval:    domain.BillingInterval(interval),
		BillingIntervalQty: intervalQty,
		TrialInterval:      domain.BillingInterval(trialInterval),
		TrialIntervalQty:   trialQty,
		Cycles:             cycles,
		Metadata:           meta,
	}, nil
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
			body, err := bodyOrData(cmd, func() (any, error) {
				return priceRequestFromFlags(cmd)
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/prices", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, priceHeaders, priceRow)
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
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/prices/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, priceHeaders, priceRow)
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
			body, err := bodyOrData(cmd, func() (any, error) {
				return priceRequestFromFlags(cmd)
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPatch, "/api/prices/"+args[0], nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, priceHeaders, priceRow)
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
			_, err := app.Client.Do(cmd.Context(), http.MethodDelete, "/api/prices/"+args[0], nil, nil)
			if err != nil {
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
	f.String("data", "", "raw JSON body (@file, -, or inline; use for tiers/filters)")
}
