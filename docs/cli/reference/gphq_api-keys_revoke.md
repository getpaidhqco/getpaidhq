## gphq api-keys revoke

Revoke an API key

### Synopsis

Permanently revoke an API key by ID.

```
gphq api-keys revoke <id> [flags]
```

### Examples

```
  gphq api-keys revoke key_abc123
```

### Options

```
  -h, --help   help for revoke
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq api-keys](gphq_api-keys.md)	 - Manage API keys

