## gphq customers payment-methods add

Add a payment method to a customer

### Synopsis

Attach a PSP payment token to a customer as a saved payment method.

```
gphq customers payment-methods add <customerId> [flags]
```

### Examples

```
  gphq customers payment-methods add cus_1 --psp paystack --name "My Card" --type card --token tok_abc
```

### Options

```
      --data string    raw JSON body (@file, -, or inline)
      --default        set as default payment method
  -h, --help           help for add
      --name string    display name for the payment method (required)
      --psp string     payment service provider (required)
      --token string   PSP charge token (required)
      --type string    payment method type, e.g. card (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq customers payment-methods](gphq_customers_payment-methods.md)	 - Manage customer payment methods

