## gphq orgs create

Create an organization

### Synopsis

Create a new organization. Name, country, and timezone are required.

```
gphq orgs create [flags]
```

### Examples

```
  gphq orgs create --name "Acme Corp" --country NG --timezone Africa/Lagos
  gphq orgs create --data '{"name":"Acme","country":"NG","timezone":"UTC"}'
```

### Options

```
      --country string         ISO 3166-1 alpha-2 country code (required)
      --data string            raw JSON body (@file, -, or inline)
  -h, --help                   help for create
      --metadata stringArray   metadata key=value pairs (repeatable)
      --name string            organization name (required)
      --timezone string        IANA timezone name, e.g. Africa/Lagos (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq orgs](gphq_orgs.md)	 - Manage organizations

