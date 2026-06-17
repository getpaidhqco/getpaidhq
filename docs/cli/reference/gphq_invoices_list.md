## gphq invoices list

List invoices

### Synopsis

List all invoices for the organization with optional pagination.

```
gphq invoices list [flags]
```

### Examples

```
  gphq invoices list
  gphq invoices list --page 2 --limit 5
```

### Options

```
  -h, --help                help for list
      --limit int           items per page (default 10)
      --page int            page number (zero-indexed)
      --sort-by string      sort field (default "created_at")
      --sort-order string   asc or desc (default "desc")
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq invoices](gphq_invoices.md)	 - Inspect invoices

