## gphq subscriptions pause

Pause a subscription

### Synopsis

Pause an active subscription, optionally providing a reason.

```
gphq subscriptions pause <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions pause sub_1 --reason "customer request"
```

### Options

```
      --data string     raw JSON body (@file, -, or inline)
  -h, --help            help for pause
      --reason string   reason for pausing
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

