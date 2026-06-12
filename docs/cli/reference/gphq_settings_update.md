## gphq settings update

Update (upsert) a setting

### Synopsis

Create or replace a setting at the given parent and id.

```
gphq settings update <parentId> <id> [flags]
```

### Examples

```
  gphq settings update ui color --value red
  gphq settings update ui color --data '{"value":"red"}'
```

### Options

```
      --data string    raw JSON body (@file, -, or inline)
  -h, --help           help for update
      --type string    value type hint
      --value string   setting value
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq settings](gphq_settings.md)	 - Manage org settings

