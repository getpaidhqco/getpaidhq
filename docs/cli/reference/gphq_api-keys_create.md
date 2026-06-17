## gphq api-keys create

Create an API key

### Synopsis

Create a new API key. The plaintext key is returned once and never shown again.

```
gphq api-keys create [flags]
```

### Examples

```
  gphq api-keys create --name ci-deploy
  gphq api-keys create --data '{"name":"my-key"}'
```

### Options

```
      --data string   raw JSON body (@file, -, or inline)
  -h, --help          help for create
      --name string   human-readable label for the key (optional)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq api-keys](gphq_api-keys.md)	 - Manage API keys

