## gphq orders complete

Complete an order

### Synopsis

Mark an order as complete, optionally providing payment method details.

```
gphq orders complete <orderId> [flags]
```

### Examples

```
  gphq orders complete ord_1 --payment-method pm_1
  gphq orders complete ord_1 --data -
```

### Options

```
      --data string             raw JSON body (@file, -, or inline)
  -h, --help                    help for complete
      --payment-method string   payment method ID to use for completion
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq orders](gphq_orders.md)	 - Manage orders

