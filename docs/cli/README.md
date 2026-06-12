# gphq CLI

`gphq` is the command-line client for the GetPaidHQ subscription-billing API. It talks to a running `gphq-server` instance and covers every endpoint the server exposes — products, variants, prices, customers, orders, subscriptions, invoices, payments, dunning, meters, usage, and more.

## Install

```
make install-cli     # go install ./cmd/gphq  (adds gphq to $GOPATH/bin)
# or
make build-cli       # writes bin/gphq in the repo
```

Requires Go 1.26+.

## Authentication

API keys are org-scoped and shown once at creation time. Your first key must come from the dashboard (key creation itself requires authentication). Once you have one, further keys can be minted from the CLI:

```
gphq api-keys create --name "my-key"
```

Provide the key in one of three ways (highest precedence first):

1. `--api-key sk_...` flag
2. `GPHQ_API_KEY=sk_...` environment variable
3. Config file `~/.config/gphq/config.toml`

```toml
api_key  = "sk_..."
base_url = "http://localhost:10081"
```

The config file path respects `$XDG_CONFIG_HOME` if set.

## Configuration

| Setting    | Flag         | Env var          | Default                    |
|------------|--------------|------------------|----------------------------|
| API key    | `--api-key`  | `GPHQ_API_KEY`   | (none — required for most commands) |
| Server URL | `--base-url` | `GPHQ_BASE_URL`  | `http://localhost:10081`   |
| Output     | `-o`/`--output` | `GPHQ_OUTPUT` | `table`                    |

Precedence: flag > env > config file > default.

## Output and scripting

By default, commands print a human-readable table. Pass `-o json` to get the raw API response body as JSON, which is useful for piping to `jq`:

```
gphq customers list -o json | jq '.data[].id'
```

Errors are written to stderr. API errors carry the server's error code:

```
error (<code>): <message>
```

Usage and network errors print as `error: <message>`.

Exit codes: `0` success, `1` API/network/config error, `2` usage error (bad flags or arguments).

## Request bodies

Common fields have typed flags (e.g. `--email`, `--currency`). For full or nested payloads use `--data`:

- `--data @file.json` — read body from a file
- `--data -` — read body from stdin
- `--data '{"key":"value"}'` — inline JSON

`--data` and typed field flags are mutually exclusive on any given invocation.

## Pagination

List commands accept:

- `--page <n>` — page number, zero-indexed (default 0, the first page)
- `--limit <n>` — results per page (default 10)
- `--sort-by <field>` — field to sort by
- `--sort-order asc|desc` — sort direction

Table output includes a footer showing the total count, current page, and limit.

## End-to-end walkthrough

This sequence creates a product with a variant and price, then creates a customer and an order against it.

### 1. Create a product with a variant and price

The API requires at least one variant. Use `--data` with a complete body:

```
cat > product.json <<'EOF'
{
  "name": "Acme Pro",
  "description": "Monthly subscription plan",
  "variants": [
    {
      "name": "Standard",
      "prices": [
        {
          "category": "subscription",
          "scheme": "fixed",
          "currency": "USD",
          "unit_price": 2900,
          "billing_interval": "month",
          "billing_interval_qty": 1
        }
      ]
    }
  ]
}
EOF

gphq products create --data @product.json -o json > created_product.json
```

### 2. Capture IDs from the response

```
PRODUCT_ID=$(jq -r '.id' created_product.json)
VARIANT_ID=$(jq -r '.variants[0].id' created_product.json)
PRICE_ID=$(jq -r '.variants[0].prices[0].id' created_product.json)

echo "product=$PRODUCT_ID  variant=$VARIANT_ID  price=$PRICE_ID"
```

### 3. Create a customer

```
gphq customers create --email ada@example.com --first-name Ada --last-name Lovelace -o json > customer.json
CUSTOMER_ID=$(jq -r '.id' customer.json)
```

### 4. Create an order

```
gphq orders create \
  --customer-id "$CUSTOMER_ID" \
  --psp paystack \
  --currency USD \
  --item "product=$PRODUCT_ID,price=$PRICE_ID" \
  -o json > order.json

ORDER_ID=$(jq -r '.order.id' order.json)
```

### 5. Complete the order

```
gphq orders complete "$ORDER_ID"
```

### 6. List subscriptions

```
gphq subscriptions list
```

Or inspect the subscriptions tied to that order:

```
gphq orders subscriptions "$ORDER_ID"
```

## Command reference

Full per-command documentation: [reference/gphq.md](reference/gphq.md)
