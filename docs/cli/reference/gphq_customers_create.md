## gphq customers create

Create a customer

### Synopsis

Create a new customer. Pass flags for common fields or --data for a raw JSON body.

```
gphq customers create [flags]
```

### Examples

```
  gphq customers create --email ada@example.com --first-name Ada
  gphq customers create --data '{"email":"ada@example.com"}'
```

### Options

```
      --data string            raw JSON body (@file, -, or inline)
      --email string           customer email address (required)
      --first-name string      first name
  -h, --help                   help for create
      --last-name string       last name
      --metadata stringArray   metadata key=value pairs (repeatable)
      --phone string           phone number
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq customers](gphq_customers.md)	 - Manage customers

