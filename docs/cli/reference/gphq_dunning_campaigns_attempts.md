## gphq dunning campaigns attempts

List attempts for a dunning campaign

### Synopsis

List all retry attempts for a dunning campaign.

```
gphq dunning campaigns attempts <id> [flags]
```

### Examples

```
  gphq dunning campaigns attempts dc_1
  gphq dunning campaigns attempts dc_1 --limit 50
```

### Options

```
  -h, --help                help for attempts
      --limit int           items per page (default 10)
      --page int            page number (zero-indexed)
      --sort-by string      sort field (default "created_at")
      --sort-order string   asc or desc (default "desc")
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq dunning campaigns](gphq_dunning_campaigns.md)	 - Manage dunning campaigns

