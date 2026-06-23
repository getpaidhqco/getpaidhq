package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

// ---------------------------------------------------------------------------
// api-keys
// ---------------------------------------------------------------------------

var apiKeyHeaders = []string{"ID", "NAME", "KEY", "CREATED"}

func apiKeyCreateRow(k apigen.ApiKeyCreateResponse) []string {
	return []string{
		k.ID.Or(""),
		output.Str(k.Name.Or("")),
		k.Key.Or(""),
		output.Time(k.CreatedAt.Or(time.Time{})),
	}
}

var apiKeyListHeaders = []string{"ID", "NAME", "CREATED"}

func apiKeyListRow(k apigen.ApiKeyCreateResponse) []string {
	return []string{
		k.ID.Or(""),
		output.Str(k.Name.Or("")),
		output.Time(k.CreatedAt.Or(time.Time{})),
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
			body, err := bindBody(cmd, func(in *apigen.CreateApiKeyInput) error {
				if s, _ := cmd.Flags().GetString("name"); s != "" {
					in.Name = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateApiKey(cmd.Context(), body, apigen.CreateApiKeyParams{})
			key, err := expectOK[*apigen.ApiKeyCreateResponse](res, err)
			if err != nil {
				return err
			}
			// Always print the one-time note to stderr — it never pollutes stdout
			// and still reaches the operator even in json mode where scripts pipe stdout.
			fmt.Fprintln(app.ErrOut, "note: store this key now — it is shown only once")
			return renderOne(app, *key, apiKeyHeaders, apiKeyCreateRow)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "human-readable label for the key (optional)")
	addDataFlag(cmd)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListApiKeys(cmd.Context(), apigen.ListApiKeysParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, apiKeyListHeaders, apiKeyListRow)
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
			res, err := app.API.DeleteApiKey(cmd.Context(), apigen.DeleteApiKeyParams{ID: args[0]})
			if _, err := expectOK[*apigen.EmptyResponse](res, err); err != nil {
				return err
			}
			return renderDeleted(app, "api-key "+args[0])
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
			body, err := bindBody(cmd, func(in *apigen.CreateOrgRequest) error {
				name, _ := cmd.Flags().GetString("name")
				country, _ := cmd.Flags().GetString("country")
				timezone, _ := cmd.Flags().GetString("timezone")
				if name == "" || country == "" || timezone == "" {
					return Usagef("--name, --country, and --timezone are required (or use --data)")
				}
				in.Name = name
				in.Country = country
				in.Timezone = timezone
				metaPairs, _ := cmd.Flags().GetStringArray("metadata")
				meta, err := parseKV(metaPairs, "metadata")
				if err != nil {
					return err
				}
				if meta != nil {
					in.Metadata = apigen.NewOptCreateOrgRequestMetadata(apigen.CreateOrgRequestMetadata(meta))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateOrganization(cmd.Context(), body, apigen.CreateOrganizationParams{})
			org, err := expectOK[*apigen.OrgResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, org)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "organization name (required)")
	f.String("country", "", "ISO 3166-1 alpha-2 country code (required)")
	f.String("timezone", "", "IANA timezone name, e.g. Africa/Lagos (required)")
	f.StringArray("metadata", nil, "metadata key=value pairs (repeatable)")
	addDataFlag(cmd)
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
			body, err := bindBody(cmd, func(in *apigen.CreateGatewayRequest) error {
				name, _ := cmd.Flags().GetString("name")
				psp, _ := cmd.Flags().GetString("psp")
				if name == "" || psp == "" {
					return Usagef("--name and --psp are required (or use --data)")
				}
				credPairs, _ := cmd.Flags().GetStringArray("credential")
				if len(credPairs) == 0 {
					return Usagef("at least one --credential key=value is required (or use --data)")
				}
				creds, err := parseKV(credPairs, "credential")
				if err != nil {
					return err
				}
				configPairs, _ := cmd.Flags().GetStringArray("config")
				cfg, err := parseKV(configPairs, "config")
				if err != nil {
					return err
				}
				in.Name = name
				in.Psp = psp
				in.Credentials = apigen.CreateGatewayRequestCredentials(creds)
				if cfg != nil {
					in.Config = apigen.NewOptCreateGatewayRequestConfig(apigen.CreateGatewayRequestConfig(cfg))
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateGateway(cmd.Context(), body, apigen.CreateGatewayParams{})
			gw, err := expectOK[*apigen.GatewayResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, gw)
		},
	}
	f := cmd.Flags()
	f.String("name", "", "human-readable gateway name (required)")
	f.String("psp", "", "payment service provider id, e.g. paystack, checkout_com (required)")
	f.StringArray("config", nil, "non-secret config key=value pairs (repeatable)")
	f.StringArray("credential", nil, "secret credential key=value pairs (repeatable, required)")
	addDataFlag(cmd)
	return annotate(cmd, "POST", "/api/gateways")
}

// ---------------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------------

var settingHeaders = []string{"PARENT", "ID", "TYPE", "VALUE", "CREATED"}

func settingRow(s apigen.SettingResponse) []string {
	return []string{
		output.Str(s.ParentID.Or("")),
		s.ID.Or(""),
		output.Str(s.Type.Or("")),
		output.Str(s.Value.Or("")),
		output.Time(s.CreatedAt.Or(time.Time{})),
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
			body, err := bindBody(cmd, func(in *apigen.CreateSettingRequest) error {
				id, _ := cmd.Flags().GetString("id")
				if id == "" {
					return Usagef("--id is required (or use --data)")
				}
				in.ID = id
				if s, _ := cmd.Flags().GetString("parent"); s != "" {
					in.ParentID = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("type"); s != "" {
					in.Type = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("value"); s != "" {
					in.Value = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateSetting(cmd.Context(), body, apigen.CreateSettingParams{})
			s, err := expectOK[*apigen.SettingResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *s, settingHeaders, settingRow)
		},
	}
	f := cmd.Flags()
	f.String("parent", "", "parent namespace id (optional)")
	f.String("id", "", "setting id (required)")
	f.String("type", "", "value type hint, e.g. string, json")
	f.String("value", "", "setting value")
	addDataFlag(cmd)
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
			page, limit, sortBy, sortOrder := listArgs(cmd)
			res, err := app.API.ListSettings(cmd.Context(), apigen.ListSettingsParams{
				Page:      apigen.NewOptInt(page),
				Limit:     apigen.NewOptInt(limit),
				SortBy:    apigen.NewOptString(sortBy),
				SortOrder: apigen.NewOptString(sortOrder),
			})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderList(app, lr, settingHeaders, settingRow)
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
			res, err := app.API.GetSetting(cmd.Context(), apigen.GetSettingParams{ParentId: args[0], ID: args[1]})
			s, err := expectOK[*apigen.SettingResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *s, settingHeaders, settingRow)
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
			body, err := bindBody(cmd, func(in *apigen.UpdateSettingRequest) error {
				if s, _ := cmd.Flags().GetString("type"); s != "" {
					in.Type = apigen.NewOptString(s)
				}
				if s, _ := cmd.Flags().GetString("value"); s != "" {
					in.Value = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.UpdateSetting(cmd.Context(), body, apigen.UpdateSettingParams{ParentId: args[0], ID: args[1]})
			s, err := expectOK[*apigen.SettingResponse](res, err)
			if err != nil {
				return err
			}
			return renderOne(app, *s, settingHeaders, settingRow)
		},
	}
	f := cmd.Flags()
	f.String("type", "", "value type hint")
	f.String("value", "", "setting value")
	addDataFlag(cmd)
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
			res, err := app.API.DeleteSetting(cmd.Context(), apigen.DeleteSettingParams{ParentId: args[0], ID: args[1]})
			if _, err := expectOK[*apigen.EmptyResponse](res, err); err != nil {
				return err
			}
			return renderDeleted(app, "setting "+args[0]+"/"+args[1])
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
			body, err := bindBody(cmd, func(in *apigen.CreateWebhookSubscriptionRequest) error {
				urlVal, _ := cmd.Flags().GetString("url")
				events, _ := cmd.Flags().GetStringArray("event")
				if urlVal == "" {
					return Usagef("--url is required (or use --data)")
				}
				if len(events) == 0 {
					return Usagef("at least one --event is required (or use --data)")
				}
				in.URL = urlVal
				in.Events = events
				if s, _ := cmd.Flags().GetString("secret"); s != "" {
					in.Secret = apigen.NewOptString(s)
				}
				return nil
			})
			if err != nil {
				return err
			}
			res, err := app.API.CreateWebhookSubscription(cmd.Context(), body, apigen.CreateWebhookSubscriptionParams{})
			sub, err := expectOK[*apigen.UnknownInterface](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, sub)
		},
	}
	f := cmd.Flags()
	f.String("url", "", "webhook endpoint URL (required)")
	f.StringArray("event", nil, "event type to subscribe to (repeatable, required)")
	f.String("secret", "", "optional signing secret")
	addDataFlag(cmd)
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
			res, err := app.API.ListWebhookSubscriptions(cmd.Context(), apigen.ListWebhookSubscriptionsParams{})
			lr, err := expectOK[*apigen.ListResponse](res, err)
			if err != nil {
				return err
			}
			return renderValue(app, lr)
		},
	}
	return annotate(cmd, "GET", "/api/webhooks")
}
