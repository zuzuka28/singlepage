# Mindrop

A zero-knowledge infinite outline.

## Development

```bash
npm install
npm run dev          # frontend on :5173, proxies /api to :8080
go run .             # API on :8080
```

## Production

```bash
npm run build
go build -o mindrop . # use Go 1.27 or newer
./mindrop -listen 127.0.0.1:8080 -db ./data.db
```

The application server speaks plain HTTP and binds to loopback by default. For
remote access, use the Caddy stack below or another HTTPS reverse proxy. Do not
publish the application port directly on an untrusted network.

The resulting Go binary serves the embedded Svelte build and stores only opaque ciphertext, KDF salt, revision, and a hash of the write capability in SQLite.

## Docker

Build and start the application in the background:

```bash
docker compose up -d --build
```

Mindrop will be available at <http://localhost:8080> and is bound only to the
host loopback interface. SQLite data is kept in
the named `mindrop-data` volume and survives container recreation.

To publish the service on another host port:

```bash
PORT=3000 docker compose up -d --build
```

Stop the application without deleting its data:

```bash
docker compose down
```

To also delete the database volume, run `docker compose down -v`.

### Docker with automatic HTTPS

Point the domain's `A`/`AAAA` DNS records to the server, allow inbound TCP
ports 80 and 443, then start the standalone Caddy stack:

```bash
DOMAIN=mindrop.example.com docker compose -f compose.caddy.yaml up -d --build
```

Caddy obtains and renews the TLS certificate automatically. Mindrop is only
reachable through Caddy; its port 8080 is not published on the host. Caddy's
certificate and configuration state are persisted in named Docker volumes.

## Security controls

The browser derives the encryption key from the password and the secret URL
fragment. The fragment is not sent in HTTP requests, and the server stores only
opaque ciphertext, salt, revision, timestamps, and hashes of write
capabilities. Changing the password also rotates the write capability; creating
a new access link replaces the page ID, fragment secret, and write capability.

Anyone who obtains the complete access link can download the ciphertext and try
password guesses offline. New passwords must contain at least 16 characters;
use a long, unique passphrase and share the link and
password through separate trusted channels. The fragment can still be exposed
through browser history, screenshots, clipboard managers, or browser
extensions.

The server applies conservative defaults to unauthenticated storage:

- 3 MiB maximum JSON request body;
- 2 MiB maximum encrypted page;
- 100,000 stored pages;
- 512 MiB SQLite logical size;
- one sustained page creation per second per client address, with a burst of 20.

They can be adjusted with `-max-request-bytes`, `-max-page-bytes`,
`-max-pages`, `-max-database-bytes`, `-create-rate`, and `-create-burst`.
Set a database, page-count, or create-rate value to `0` only when an external
control provides the equivalent protection.

The Caddy Compose stack enables `-trust-proxy-headers`; the server then uses the
last address appended to `X-Forwarded-For`. Do not enable this option when an
untrusted client can reach the application port without going through a reverse
proxy that appends the real client address.

Browser responses include a restrictive Content Security Policy, anti-framing,
cross-origin isolation, no-referrer, and no-index headers. HSTS is emitted for
HTTPS requests and by the included Caddy configuration.

### Optional administrative deletion

The deletion API is disabled unless an admin token file is configured. Generate
a file readable only by the service account and start the server with it:

```bash
umask 077
openssl rand -base64 32 > admin-token
./mindrop -admin-token-file ./admin-token
```

Delete an abusive or compromised page without decrypting it:

```bash
curl -X DELETE \
  -H "Authorization: Bearer $(cat admin-token)" \
  http://127.0.0.1:8080/api/admin/pages/PAGE_ID
```

For containers, mount the token as a read-only Docker secret or bind mount and
pass `-admin-token-file /run/secrets/mindrop-admin`. Never put the token directly
in the image, Compose file, environment, URL, or command line.

Stop the HTTPS stack without deleting its data:

```bash
DOMAIN=mindrop.example.com docker compose -f compose.caddy.yaml down
```

#### Local HTTPS

Use `mindrop.localhost` to run the same stack locally without configuring DNS:

```bash
DOMAIN=mindrop.localhost docker compose -f compose.caddy.yaml up -d --build
```

Caddy uses its local CA for this address. Export its root certificate once:

```bash
DOMAIN=mindrop.localhost docker compose -f compose.caddy.yaml cp \
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
$env:DOMAIN = "mindrop.localhost"
docker compose -f compose.caddy.yaml cp `
  caddy:/data/caddy/pki/authorities/local/root.crt .\caddy-local-root.crt
certutil -addstore root .\caddy-local-root.crt
```

Restart the browser, then open <https://mindrop.localhost>.

## Verification

```bash
npm run check
npm test
npm run test:e2e
go test -race ./...
go vet ./...
```

Playwright uses an installed Google Chrome channel by default. The browser performs all document parsing, indexing, searching, key derivation, encryption, and decryption.
