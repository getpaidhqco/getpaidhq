## gphq api-keys list

List API keys

### Synopsis

List all API keys for the organization.

```
gphq api-keys list [flags]
```

### Examples

```
  gphq api-keys list
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

* [gphq api-keys](gphq_api-keys.md)	 - Manage API keys

