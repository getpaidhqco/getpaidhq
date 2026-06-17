## gphq payment-tokens activate

Activate a payment update token

### Synopsis

Activate a payment update token (marks it as used for a payment method update).

```
gphq payment-tokens activate <tokenId> [flags]
```

### Examples

```
  gphq payment-tokens activate tok_abc123
```

### Options

```
  -h, --help   help for activate
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq payment-tokens](gphq_payment-tokens.md)	 - Manage payment update tokens

