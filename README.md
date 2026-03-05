## shopify-app-authentication (go-chi)

### Doc
[shopify-session-tokens](https://shopify.dev/docs/apps/build/authentication-authorization/session-tokens)

### Run

```bash
cd shopify-app-authentication
go run ./cmd/server -config ./config/config.yaml
```

### Test

```bash
curl http://localhost:9998/ping
```

Expected output:

```text
pong
```

### Protected route with Shopify session token

When the config file contains `shopify_api_key` and `shopify_api_secret`, the app enables:

- `GET /admin/ping`

The route expects:

- `Authorization: Bearer <shopify_session_token>`

The middleware validates:

- JWT signature algorithm is `HS256`
- `exp`, `nbf`, `aud`, `iss`, and `dest` claims
- `iss` host and `dest` host must match

On successful validation, the server logs JWT claims:

- `shopify_session_token claims=...`

### Logging

Structured logging via [zerolog](https://github.com/rs/zerolog). Logs are written to both **stdout** (human-readable) and **weekly-rotated files** (JSON).

Log files are always written to `./logs` (relative to the working directory).  
Files are named `app-{year}-W{week}.log`, e.g. `app-2026-W09.log`.

Config keys in `config/config.yaml`:

| Key | Default | Description |
|---|---|---|
| `log_level` | `info` | Minimum log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `debug_auth` | `false` | Print detailed JWT / HMAC auth debug logs (at debug/warn level) |

In Docker, the compose file maps the container `./logs` to a host directory via `LOG_DIR`:

```bash
LOG_DIR=/var/log/shopify-app docker compose up -d
```

Or edit `docker-compose.yaml` directly to mount any server path.

### Project layout

- `cmd/server/main.go`: server entrypoint
- `internal/config/config.go`: YAML config loader
- `internal/logger/`: zerolog setup + weekly file rotation
- `internal/middleware/`: CORS, auth, request logger
- `internal/httpapi/`: router and handlers
- `config/config.yaml`: runtime config file

