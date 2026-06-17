# `gphq` CLI — Design

2026-06-12. Approved direction: hand-written Cobra commands, living in this repo, flags + JSON fallback for input, no TUI, comprehensive docs.

## Purpose

A command-line client for the GetPaidHQ REST API, covering every endpoint the server
exposes (21 resource groups, ~79 operations). Primary uses: day-to-day CRUD against a
running server (create orders, manage subscriptions, customers, products), and
scriptable access (`-o json`) for automation. Follows the conventions of Stripe CLI /
`gh` / kubectl: Go, Cobra + Viper, noun-verb commands, env-var auth, table-or-JSON
output, meaningful exit codes.

## Shape

- Single binary `gphq`, entrypoint `cmd/gphq/main.go`; all real code in `internal/cli/`.
- Same Go module as the server: command code imports the server's own request/response
  DTOs from `internal/adapter/http` and domain types from `internal/core/domain` —
  typed end-to-end, no codegen, no drift.
- New dependency: `github.com/spf13/cobra` (Viper is already in go.mod).
- Makefile targets: `build-cli` (binary to `bin/gphq`), `install-cli` (`go install ./cmd/gphq`),
  `docs-cli` (regenerate the markdown command reference).
- Explicitly **no TUI** — no interactive prompts, no charmbracelet deps. Plain
  stdin/stdout/stderr only.

## Configuration

Precedence: flag > environment > config file > default (Viper layering).

| Setting  | Flag         | Env             | Default                  |
| -------- | ------------ | --------------- | ------------------------ |
| API key  | `--api-key`  | `GPHQ_API_KEY`  | — (required, except `health`) |
| Base URL | `--base-url` | `GPHQ_BASE_URL` | `http://localhost:10081` |
| Output   | `-o/--output`| `GPHQ_OUTPUT`   | `table`                  |

Optional config file `~/.config/gphq/config.toml` with the same keys. The API key is
sent as `x-api-key`. Keys are minted in the dashboard or via `POST /api/api-keys`
(itself authenticated), so the CLI assumes the user already holds a key — same
bootstrap model as Stripe.

## Command tree

Noun-verb. Full coverage of every route registered in `internal/config/server.go`:

```
gphq orders          create | list | get | complete | subscriptions
gphq subscriptions   list | get | update | pause | resume | cancel | billing-anchor | payments | invoices | usage
gphq customers       create | list | get | payment-methods add/update | dunning-history
gphq products        create | list | get | update | delete | archive | unarchive | variants list/add
gphq variants        get | update | delete | prices
gphq prices          create | get | update | delete
gphq invoices        list | get
gphq payments        list | get
gphq payment-methods get
gphq payment-tokens  verify | activate
gphq carts           add | remove
gphq sessions        create
gphq api-keys        create | list | revoke
gphq orgs            create
gphq gateways        create
gphq dunning         campaigns list/get/update/attempts/communications | configs list/get/create
gphq meters          create | list | get
gphq usage           ingest
gphq reminders       get | set
gphq settings        list | get | create | update | delete
gphq webhooks        create | list
gphq health
gphq completion      (Cobra built-in)
gphq version
```

When the server grows an endpoint, the matching command is added by hand in the same
PR — checked by the coverage test (see Testing).

## Input

- Typed flags for the common fields of each create/update
  (`gphq customers create --email x@y.com --name "Acme"`).
- Every create/update also accepts `--data @file.json` or `--data -` (stdin) carrying
  the full request body — required path for nested payloads (e.g. order line items).
  `--data` and typed flags are mutually exclusive; mixing them is a usage error.
- List commands take `--page --limit --sort-by --sort-order`, mirroring the server's
  query params.

## Output, errors, exit codes

- Default: human-readable tables via stdlib `text/tabwriter`; a curated column set per
  resource (id, key business fields, status, created_at).
- `-o json`: the raw API response body, pretty-printed — stable surface for jq/scripts.
- Errors: parse the `{code,message,details}` `ApiError` envelope and print
  `error (<code>): <message>` plus details to **stderr**; tables/JSON go to stdout only.
- Exit codes: `0` success, `1` API or network error, `2` usage error (Cobra default).

## Internals

```
cmd/gphq/main.go              thin entrypoint (calls cli.Execute)
internal/cli/root.go          root command, global flags, viper wiring
internal/cli/client/          HTTP client: base URL, x-api-key header, timeout,
                              ApiError → error mapping, list-envelope helpers
internal/cli/output/          table and JSON renderers
internal/cli/commands/        one file per resource group (orders.go, subscriptions.go, …)
```

The client is a thin `net/http` wrapper (no SDK, no retries in v1; 30s timeout).
Commands construct server DTOs, call the client, hand the response to the renderer.

## Documentation (deliverable, not afterthought)

1. **`--help` everywhere**: every command gets `Short`, `Long`, and at least one
   `Example`. This is the primary docs surface.
2. **Guide**: `docs/cli/README.md` — install, getting an API key, configuration,
   output formats, scripting with `-o json`, exit codes, end-to-end walkthrough
   (create product → price → customer → order → complete → inspect subscription).
3. **Generated reference**: `docs/cli/reference/*.md`, one page per command, generated
   from the Cobra tree via `cobra/doc` through `make docs-cli`. A test fails if the
   committed reference is stale relative to the command tree.
4. Top-level `README.md` gains a short CLI section pointing at `docs/cli/`.

## Testing

- Table-driven tests per command file against `httptest.Server` fakes: assert exact
  method, path, query, headers (`x-api-key`), and body sent; assert rendered table and
  JSON output; assert error rendering and exit codes from canned `ApiError` responses.
- A coverage test walks `openapi.json` at the repo root and fails if any documented
  operation has no corresponding command mapping — keeps "full coverage" honest over time.
- No live-server or integration-tagged tests; the CLI never touches a database.

## Out of scope (v1)

TUI/interactive mode, `--all` auto-pagination, generic `gphq api <method> <path>`
passthrough, request retries, response caching, self-update, telemetry.
