## gphq prices create

Create a price

### Synopsis

Create a new price for a variant. Tiers and filter fields can only be set via --data.

```
gphq prices create [flags]
```

### Examples

```
  gphq prices create --variant var_1 --category subscription --scheme fixed --currency USD --unit-price 999 --interval month --interval-qty 1
  gphq prices create --data @price.json
```

### Options

```
      --category string         price category: one_time, subscription, free, variable (required)
      --currency string         ISO 4217 currency code, e.g. USD (required)
      --cycles int              number of billing cycles (0 = unlimited)
      --data string             raw JSON body (@file, -, or inline)
  -h, --help                    help for create
      --interval string         billing interval: none, minute, hour, day, week, month, year
      --interval-qty int        billing interval quantity
      --label string            display label
      --metadata stringArray    metadata key=value pairs (repeatable)
      --scheme string           price scheme: fixed, tiered, volume, graduated, package (required)
      --trial-interval string   trial period interval: none, minute, hour, day, week, month, year
      --trial-qty int           trial period quantity
      --unit-price int          unit price in smallest currency unit (e.g. cents)
      --variant string          variant ID (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq prices](gphq_prices.md)	 - Manage prices

