# Local dev: Hatchet UI & the cookie-secret 500

## Opening the Hatchet UI

The all-in-one `hatchet-lite` container (engine + API + UI) is in `docker/docker-compose.yml`.
Its UI/API port `8888` is published to the host as **`10888`** (loopback only):

```
ports:
  - "127.0.0.1:10888:8888"   # UI + REST API
  - "127.0.0.1:10707:7077"   # gRPC (SDK dials this; broadcast addr must match)
```

So the UI is at **http://localhost:10888**.

Bring the stack up first:

```bash
docker compose -f docker/docker-compose.yml up -d
```

## The `/auth/login` 500 error

### Symptom
Visiting `http://localhost:10888/auth/login` returns **500**. Container logs (`docker logs hatchet-lite`) show, on every request:

```
ERR error saving unauthenticated session
    error="securecookie: error - caused by: crypto/aes: invalid key size 7"
ERR API status=500 uri=/api/v1/users/current ...
```

### Root cause
The login *page* is fine — the 500 comes from the Hatchet API failing to **sign session cookies**.
`securecookie` uses two keys: a hash key and an **AES block (encryption) key**. The AES key must be
exactly **16, 24, or 32 bytes**. The compose file had:

```yaml
SERVER_AUTH_COOKIE_SECRETS: "secret1 secret2"
```

`"secret2"` is **7 bytes** → `invalid key size 7` → every session write 500s, including the login flow.

### Fix
Use two properly-sized secrets (32 hex chars = 16 bytes each works):

```yaml
# securecookie keys: first is the hash key, second is the AES block
# (encryption) key — the block key MUST be 16, 24, or 32 bytes or the
# server 500s on every session with "crypto/aes: invalid key size".
SERVER_AUTH_COOKIE_SECRETS: "<32-hex-chars> <32-hex-chars>"
```

Generate with `openssl rand -hex 16` (twice). Then recreate the container:

```bash
docker compose -f docker/docker-compose.yml up -d hatchet-lite
```

### Verify
`/api/v1/users/current` should return **403** (unauthenticated) instead of **500** (cookie can now be
signed). Reload the login page and log in fresh — changing the secrets invalidates any old session
cookies. The Hatchet **client token** (`HATCHET_CLIENT_TOKEN`) is unaffected; it lives in the
`/config` volume and uses a separate keyset.

## Port reference (local)

| Service | Host port | Notes |
| --- | --- | --- |
| Hatchet UI + REST | `10888` | http://localhost:10888 |
| Hatchet gRPC | `10707` | SDK `HATCHET_CLIENT_HOST_PORT`; broadcast addr must match |
| Postgres (app/reports/hatchet DBs) | `10432` | `getpaidhq` / `getpaidhq_reports` / `hatchet` |
| API server | `SERVER_PORT` | see [run gotchas memory] — runs on :10081 locally |
