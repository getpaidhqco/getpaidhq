## gphq subscriptions resume

Resume a subscription

### Synopsis

Resume a paused subscription. Use --behavior to control billing period behavior.

```
gphq subscriptions resume <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions resume sub_1 --behavior start_new_billing_period
```

### Options

```
      --behavior string   resume behavior: continue_existing_billing_period or start_new_billing_period
      --data string       raw JSON body (@file, -, or inline)
  -h, --help              help for resume
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

