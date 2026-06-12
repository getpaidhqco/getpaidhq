## gphq dunning configs update

Update a dunning configuration

### Synopsis

Update fields on a dunning configuration. Unset flags are sent as empty values, which the server ignores. Note: priority 0 cannot be set via flags (the server treats 0 as unset); use --data.

```
gphq dunning configs update <id> [flags]
```

### Examples

```
  gphq dunning configs update dcfg_1 --name "Updated name" --status active
  gphq dunning configs update dcfg_1 --data '{"name":"New name","status":"active"}'
```

### Options

```
      --data string          raw JSON body (@file, -, or inline)
      --description string   updated description
  -h, --help                 help for update
      --name string          new configuration name
      --priority int         updated priority
      --status string        new status, e.g. active, inactive
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq dunning configs](gphq_dunning_configs.md)	 - Manage dunning configurations

