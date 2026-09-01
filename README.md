# Singlepage

Singlepage is a self-hosted, zero-knowledge infinite outline for private notes.
The browser encrypts every page before it is sent to the server. The server
stores only ciphertext and cannot read the document contents or password.

Main capabilities:

- nested outline editing and full-text search;
- encrypted sharing links;
- password and access-link rotation;
- Markdown import and export;
- light and dark themes;
- a single Go binary with the frontend embedded.

## Quick start

Docker Compose is the simplest way to run Singlepage:

```bash
docker compose up -d --build
```

Open <http://localhost:8080>. Data is stored in the persistent
`singlepage-data` Docker volume.

Stop the application without deleting its data:

```bash
docker compose down
```

Use `docker compose down -v` only when the stored pages should also be deleted.

## Local development

```bash
npm install
make dev
```

The frontend is available at <http://localhost:5173> and proxies API requests
to the Go server at `127.0.0.1:8080`. Private Prometheus metrics listen on
`127.0.0.1:9090`.

## Standalone production binary

```bash
npm run build
go build -o singlepage . # use Go 1.24.4 or newer
SINGLEPAGE_HTTP_LISTEN=127.0.0.1:8080 \
SINGLEPAGE_SQLITE_DSN=./data.db \
./singlepage
```

The application server speaks plain HTTP and binds to loopback by default. For
remote access, use the Caddy stack below or another HTTPS reverse proxy. Do not
publish the application port directly on an untrusted network.

The resulting Go binary serves the embedded Svelte build and stores only opaque ciphertext, KDF salt, revision, and a hash of the write capability in SQLite.

## Docker configuration

The application is bound only to the host loopback interface by default.

To publish the service on another host port:

```bash
PORT=3000 docker compose up -d --build
```

### Docker with automatic HTTPS

Point the domain's `A`/`AAAA` DNS records to the server, allow inbound TCP
ports 80 and 443, then start the standalone Caddy stack:

```bash
DOMAIN=singlepage.example.com docker compose -f compose.caddy.yaml up -d --build
```

Caddy obtains and renews the TLS certificate automatically. Singlepage is only
reachable through Caddy; its port 8080 is not published on the host. Caddy's
certificate and configuration state are persisted in named Docker volumes.

## Security controls

The browser derives the encryption key from the password and the secret URL
fragment. The fragment is not sent in HTTP requests, and the server stores only
opaque ciphertext, salt, revision, timestamps, and hashes of write
capabilities. Changing the password also rotates the write capability; creating
a new access link replaces the page ID, fragment secret, and write capability.

Anyone who obtains the complete access link can download the ciphertext and try
password guesses offline. New passwords must contain at least 8 characters,
including a letter and a number; use a long, unique passphrase and share the
link and password through separate trusted channels. The fragment can still be exposed
through browser history, screenshots, clipboard managers, or browser
extensions.

The server applies conservative defaults to unauthenticated storage:

- 16 MiB maximum request body;
- 100,000 stored pages;
- 512 MiB SQLite logical size;
- one sustained page creation per second per client address, with a burst of 20.

They can be adjusted with `SINGLEPAGE_REQUEST_MAX_BODY_BYTES`,
`SINGLEPAGE_PAGE_MAX_PAGES`, `SINGLEPAGE_SQLITE_MAX_BYTES`,
`SINGLEPAGE_CREATE_RATE_PER_SECOND`, and `SINGLEPAGE_CREATE_BURST`.
Set a database, page-count, or create-rate value to `0` only when an external
control provides the equivalent protection.

The Caddy Compose stack enables `SINGLEPAGE_TRUST_PROXY_HEADERS`; the server then uses the
last address appended to `X-Forwarded-For`. Do not enable this option when an
untrusted client can reach the application port without going through a reverse
proxy that appends the real client address.

### Environment configuration

All runtime parameters are read from `SINGLEPAGE_*` environment variables. Copy
`.env.example` as a reference; the application does not load dotenv files by
itself. Durations use Go syntax such as `5s` or `10m`.

