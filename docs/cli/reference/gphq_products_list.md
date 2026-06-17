## gphq products list

List products

### Synopsis

List products with optional pagination and status filter.

```
gphq products list [flags]
```

### Examples

```
  gphq products list
  gphq products list --status archived
  gphq products list --status all --page 2
```

### Options

```
  -h, --help                help for list
      --limit int           items per page (default 10)
      --page int            page number (zero-indexed)
      --sort-by string      sort field (default "created_at")
      --sort-order string   asc or desc (default "desc")
      --status string       filter by status: active, archived, or all (default: active)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq products](gphq_products.md)	 - Manage products

