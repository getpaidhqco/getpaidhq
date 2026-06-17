## gphq products create

Create a product

### Synopsis

Create a new product. The API requires at least one variant — use --data with a complete JSON body.
Flag-only creates (--name only) will be rejected by the server unless you provide variants via --data.

```
gphq products create [flags]
```

### Examples

```
  gphq products create --name "Acme Pro" --description "My product"
  gphq products create --data @product.json
```

### Options

```
      --data string            raw JSON body (@file, -, or inline)
      --description string     product description
  -h, --help                   help for create
      --metadata stringArray   metadata key=value pairs (repeatable)
      --name string            product name
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq products](gphq_products.md)	 - Manage products

