## gphq subscriptions billing-anchor

Update subscription billing anchor

### Synopsis

Update the billing anchor day (1-31) for a subscription.

```
gphq subscriptions billing-anchor <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions billing-anchor sub_1 --anchor 15 --proration prorate
```

### Options

```
      --anchor int         billing anchor day 1-31 (required)
      --data string        raw JSON body (@file, -, or inline)
  -h, --help               help for billing-anchor
      --proration string   proration mode: none or prorate (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

