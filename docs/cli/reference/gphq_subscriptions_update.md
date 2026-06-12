## gphq subscriptions update

Update a subscription

### Synopsis

Update subscription status, default payment method, or metadata. Unset flags are sent as zero values.

```
gphq subscriptions update <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions update sub_1 --status paused
  gphq subscriptions update sub_1 --metadata key=value
```

### Options

```
      --data string                     raw JSON body (@file, -, or inline)
      --default-payment-method string   default payment method ID
  -h, --help                            help for update
      --metadata stringArray            metadata key=value pairs (repeatable)
      --status string                   subscription status
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

