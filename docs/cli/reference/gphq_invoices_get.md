## gphq invoices get

Get an invoice

### Synopsis

Fetch a single invoice by ID.

```
gphq invoices get <invoiceId> [flags]
```

### Examples

```
  gphq invoices get inv_abc123
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

* [gphq invoices](gphq_invoices.md)	 - Inspect invoices

