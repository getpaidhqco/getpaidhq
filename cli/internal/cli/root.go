// Package cli wires the gphq command-line client: configuration resolution
// (flags > env > config file > defaults), the cobra root command, and process
// exit codes. It talks to the API exclusively through the OpenAPI-generated
// client in internal/apigen — it imports nothing from the server.
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
)

var version = "dev" // set via -ldflags "-X .../cli/internal/cli.version=..."

// App carries the resolved configuration and the generated API client into
// every command.
type App struct {
	Out, ErrOut io.Writer
	API         *apigen.Client
	Output      string // "table" or "json"
}

// Run executes the CLI. Exit codes: 0 success, 1 API/network/config error,
// 2 usage error.
func Run(args []string, in io.Reader, out, errOut io.Writer) int {
	root := NewRootCmd(in, out, errOut)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var usage *UsageError
		if errors.As(err, &usage) || strings.HasPrefix(err.Error(), "unknown command") {
			fmt.Fprintf(errOut, "error: %s\n", err.Error())
			return 2
		}
		fmt.Fprintln(errOut, FormatError(err))
		return 1
	}
	return 0
}

func NewRootCmd(in io.Reader, out, errOut io.Writer) *cobra.Command {
	app := &App{Out: out, ErrOut: errOut}

	root := &cobra.Command{
		Use:   "gphq",
		Short: "Command-line client for the GetPaidHQ API",
		Long: `gphq is the command-line client for the GetPaidHQ subscription-billing API.

Authentication uses an organization API key sent as the x-api-key header.
Configuration precedence: flags > GPHQ_* environment variables >
~/.config/gphq/config.toml > defaults.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)
	root.DisableAutoGenTag = true // keep generated docs deterministic

	flags := root.PersistentFlags()
	flags.String("api-key", "", "API key (env GPHQ_API_KEY)")
	flags.String("base-url", "http://localhost:10081", "API base URL (env GPHQ_BASE_URL)")
	flags.StringP("output", "o", "table", "output format: table|json (env GPHQ_OUTPUT)")

	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return Usagef("%s", err.Error())
	})

	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		v := viper.New()
		v.SetEnvPrefix("GPHQ")
		bind := func(key, flag string) {
			_ = v.BindPFlag(key, flags.Lookup(flag))
			_ = v.BindEnv(key)
		}
		bind("api_key", "api-key")
		bind("base_url", "base-url")
		bind("output", "output")
		// Resolve config path: $XDG_CONFIG_HOME/gphq/config.toml, or
		// $HOME/.config/gphq/config.toml. Skipped if home dir is unavailable.
		var cfgPath string
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			cfgPath = filepath.Join(xdg, "gphq", "config.toml")
		} else if home, err := os.UserHomeDir(); err == nil {
			cfgPath = filepath.Join(home, ".config", "gphq", "config.toml")
		}
		if cfgPath != "" {
			v.SetConfigFile(cfgPath)
			if err := v.ReadInConfig(); err != nil {
				var notFound viper.ConfigFileNotFoundError
				if !errors.Is(err, fs.ErrNotExist) && !errors.As(err, &notFound) {
					return fmt.Errorf("reading config file %s: %w", cfgPath, err)
				}
			}
		}
		app.Output = v.GetString("output")
		if app.Output == "" {
			app.Output = "table"
		}
		if app.Output != "table" && app.Output != "json" {
			return Usagef("invalid --output %q (want table or json)", app.Output)
		}
		baseURL := strings.TrimRight(v.GetString("base_url"), "/")
		client, err := apigen.NewClient(baseURL, apiKeySource{key: v.GetString("api_key")})
		if err != nil {
			return fmt.Errorf("configuring API client: %w", err)
		}
		app.API = client
		return nil
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the gphq CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "gphq version %s\n", version)
		},
	})

	addAll(root, app)
	root.AddCommand(newDocsCmd(root))
	return root
}

// addAll registers every resource-group command on the root.
func addAll(root *cobra.Command, app *App) {
	root.AddCommand(
		newHealthCmd(app),
		newApiKeysCmd(app),
		newCartsCmd(app),
		newCustomersCmd(app),
		newDunningCmd(app),
		newGatewaysCmd(app),
		newInvoicesCmd(app),
		newMetersCmd(app),
		newOrdersCmd(app),
		newOrgsCmd(app),
		newPaymentMethodsCmd(app),
		newPaymentTokensCmd(app),
		newPaymentsCmd(app),
		newPricesCmd(app),
		newProductsCmd(app),
		newRemindersCmd(app),
		newSessionsCmd(app),
		newSettingsCmd(app),
		newSubscriptionsCmd(app),
		newUsageCmd(app),
		newVariantsCmd(app),
		newWebhooksCmd(app),
	)
}
