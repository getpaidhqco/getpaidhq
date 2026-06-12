## gphq subscriptions cancel

Cancel a subscription

### Synopsis

Cancel an active subscription, optionally providing a reason.

```
gphq subscriptions cancel <subscriptionId> [flags]
```

### Examples

```
  gphq subscriptions cancel sub_1 --reason "non-payment"
```

### Options

```
      --data string     raw JSON body (@file, -, or inline)
  -h, --help            help for cancel
      --reason string   reason for cancellation
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq subscriptions](gphq_subscriptions.md)	 - Manage subscriptions

