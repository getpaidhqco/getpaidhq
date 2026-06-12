## gphq orders create

Create an order

### Synopsis

Create a new order. Pass flags for common fields or --data for a raw JSON body.

```
gphq orders create [flags]
```

### Examples

```
  gphq orders create --customer-id cus_1 --psp paystack --currency NGN --item product=prod_1,price=pri_1
  gphq orders create --data '{"psp_id":"paystack","customer":{"id":"cus_1"}}'
```

### Options

```
      --currency string         cart currency
      --customer-id string      customer ID
      --data string             raw JSON body (@file, -, or inline)
      --email string            customer email
      --first-name string       customer first name
  -h, --help                    help for create
      --item stringArray        cart item: product=<id>,price=<id>[,qty=<n>] (repeatable)
      --last-name string        customer last name
      --metadata stringArray    metadata key=value pairs (repeatable)
      --payment-method string   payment method ID
      --phone string            customer phone
      --psp string              payment service provider ID (required)
      --session string          session ID
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq orders](gphq_orders.md)	 - Manage orders

