package commands

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	api "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/cli/output"
)

// ---------------------------------------------------------------------------
// api-keys
// ---------------------------------------------------------------------------

var apiKeyHeaders = []string{"ID", "NAME", "KEY", "CREATED"}

func apiKeyCreateRow(k api.ApiKeyCreateResponse) []string {
	return []string{
		k.Id,
		output.Str(k.Name),
		k.Key,
		output.Time(k.CreatedAt),
	}
}

var apiKeyListHeaders = []string{"ID", "NAME", "CREATED"}

func apiKeyListRow(k api.ApiKeyResponse) []string {
	return []string{
		k.Id,
		output.Str(k.Name),
		output.Time(k.CreatedAt),
	}
}

func newApiKeysCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-keys",
		Short: "Manage API keys",
		Long:  "Create, list, and revoke org-scoped API keys.",
	}
	cmd.AddCommand(
		newApiKeysCreateCmd(app),
		newApiKeysListCmd(app),
		newApiKeysRevokeCmd(app),
	)
	return cmd
}

func newApiKeysCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an API key",
		Long:    "Create a new API key. The plaintext key is returned once and never shown again.",
		Example: "  gphq api-keys create --name ci-deploy\n  gphq api-keys create --data '{\"name\":\"my-key\"}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				return api.CreateApiKeyInput{Name: name}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/api-keys", nil, body)
			if err != nil {
				return err
			}
			// Always print the one-time note to stderr — it never pollutes stdout
			// and still reaches the operator even in json mode where scripts pipe stdout.
			fmt.Fprintln(app.ErrOut, "note: store this key now — it is shown only once")
			return renderOne(app, raw, apiKeyHeaders, apiKeyCreateRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "human-readable label for the key (optional)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/api-keys")
}

func newApiKeysListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List API keys",
		Long:    "List all API keys for the organization.",
		Example: "  gphq api-keys list",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/api-keys", listQuery(cmd), nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, apiKeyListHeaders, apiKeyListRow)
		},
	}
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/api-keys")
}

func newApiKeysRevokeCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revoke <id>",
		Short:   "Revoke an API key",
		Long:    "Permanently revoke an API key by ID.",
		Example: "  gphq api-keys revoke key_abc123",
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := app.Client.Do(cmd.Context(), http.MethodDelete, "/api/api-keys/"+args[0], nil, nil)
			if err != nil {
				return err
			}
			if app.Output != "json" {
				_, err = fmt.Fprintf(app.Out, "%s revoked\n", args[0])
			}
			return err
		},
	}
	return annotate(cmd, "DELETE", "/api/api-keys/{id}")
}

// ---------------------------------------------------------------------------
// orgs
// ---------------------------------------------------------------------------

func newOrgsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orgs",
		Short: "Manage organizations",
		Long:  "Create and manage organizations.",
	}
	cmd.AddCommand(
		newOrgsCreateCmd(app),
	)
	return cmd
}

func newOrgsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an organization",
		Long:    "Create a new organization. Name, country, and timezone are required.",
		Example: "  gphq orgs create --name \"Acme Corp\" --country NG --timezone Africa/Lagos\n  gphq orgs create --data '{\"name\":\"Acme\",\"country\":\"NG\",\"timezone\":\"UTC\"}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				country, _ := cmd.Flags().GetString("country")
				timezone, _ := cmd.Flags().GetString("timezone")
				if name == "" || country == "" || timezone == "" {
					return nil, Usagef("--name, --country, and --timezone are required (or use --data)")
				}
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return nil, err
				}
				return api.CreateOrgRequest{
					Name:     name,
					Country:  country,
					Timezone: timezone,
					Metadata: meta,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/organizations", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "organization name (required)")
	f.String("country", "", "ISO 3166-1 alpha-2 country code (required)")
	f.String("timezone", "", "IANA timezone name, e.g. Africa/Lagos (required)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/organizations")
}

// ---------------------------------------------------------------------------
// gateways
// ---------------------------------------------------------------------------

func newGatewaysCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateways",
		Short: "Manage payment gateways",
		Long:  "Configure payment service provider (PSP) gateways for the organization.",
	}
	cmd.AddCommand(
		newGatewaysCreateCmd(app),
	)
	return cmd
}

func newGatewaysCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Configure a payment gateway",
		Long:    "Configure a new payment service provider gateway. At least one --credential is required.",
		Example: "  gphq gateways create --name prod-paystack --psp paystack --credential secret_key=sk_live_x\n  gphq gateways create --data '{\"name\":\"prod\",\"psp\":\"paystack\",\"credentials\":{\"secret_key\":\"sk_live_x\"}}'",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				name, _ := cmd.Flags().GetString("name")
				psp, _ := cmd.Flags().GetString("psp")
				if name == "" || psp == "" {
					return nil, Usagef("--name and --psp are required (or use --data)")
				}
				credPairs, _ := cmd.Flags().GetStringArray("credential")
				if len(credPairs) == 0 {
					return nil, Usagef("at least one --credential key=value is required (or use --data)")
				}
				creds, err := parseKV(credPairs, "credential")
				if err != nil {
					return nil, err
				}
				configPairs, _ := cmd.Flags().GetStringArray("config")
				cfg, err := parseKV(configPairs, "config")
				if err != nil {
					return nil, err
				}

				// SECURITY: We must NOT use api.CreateGatewayRequest here.
				// That struct has Credentials as map[string]domain.Secret, and
				// domain.Secret redacts itself to "[REDACTED]" when JSON-marshaled.
				// Using the typed struct would send literal "[REDACTED]" to the
				// server instead of the real credential values. We build a plain
				// map so the raw string values pass through unmodified.
				return map[string]any{
					"name":        name,
					"psp":         psp,
					"config":      cfg,
					"credentials": creds,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/gateways", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "human-readable gateway name (required)")
	f.String("psp", "", "payment service provider id, e.g. paystack, checkout_com (required)")
	f.StringArray("config", nil, "non-secret config key=value pairs (repeatable)")
	f.StringArray("credential", nil, "secret credential key=value pairs (repeatable, required)")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/gateways")
}

// ---------------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------------

var settingHeaders = []string{"PARENT", "ID", "TYPE", "VALUE", "CREATED"}

func settingRow(s api.SettingResponse) []string {
	return []string{
		output.Str(s.ParentId),
		s.Id,
		output.Str(s.Type),
		output.Str(s.Value),
		output.Time(s.CreatedAt),
	}
}

func newSettingsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage org settings",
		Long:  "Create, list, get, update, and delete org-scoped key/value settings.",
	}
	cmd.AddCommand(
		newSettingsCreateCmd(app),
		newSettingsListCmd(app),
		newSettingsGetCmd(app),
		newSettingsUpdateCmd(app),
		newSettingsDeleteCmd(app),
	)
	return cmd
}

func newSettingsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a setting",
		Long:    "Create a new org setting. The --id flag is required.",
		Example: "  gphq settings create --id theme --value dark\n  gphq settings create --parent ui --id color --type string --value blue",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				id, _ := cmd.Flags().GetString("id")
				if id == "" {
					return nil, Usagef("--id is required (or use --data)")
				}
				parent, _ := cmd.Flags().GetString("parent")
				typ, _ := cmd.Flags().GetString("type")
				val, _ := cmd.Flags().GetString("value")
				return api.CreateSettingRequest{
					ParentId: parent,
					Id:       id,
					Type:     typ,
					Value:    val,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/settings", nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, settingHeaders, settingRow)
		},
	}
	f := cmd.Flags()
	f.String("parent", "", "parent namespace id (optional)")
	f.String("id", "", "setting id (required)")
	f.String("type", "", "value type hint, e.g. string, json")
	f.String("value", "", "setting value")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/settings")
}

func newSettingsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List settings",
		Long:    "List org settings. Pass --parent to filter by parent namespace.",
		Example: "  gphq settings list\n  gphq settings list --parent ui",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			q := listQuery(cmd)
			parent, _ := cmd.Flags().GetString("parent")
			if parent != "" {
				if q == nil {
					q = make(url.Values)
				}
				q.Set("parent_id", parent)
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/settings", q, nil)
			if err != nil {
				return err
			}
			return renderList(app, raw, settingHeaders, settingRow)
		},
	}
	f := cmd.Flags()
	f.String("parent", "", "filter by parent namespace id")
	addListFlags(cmd)
	return annotate(cmd, "GET", "/api/settings")
}

func newSettingsGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <parentId> <id>",
		Short:   "Get a setting",
		Long:    "Fetch a single setting by parent ID and setting ID.",
		Example: "  gphq settings get ui color",
		Args:    exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/api/settings/%s/%s", args[0], args[1])
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return err
			}
			return renderOne(app, raw, settingHeaders, settingRow)
		},
	}
	return annotate(cmd, "GET", "/api/settings/{parentId}/{id}")
}

func newSettingsUpdateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <parentId> <id>",
		Short:   "Update (upsert) a setting",
		Long:    "Create or replace a setting at the given parent and id.",
		Example: "  gphq settings update ui color --value red\n  gphq settings update ui color --data '{\"value\":\"red\"}'",
		Args:    exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				typ, _ := cmd.Flags().GetString("type")
				val, _ := cmd.Flags().GetString("value")
				return api.UpdateSettingRequest{
					Type:  typ,
					Value: val,
				}, nil
			})
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/settings/%s/%s", args[0], args[1])
			raw, err := app.Client.Do(cmd.Context(), http.MethodPut, path, nil, body)
			if err != nil {
				return err
			}
			return renderOne(app, raw, settingHeaders, settingRow)
		},
	}
	f := cmd.Flags()
	f.String("type", "", "value type hint")
	f.String("value", "", "setting value")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "PUT", "/api/settings/{parentId}/{id}")
}

func newSettingsDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <parentId> <id>",
		Short:   "Delete a setting",
		Long:    "Permanently delete a setting by parent ID and setting ID.",
		Example: "  gphq settings delete ui color",
		Args:    exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/api/settings/%s/%s", args[0], args[1])
			_, err := app.Client.Do(cmd.Context(), http.MethodDelete, path, nil, nil)
			if err != nil {
				return err
			}
			return renderDeleted(app, args[0]+"/"+args[1])
		},
	}
	return annotate(cmd, "DELETE", "/api/settings/{parentId}/{id}")
}

// ---------------------------------------------------------------------------
// webhooks
// ---------------------------------------------------------------------------

func newWebhooksCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Manage webhook subscriptions",
		Long:  "Create and list webhook subscriptions for the organization.",
	}
	cmd.AddCommand(
		newWebhooksCreateCmd(app),
		newWebhooksListCmd(app),
	)
	return cmd
}

func newWebhooksCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a webhook subscription",
		Long:    "Subscribe an endpoint URL to one or more event types.",
		Example: "  gphq webhooks create --url https://example.com/hook --event subscription.created --event payment.succeeded",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := bodyOrData(cmd, func() (any, error) {
				urlVal, _ := cmd.Flags().GetString("url")
				events, _ := cmd.Flags().GetStringArray("event")
				if urlVal == "" {
					return nil, Usagef("--url is required (or use --data)")
				}
				if len(events) == 0 {
					return nil, Usagef("at least one --event is required (or use --data)")
				}
				secret, _ := cmd.Flags().GetString("secret")
				return api.CreateWebhookSubscriptionRequest{
					Url:    urlVal,
					Events: events,
					Secret: secret,
				}, nil
			})
			if err != nil {
				return err
			}
			raw, err := app.Client.Do(cmd.Context(), http.MethodPost, "/api/webhooks", nil, body)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	f := cmd.Flags()
	f.String("url", "", "webhook endpoint URL (required)")
	f.StringArray("event", nil, "event type to subscribe to (repeatable, required)")
	f.String("secret", "", "optional signing secret")
	f.String("data", "", "raw JSON body (@file, -, or inline)")
	return annotate(cmd, "POST", "/api/webhooks")
}

func newWebhooksListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List webhook subscriptions",
		Long:    "List all webhook subscriptions for the organization.",
		Example: "  gphq webhooks list",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := app.Client.Do(cmd.Context(), http.MethodGet, "/api/webhooks", nil, nil)
			if err != nil {
				return err
			}
			return renderJSON(app, raw)
		},
	}
	return annotate(cmd, "GET", "/api/webhooks")
}
