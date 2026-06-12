## gphq dunning campaigns update

Update a dunning campaign

### Synopsis

Update the status of a dunning campaign.

Status must be one of: active, paused, cancelled.
Pass --status to change the campaign state and optionally --reason to record why.
Use --data to send a raw JSON body instead of flags.

```
gphq dunning campaigns update <id> [flags]
```

### Examples

```
  gphq dunning campaigns update dc_1 --status paused --reason "investigating payment issue"
  gphq dunning campaigns update dc_1 --data '{"status":"cancelled","reason":"customer churned"}'
```

### Options

```
      --data string     raw JSON body (@file, -, or inline)
  -h, --help            help for update
      --reason string   reason for the status change
      --status string   new campaign status: active, paused, or cancelled (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq dunning campaigns](gphq_dunning_campaigns.md)	 - Manage dunning campaigns

