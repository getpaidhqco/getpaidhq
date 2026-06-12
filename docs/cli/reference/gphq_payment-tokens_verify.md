## gphq payment-tokens verify

Verify a payment update token

### Synopsis

Verify that a payment update token is valid and retrieve its metadata.

```
gphq payment-tokens verify <tokenId> [flags]
```

### Examples

```
  gphq payment-tokens verify tok_abc123
```

### Options

```
  -h, --help   help for verify
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq payment-tokens](gphq_payment-tokens.md)	 - Manage payment update tokens

