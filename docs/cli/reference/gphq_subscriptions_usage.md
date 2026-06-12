## gphq subscriptions usage

Get current-period usage for a subscription

### Synopsis

Fetch metered usage for a subscription's current billing period.

```
gphq subscriptions usage <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions usage sub_1
```

### Options

```
  -h, --help   help for usage
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

