## gphq payment-methods get

Get a payment method

### Synopsis

Fetch a single payment method by ID.

```
gphq payment-methods get <paymentMethodId> [flags]
```

### Examples

```
  gphq payment-methods get pm_abc123
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq payment-methods](gphq_payment-methods.md)	 - Inspect payment methods

