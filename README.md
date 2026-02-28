## shopifyapp-authentication demo (go-chi)

### Doc
[shopify-session-tokens](https://shopify.dev/docs/apps/build/authentication-authorization/session-tokens)

### Run

```bash
cd shopifyapp-authentication
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

- `GET /protected/ping`

The route expects:

- `Authorization: Bearer <shopify_session_token>`

The middleware validates:

- JWT signature algorithm is `HS256`
- `exp`, `nbf`, `aud`, `iss`, and `dest` claims
- `iss` host and `dest` host must match

On successful validation, the server logs JWT claims:

- `shopify_session_token claims=...`

### Project layout

- `cmd/server/main.go`: server entrypoint
- `internal/config/config.go`: YAML config loader
- `internal/httpapi/`: router and Shopify JWT middleware
- `config/config.yaml`: runtime config file

