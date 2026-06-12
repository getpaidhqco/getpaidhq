## gphq usage ingest

Ingest a usage event

### Synopsis

Ingest a single usage event (wrapped as a one-element batch).

Exactly one of --customer or --external-customer should identify the customer.
The event timestamp defaults to the ingestion time when omitted; pass an RFC3339
value to set an explicit time (e.g. 2026-06-12T10:00:00Z).

To ingest multiple events in one request pass --data with a full
{"events":[...]} body.

```
gphq usage ingest [flags]
```

### Examples

```
  gphq usage ingest --metric api_calls --customer cus_1
  gphq usage ingest --metric bytes --customer cus_1 --metadata bytes=1024 --timestamp 2026-06-12T10:00:00Z
  gphq usage ingest --data '{"events":[{"metric_code":"api_calls","customer_id":"cus_1"}]}'
```

### Options

```
      --customer string            customer ID (customer_id)
      --data string                raw JSON body — full {"events":[...]} batch (@file, -, or inline)
      --external-customer string   external customer ID (external_customer_id)
      --external-id string         idempotency key for the event (external_id)
  -h, --help                       help for ingest
      --metadata stringArray       event metadata key=value pairs (repeatable)
      --metric string              meter code the event counts against (metric_code; required)
      --subscription string        subscription ID (subscription_id)
      --timestamp string           event timestamp in RFC3339 format (defaults to zero time)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq usage](gphq_usage.md)	 - Ingest usage events

