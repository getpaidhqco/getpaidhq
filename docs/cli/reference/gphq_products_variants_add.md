## gphq products variants add

Add a variant to a product

### Synopsis

Create a new variant under an existing product.

```
gphq products variants add <productId> [flags]
```

### Examples

```
  gphq products variants add prod_1 --name Premium
  gphq products variants add prod_1 --data @variant.json
```

### Options

```
      --data string            raw JSON body (@file, -, or inline)
      --description string     variant description
  -h, --help                   help for add
      --metadata stringArray   metadata key=value pairs (repeatable)
      --name string            variant name (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq products variants](gphq_products_variants.md)	 - Manage product variants

