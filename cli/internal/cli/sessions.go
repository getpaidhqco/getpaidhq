package cli

import (
	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
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
			body, err := bindBody(cmd, func(in *apigen.CreateSessionRequest) error {
				currency, _ := cmd.Flags().GetString("currency")
				country, _ := cmd.Flags().GetString("country")
				if currency == "" || country == "" {
					return Usagef("--currency and --country are required (or use --data)")
				}
				in.Currency = currency
				in.Country = country
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptCreateSessionRequestMetadata(apigen.CreateSessionRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateSession(cmd.Context(), body, apigen.CreateSessionParams{})
			sess, err := expectOK[*apigen.CreateSessionResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, sess)
		},
	}
	f := cmd.Flags()
	f.String("currency", "", "session currency (required)")
	f.String("country", "", "session country (required)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/sessions")
}
