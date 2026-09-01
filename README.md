# Singlepage

Singlepage is a self-hosted encrypted outline for private notes. Page contents
are encrypted before they leave the browser or desktop application, so the
server stores no readable document text or password.

## Features

- nested outline editing;
- full-text search inside an opened page;
- encrypted sharing links;
- password and access-link rotation;
- Markdown import and export;
- light and dark themes;
- browser, desktop, and headless daemon modes;
- optional HTTPS through the included Caddy configuration;
- optional Prometheus metrics;
- optional administrative deletion of encrypted pages.

Mobile applications are postponed and are not currently supported.

## Running Singlepage

### Docker Compose

Start the browser application on <http://localhost:8080>:

```bash
docker compose -f deploy/compose/standalone/compose.yaml up -d --build
```

Use another host port:

```bash
PORT=3000 docker compose -f deploy/compose/standalone/compose.yaml up -d --build
```

Stop the application while preserving its data:

```bash
docker compose -f deploy/compose/standalone/compose.yaml down
```

Delete the application together with the stored data:

```bash
docker compose -f deploy/compose/standalone/compose.yaml down -v
```

### Docker Compose with HTTPS

Point the domain's DNS records to the server, allow inbound TCP ports 80 and
443, and run:

```bash
DOMAIN=singlepage.example.com \
docker compose -f deploy/compose/caddy/compose.yaml up -d --build
```

Open `https://singlepage.example.com` in a browser. Stop the HTTPS deployment
without deleting its data:

```bash
DOMAIN=singlepage.example.com \
docker compose -f deploy/compose/caddy/compose.yaml down
```

### Browser application without Docker

Install the frontend dependencies and run the development servers:

```bash
npm install
make dev
```

Open <http://localhost:5173>.

Build and run the standalone browser application:

```bash
make build
SINGLEPAGE_HTTP_LISTEN=127.0.0.1:8080 \
SINGLEPAGE_SQLITE_DSN=./data.db \
./singlepage
```

Open <http://localhost:8080>.

### Desktop application

Build and run the desktop application:

```bash
make build-app
make run-app
```

For a direct development run:

```bash
make dev-app
```

Desktop data is stored in the current user's application-data directory:

- macOS: `~/Library/Application Support/Singlepage`;
- Windows: `%AppData%/Singlepage`;
- Linux: `$XDG_DATA_HOME/singlepage`, or `~/.local/share/singlepage`.

The desktop application does not open an HTTP port. It uses only
`SINGLEPAGE_SQLITE_MAX_BYTES` and `SINGLEPAGE_PAGE_MAX_PAGES`; the database
location is selected automatically.

### Headless daemon

Build and run the daemon without the browser interface:

```bash
make build-daemon
SINGLEPAGE_HTTP_LISTEN=127.0.0.1:8080 \
SINGLEPAGE_SQLITE_DSN=./data.db \
./bin/singlepage-daemon
```

The API description is available at <http://127.0.0.1:8080/openapi.json> for
third-party applications.

## Configuration

Runtime configuration is read from `SINGLEPAGE_*` environment variables. The
application does not load `.env` files automatically; `.env.example` contains a
complete example. Duration values use forms such as `5s`, `2m`, or `1h`.

### HTTP and metrics

| Variable | Default | Purpose |
| --- | --- | --- |
| `SINGLEPAGE_HTTP_LISTEN` | `127.0.0.1:8080` | Public browser or daemon address |
| `SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT` | `5s` | Request-header timeout |
| `SINGLEPAGE_HTTP_READ_TIMEOUT` | `20s` | Request-read timeout |
| `SINGLEPAGE_HTTP_WRITE_TIMEOUT` | `20s` | Response-write timeout |
| `SINGLEPAGE_HTTP_IDLE_TIMEOUT` | `60s` | Idle connection timeout |
| `SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `SINGLEPAGE_METRICS_LISTEN` | `127.0.0.1:9090` | Private Prometheus address |

Prometheus metrics are available at `/metrics` on the metrics address. The
metrics address must be loopback-only and use a different port from the public
application.

### Storage and limits

| Variable | Default | Purpose |
| --- | --- | --- |
| `SINGLEPAGE_SQLITE_DSN` | `data.db` | Database path or SQLite `file:` URI |
| `SINGLEPAGE_SQLITE_MAX_BYTES` | `536870912` | Maximum database size in bytes; `0` disables the limit |
| `SINGLEPAGE_PAGE_MAX_PAGES` | `100000` | Maximum stored page count; `0` disables the limit |
| `SINGLEPAGE_REQUEST_MAX_BODY_BYTES` | `16777216` | Maximum HTTP request size in bytes |

Keep the database on a local filesystem or Docker volume.

### Request protection

| Variable | Default | Purpose |
| --- | --- | --- |
| `SINGLEPAGE_CREATE_RATE_PER_SECOND` | `1` | Sustained page-creation rate per client; `0` disables the limit |
| `SINGLEPAGE_CREATE_BURST` | `20` | Page-creation burst size |
| `SINGLEPAGE_ADMIN_RATE_PER_SECOND` | `1` | Sustained administrative request rate; `0` disables the limit |
| `SINGLEPAGE_ADMIN_BURST` | `5` | Administrative request burst size |
| `SINGLEPAGE_TRUST_PROXY_HEADERS` | `false` | Trust client addresses supplied by a reverse proxy |
| `SINGLEPAGE_ADMIN_TOKEN_FILE` | empty | File containing the administrative token |

Enable `SINGLEPAGE_TRUST_PROXY_HEADERS` only when every request passes through a
trusted reverse proxy. The included Caddy deployment enables it automatically.

Administrative deletion remains disabled while
`SINGLEPAGE_ADMIN_TOKEN_FILE` is empty. Create and use a token file:

```bash
umask 077
openssl rand -base64 32 > admin-token
SINGLEPAGE_ADMIN_TOKEN_FILE=./admin-token ./singlepage
```

Delete an encrypted page by ID:

```bash
curl -X DELETE \
  -H "Authorization: Bearer $(cat admin-token)" \
  http://127.0.0.1:8080/api/admin/pages/PAGE_ID
```

### CORS

| Variable | Default |
| --- | --- |
| `SINGLEPAGE_CORS_ALLOWED_ORIGINS` | empty; CORS disabled |
| `SINGLEPAGE_CORS_ALLOWED_METHODS` | `GET,HEAD,POST,PUT,DELETE,OPTIONS` |
| `SINGLEPAGE_CORS_ALLOWED_HEADERS` | `Authorization,Content-Type` |
| `SINGLEPAGE_CORS_EXPOSED_HEADERS` | `X-Request-ID` |
| `SINGLEPAGE_CORS_ALLOW_CREDENTIALS` | `false` |
| `SINGLEPAGE_CORS_MAX_AGE` | `0s` |

List values are comma-separated. Wildcard origins cannot be combined with
credentials.
