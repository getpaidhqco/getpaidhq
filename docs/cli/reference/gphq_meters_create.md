## gphq meters create

Create a meter

### Synopsis

Create a new billable meter.

A meter defines what usage events to measure and how to aggregate them. The
aggregation type determines how events are combined into a billable quantity.

Aggregation types: count sum max latest weighted_sum unique_count

Meter filters (rate dimensions per price) can only be specified via --data.
Example --data payload:
  {
    "code": "api_calls",
    "name": "API Calls",
    "aggregation": "count",
    "group_by": ["region"],
    "filters": [{"field": "tier", "values": ["pro", "enterprise"]}]
  }

```
gphq meters create [flags]
```

### Examples

```
  gphq meters create --code api_calls --name "API Calls" --aggregation count
  gphq meters create --code bytes --name "Bytes" --aggregation sum --field bytes_used --carry-over
  gphq meters create --data '{"code":"api_calls","name":"API Calls","aggregation":"count"}'
```

### Options

```
      --aggregation string     aggregation type: count sum max latest weighted_sum unique_count (required)
      --carry-over             carry over unused quota to the next billing period
      --code string            meter code — referenced by usage events (required)
      --data string            raw JSON body (@file, -, or inline); required for filter definitions
      --field string           event metadata field to aggregate (field_name); required for sum/max/latest/weighted_sum
      --group-by stringArray   metadata keys to group usage by (repeatable; v1 honours one key)
  -h, --help                   help for create
      --metadata stringArray   metadata key=value pairs (repeatable)
      --name string            human-readable meter name (required)
      --rounding-mode string   rounding mode: round ceil floor
      --rounding-scale int     rounding decimal scale (0–18)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq meters](gphq_meters.md)	 - Manage billable meters

