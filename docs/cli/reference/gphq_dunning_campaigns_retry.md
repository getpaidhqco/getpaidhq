## gphq dunning campaigns retry

Trigger a manual retry attempt

### Synopsis

Trigger a manual dunning retry attempt for a campaign.

Optionally pass --payment-method to specify the payment method ID to charge.
Use --data to send a raw JSON body instead of flags.

```
gphq dunning campaigns retry <id> [flags]
```

### Examples

```
  gphq dunning campaigns retry dc_1
  gphq dunning campaigns retry dc_1 --payment-method pm_abc
  gphq dunning campaigns retry dc_1 --data '{"payment_method_id":"pm_abc"}'
```

### Options

```
      --data string             raw JSON body (@file, -, or inline)
  -h, --help                    help for retry
      --payment-method string   payment method ID to charge (optional)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq dunning campaigns](gphq_dunning_campaigns.md)	 - Manage dunning campaigns

