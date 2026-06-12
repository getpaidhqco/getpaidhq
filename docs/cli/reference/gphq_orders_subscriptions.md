## gphq orders subscriptions

List subscriptions for an order

### Synopsis

Fetch all subscriptions attached to an order. The response is a plain JSON array.

```
gphq orders subscriptions <orderId> [flags]
```

### Examples

```
  gphq orders subscriptions ord_1
```

### Options

```
  -h, --help   help for subscriptions
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq orders](gphq_orders.md)	 - Manage orders

