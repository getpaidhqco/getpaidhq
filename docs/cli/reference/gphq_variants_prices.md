## gphq variants prices

List prices of a variant

### Synopsis

List all prices for a variant. Returns a paginated {data,meta} envelope.

```
gphq variants prices <variantId> [flags]
```

### Examples

```
  gphq variants prices var_1
```

### Options

```
  -h, --help                help for prices
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

* [gphq variants](gphq_variants.md)	 - Manage variants