| Variable | Default |
| --- | --- |
| `SINGLEPAGE_HTTP_LISTEN` | `127.0.0.1:8080` |
| `SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT` | `5s` |
| `SINGLEPAGE_HTTP_READ_TIMEOUT` | `20s` |
| `SINGLEPAGE_HTTP_WRITE_TIMEOUT` | `20s` |
| `SINGLEPAGE_HTTP_IDLE_TIMEOUT` | `60s` |
| `SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT` | `10s` |
| `SINGLEPAGE_METRICS_LISTEN` | `127.0.0.1:9090` |
| `SINGLEPAGE_SQLITE_DSN` | `data.db` |
| `SINGLEPAGE_SQLITE_MAX_BYTES` | `536870912` |
| `SINGLEPAGE_PAGE_MAX_PAGES` | `100000` |
| `SINGLEPAGE_REQUEST_MAX_BODY_BYTES` | `16777216` |
| `SINGLEPAGE_CREATE_RATE_PER_SECOND` / `SINGLEPAGE_CREATE_BURST` | `1` / `20` |
| `SINGLEPAGE_ADMIN_RATE_PER_SECOND` / `SINGLEPAGE_ADMIN_BURST` | `1` / `5` |
| `SINGLEPAGE_TRUST_PROXY_HEADERS` | `false` |
| `SINGLEPAGE_ADMIN_TOKEN_FILE` | empty; admin deletion disabled |
| `SINGLEPAGE_CORS_ALLOWED_ORIGINS` | empty; CORS disabled |
| `SINGLEPAGE_CORS_ALLOWED_METHODS` | `GET,HEAD,POST,PUT,DELETE,OPTIONS` |
| `SINGLEPAGE_CORS_ALLOWED_HEADERS` | `Authorization,Content-Type` |
| `SINGLEPAGE_CORS_EXPOSED_HEADERS` | `X-Request-ID` |
| `SINGLEPAGE_CORS_ALLOW_CREDENTIALS` | `false` |
| `SINGLEPAGE_CORS_MAX_AGE` | `0s` |

CORS lists are comma-separated. Origins are validated explicitly; wildcard
origins cannot be combined with credentials.

Browser responses include a restrictive Content Security Policy, anti-framing,
cross-origin isolation, no-referrer, and no-index headers. HSTS is emitted for
HTTPS requests and by the included Caddy configuration.

### Observability

Every response includes a cryptographically random `X-Request-ID`. The server
writes one structured JSON log entry for every request, including its request
ID, HTTP method, matched route pattern, status, response size, and duration.
Client errors are logged at `WARN`; server errors are logged at `ERROR` and
include the recorded request error when one is available. Route templates are
logged instead of concrete page IDs.

Prometheus metrics for API traffic are exposed only by the separate metrics
server at `http://127.0.0.1:9090/metrics` by default. The metrics listener must
use a loopback address and a port different from the public application server;
it is not published by either Docker Compose configuration.

Available metrics:

- `singlepage_api_requests_total`;
- `singlepage_api_request_errors_total`;
- `singlepage_api_request_duration_seconds`;
- `singlepage_api_requests_in_flight`.

The request counters and duration histogram use bounded `method`, `route`, and
`status` labels. The metrics endpoint itself and frontend traffic are excluded
from API usage metrics.

### Optional administrative deletion

The deletion API is disabled unless an admin token file is configured. Generate
a file readable only by the service account and start the server with it:

```bash
umask 077
openssl rand -base64 32 > admin-token
SINGLEPAGE_ADMIN_TOKEN_FILE=./admin-token ./singlepage
```

Delete an abusive or compromised page without decrypting it:

```bash
curl -X DELETE \
  -H "Authorization: Bearer $(cat admin-token)" \
  http://127.0.0.1:8080/api/admin/pages/PAGE_ID
```

For containers, mount the token as a read-only Docker secret or bind mount and
set `SINGLEPAGE_ADMIN_TOKEN_FILE=/run/secrets/singlepage-admin`. The environment
contains only the file path; never put the token value directly in the image,
Compose file, environment, URL, or command line.

Stop the HTTPS stack without deleting its data:

```bash
DOMAIN=singlepage.example.com docker compose -f compose.caddy.yaml down
```

#### Local HTTPS

Use `singlepage.localhost` to run the same stack locally without configuring DNS:

```bash
DOMAIN=singlepage.localhost docker compose -f compose.caddy.yaml up -d --build
```

Caddy uses its local CA for this address. Export its root certificate once:

```bash
DOMAIN=singlepage.localhost docker compose -f compose.caddy.yaml cp \
  caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-local-root.crt
```

Trust it on macOS:

```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ./caddy-local-root.crt
```

On Ubuntu:

```bash
sudo apt-get install -y ca-certificates
sudo cp ./caddy-local-root.crt /usr/local/share/ca-certificates/caddy-local-root.crt
sudo update-ca-certificates
```

On Windows, run PowerShell as Administrator:

```powershell
$env:DOMAIN = "singlepage.localhost"
docker compose -f compose.caddy.yaml cp `
  caddy:/data/caddy/pki/authorities/local/root.crt .\caddy-local-root.crt
certutil -addstore root .\caddy-local-root.crt
```

Restart the browser, then open <https://singlepage.localhost>.

## Verification

```bash
npm run check
npm test
npm run test:e2e
go test -race ./...
go vet ./...
golangci-lint run ./...
```

Playwright uses an installed Google Chrome channel by default. The browser performs all document parsing, indexing, searching, key derivation, encryption, and decryption.
