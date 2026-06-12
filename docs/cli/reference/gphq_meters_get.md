## gphq meters get

Get a meter

### Synopsis

Fetch a single billable meter by ID.

```
gphq meters get <id> [flags]
```

### Examples

```
  gphq meters get met_abc123
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq meters](gphq_meters.md)	 - Manage billable meters

