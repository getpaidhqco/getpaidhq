## gphq payment-tokens create

Create a payment update token (admin)

### Synopsis

Admin: create a payment update token for a subscription.

The token can be sent to customers as part of a dunning recovery flow to allow
them to update their payment method without logging in.

```
gphq payment-tokens create <subscriptionId> [flags]
```

### Examples

```
  gphq payment-tokens create sub_1 --max-uses 3 --expiry-hours 48 --reason "proactive retry"
  gphq payment-tokens create sub_1 --data '{"max_uses":1,"expiry_hours":24}'
```

### Options

```
      --data string        raw JSON body (@file, -, or inline); use this to set allowed_actions
      --expiry-hours int   hours until token expiry (0 = default server expiry)
  -h, --help               help for create
      --max-uses int       maximum number of times the token can be used (0 = unlimited)
      --notes string       admin notes (admin_notes)
      --reason string      admin reason for creating the token (admin_reason)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq payment-tokens](gphq_payment-tokens.md)	 - Manage payment update tokens

