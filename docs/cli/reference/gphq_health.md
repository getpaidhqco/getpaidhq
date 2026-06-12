## gphq health

Check API server health

### Synopsis

Calls the unauthenticated /api/health endpoint and prints the result.

```
gphq health [flags]
```

### Examples

```
  gphq health
```

### Options

```
  -h, --help   help for health
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq](gphq.md)	 - Command-line client for the GetPaidHQ API

