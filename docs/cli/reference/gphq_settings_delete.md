## gphq settings delete

Delete a setting

### Synopsis

Permanently delete a setting by parent ID and setting ID.

```
gphq settings delete <parentId> <id> [flags]
```

### Examples

```
  gphq settings delete ui color
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq settings](gphq_settings.md)	 - Manage org settings

