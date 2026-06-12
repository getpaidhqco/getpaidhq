## gphq settings create

Create a setting

### Synopsis

Create a new org setting. The --id flag is required.

```
gphq settings create [flags]
```

### Examples

```
  gphq settings create --id theme --value dark
  gphq settings create --parent ui --id color --type string --value blue
```

### Options

```
      --data string     raw JSON body (@file, -, or inline)
  -h, --help            help for create
      --id string       setting id (required)
      --parent string   parent namespace id (optional)
      --type string     value type hint, e.g. string, json
      --value string    setting value
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq settings](gphq_settings.md)	 - Manage org settings

