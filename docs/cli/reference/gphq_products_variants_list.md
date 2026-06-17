## gphq products variants list

List variants of a product

### Synopsis

List all variants belonging to a product. Returns a paginated {data,meta} envelope.

```
gphq products variants list <productId> [flags]
```

### Examples

```
  gphq products variants list prod_1
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

* [gphq products variants](gphq_products_variants.md)	 - Manage product variants

