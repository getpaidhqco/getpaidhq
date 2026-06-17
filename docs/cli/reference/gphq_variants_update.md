## gphq variants update

Update a variant

### Synopsis

Update a variant's name, description, or metadata.

```
gphq variants update <variantId> [flags]
```

### Examples

```
  gphq variants update var_1 --name "Premium v2"
  gphq variants update var_1 --data @variant.json
```

### Options

```
      --data string            raw JSON body (@file, -, or inline)
      --description string     variant description
  -h, --help                   help for update
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

* [gphq variants](gphq_variants.md)	 - Manage variants

