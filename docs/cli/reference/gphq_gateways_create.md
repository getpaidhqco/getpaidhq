## gphq gateways create

Configure a payment gateway

### Synopsis

Configure a new payment service provider gateway. At least one --credential is required.

```
gphq gateways create [flags]
```

### Examples

```
  gphq gateways create --name prod-paystack --psp paystack --credential secret_key=sk_live_x
  gphq gateways create --data '{"name":"prod","psp":"paystack","credentials":{"secret_key":"sk_live_x"}}'
```

### Options

```
      --config stringArray       non-secret config key=value pairs (repeatable)
      --credential stringArray   secret credential key=value pairs (repeatable, required)
      --data string              raw JSON body (@file, -, or inline)
  -h, --help                     help for create
      --name string              human-readable gateway name (required)
      --psp string               payment service provider id, e.g. paystack, checkout_com (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq gateways](gphq_gateways.md)	 - Manage payment gateways

