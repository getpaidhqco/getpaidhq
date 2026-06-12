## gphq customers dunning-history

Get a customer's dunning history

### Synopsis

Fetch the dunning campaign history for a customer.

```
gphq customers dunning-history <customerId> [flags]
```

### Examples

```
  gphq customers dunning-history cus_1
```

### Options

```
  -h, --help   help for dunning-history
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq customers](gphq_customers.md)	 - Manage customers

