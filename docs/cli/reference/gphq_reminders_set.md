## gphq reminders set

Set reminder config

### Synopsis

Set the renewal reminder configuration for the organization.

Offsets are Go duration strings relative to the renewal date (e.g. "168h" = 7
days before renewal, "24h" = 1 day before). Pass --offset multiple times to
configure several reminder points.

```
gphq reminders set [flags]
```

### Examples

```
  gphq reminders set --enabled --offset 168h --offset 24h
  gphq reminders set --data '{"enabled":true,"offsets":["168h","24h"]}'
```

### Options

```
      --data string          raw JSON body (@file, -, or inline)
      --enabled              enable renewal reminders
  -h, --help                 help for set
      --offset stringArray   reminder offset before renewal, e.g. 168h (repeatable)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq reminders](gphq_reminders.md)	 - Manage renewal reminder config

