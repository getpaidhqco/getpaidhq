package commands

import (
	"net/http"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
)

func newSessionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage sessions",
		Long:  "Create checkout sessions.",
	}
	cmd.AddCommand(
		newSessionsCreateCmd(app),
	)
	return cmd
}

func newSessionsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a session",
		Long:  "Create a new checkout session. Pass --currency and --country, or --data for a raw JSON body.",
		Example: "  gphq sessions create --currency USD --country US\n" +
			"  gphq sessions create --currency NGN --country NG --metadata src=api\n" +
			"  gphq sessions create --data '{\"currency\":\"USD\",\"country\":\"US\"}'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				currency, _ := cmd.Flags().GetString("currency")
				country, _ := cmd.Flags().GetString("country")
				if currency == "" || country == "" {
					return nil, Usagef("--currency and --country are required (or use --data)")
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return api.CreateSessionRequest{
					Currency: currency,
					Country:  country,
					Metadata: meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/sessions", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("currency", "", "session currency (required)")
	f.String("country", "", "session country (required)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/sessions")
}
