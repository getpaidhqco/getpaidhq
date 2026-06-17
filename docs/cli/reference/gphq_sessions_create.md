## gphq sessions create

Create a session

### Synopsis

Create a new checkout session. Pass --currency and --country, or --data for a raw JSON body.

```
gphq sessions create [flags]
```

### Examples

```
  gphq sessions create --currency USD --country US
  gphq sessions create --currency NGN --country NG --metadata src=api
  gphq sessions create --data '{"currency":"USD","country":"US"}'
```

### Options

```
      --country string         session country (required)
      --currency string        session currency (required)
      --data string            raw JSON body (@file, -, or inline)
  -h, --help                   help for create
      --metadata stringArray   metadata key=value pairs (repeatable)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq sessions](gphq_sessions.md)	 - Manage sessions

