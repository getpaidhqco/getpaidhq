## gphq settings list

List settings

### Synopsis

List org settings. Pass --parent to filter by parent namespace.

```
gphq settings list [flags]
```

### Examples

```
  gphq settings list
  gphq settings list --parent ui
```

### Options

```
  -h, --help                help for list
      --limit int           items per page (default 10)
      --page int            page number (zero-indexed)
      --parent string       filter by parent namespace id
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

* [gphq settings](gphq_settings.md)	 - Manage org settings

