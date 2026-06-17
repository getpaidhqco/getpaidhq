## gphq webhooks create

Create a webhook subscription

### Synopsis

Subscribe an endpoint URL to one or more event types.

```
gphq webhooks create [flags]
```

### Examples

```
  gphq webhooks create --url https://example.com/hook --event subscription.created --event payment.succeeded
```

### Options

```
      --data string         raw JSON body (@file, -, or inline)
      --event stringArray   event type to subscribe to (repeatable, required)
  -h, --help                help for create
      --secret string       optional signing secret
      --url string          webhook endpoint URL (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq webhooks](gphq_webhooks.md)	 - Manage webhook subscriptions

