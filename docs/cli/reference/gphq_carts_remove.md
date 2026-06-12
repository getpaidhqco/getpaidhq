## gphq carts remove

Remove an item from a cart

### Synopsis

Remove a line item from a cart by its item ID.

```
gphq carts remove <cartId> [flags]
```

### Examples

```
  gphq carts remove cart_1 --item-id item_abc
  gphq carts remove cart_1 --data '{"id":"item_abc"}'
```

### Options

```
      --data string      raw JSON body (@file, -, or inline)
  -h, --help             help for remove
      --item-id string   cart item ID to remove (required)
```

### Options inherited from parent commands

```
      --api-key string    API key (env GPHQ_API_KEY)
      --base-url string   API base URL (env GPHQ_BASE_URL) (default "http://localhost:10081")
  -o, --output string     output format: table|json (env GPHQ_OUTPUT) (default "table")
```

### SEE ALSO

* [gphq carts](gphq_carts.md)	 - Manage carts

