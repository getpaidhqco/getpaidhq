## gphq products update

Update a product

### Synopsis

Update a product's name, description, or metadata.

```
gphq products update <id> [flags]
```

### Examples

```
  gphq products update prod_1 --name "New Name"
  gphq products update prod_1 --data '{"name":"New Name"}'
```

### Options

```
      --data string            raw JSON body (@file, -, or inline)
      --description string     product description
  -h, --help                   help for update
      --metadata stringArray   metadata key=value pairs (repeatable)
      --name string            product name (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq products](gphq_products.md)	 - Manage products

