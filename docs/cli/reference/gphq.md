## gphq

Command-line client for the GetPaidHQ API

### Synopsis

gphq is the command-line client for the GetPaidHQ subscription-billing API.

Authentication uses an organization API key sent as the x-api-key header.
Configuration precedence: flags > GPHQ_* environment variables >
~/.config/gphq/config.toml > defaults.

### Options

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -h, --help              help for gphq
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq api-keys](gphq_api-keys.md)	 - Manage API keys
* [gphq carts](gphq_carts.md)	 - Manage carts
* [gphq completion](gphq_completion.md)	 - Generate the autocompletion script for the specified shell
* [gphq customers](gphq_customers.md)	 - Manage customers
* [gphq dunning](gphq_dunning.md)	 - Manage dunning campaigns and configurations
* [gphq gateways](gphq_gateways.md)	 - Manage payment gateways
* [gphq health](gphq_health.md)	 - Check API server health
* [gphq invoices](gphq_invoices.md)	 - Inspect invoices
* [gphq meters](gphq_meters.md)	 - Manage billable meters
* [gphq orders](gphq_orders.md)	 - Manage orders
* [gphq orgs](gphq_orgs.md)	 - Manage organizations
* [gphq payment-methods](gphq_payment-methods.md)	 - Inspect payment methods
* [gphq payment-tokens](gphq_payment-tokens.md)	 - Manage payment update tokens
* [gphq payments](gphq_payments.md)	 - Inspect payments
* [gphq prices](gphq_prices.md)	 - Manage prices
* [gphq products](gphq_products.md)	 - Manage products
* [gphq reminders](gphq_reminders.md)	 - Manage renewal reminder config
* [gphq sessions](gphq_sessions.md)	 - Manage sessions
* [gphq settings](gphq_settings.md)	 - Manage org settings
* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions
* [gphq usage](gphq_usage.md)	 - Ingest usage events
* [gphq variants](gphq_variants.md)	 - Manage variants
* [gphq version](gphq_version.md)	 - Print the gphq CLI version
* [gphq webhooks](gphq_webhooks.md)	 - Manage webhook subscriptions

