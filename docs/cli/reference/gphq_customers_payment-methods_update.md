## gphq customers payment-methods update

Update a customer payment method

### Synopsis

Update a saved payment method. Unset flags are sent as empty values, which the server ignores.

```
gphq customers payment-methods update <customerId> <paymentMethodId> [flags]
```

### Examples

```
  gphq customers payment-methods update cus_1 pm_1 --name "Updated Card"
```

### Options

```
      --data string    raw JSON body (@file, -, or inline)
      --default        set as default payment method
  -h, --help           help for update
      --name string    display name
      --token string   PSP charge token
      --type string    payment method type
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq customers payment-methods](gphq_customers_payment-methods.md)	 - Manage customer payment methods

