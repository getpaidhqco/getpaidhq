## gphq dunning configs create

Create a dunning configuration

### Synopsis

Create a new dunning retry configuration.

The API requires a nested "config" object (retry schedule, escalation policy, etc.)
that cannot be expressed with flags alone — flag-only creates will be rejected by the
server unless you also supply the full config via --data. Use --data for complete
configurations; flags are provided as a convenience for simple cases.

Example --data payload:
  {
    "name": "Standard retry",
    "applies_to": "all",
    "config": {
      "immediate_attempts": 1,
      "progressive_attempts": [{"delay_hours": 24}, {"delay_hours": 72}],
      "escalation_policy": "cancel"
    }
  }

```
gphq dunning configs create [flags]
```

### Examples

```
  gphq dunning configs create --data @config.json
  gphq dunning configs create --name "Standard" --applies-to all --data '{"config":{"immediate_attempts":1,"escalation_policy":"cancel"}}'
```

### Options

```
      --applies-to string    scope: all, product, customer, etc. (required)
      --data string          raw JSON body (@file, -, or inline); the API requires a nested config object so complex configurations must use this flag
      --description string   optional description
  -h, --help                 help for create
      --name string          configuration name (required)
      --priority int         priority (lower = higher precedence)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq dunning configs](gphq_dunning_configs.md)	 - Manage dunning configurations

