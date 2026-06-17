## gphq carts add

Add a product to a cart

### Synopsis

Add a product variant/price to an existing cart. Pass --product and --price, or --data for a raw JSON body.

```
gphq carts add <cartId> [flags]
```

### Examples

```
  gphq carts add cart_1 --product prod_1 --price pri_1 --qty 2
  gphq carts add cart_1 --data '{"product_id":"prod_1","price_id":"pri_1","quantity":1}'
```

### Options

```
      --data string      raw JSON body (@file, -, or inline)
  -h, --help             help for add
      --price string     price ID (required)
      --product string   product ID (required)
      --qty int          quantity (default 1)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq carts](gphq_carts.md)	 - Manage carts

