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
go build -o mindrop .
./mindrop -listen :8080 -db ./data.db
```

The resulting Go binary serves the embedded Svelte build and stores only opaque ciphertext, KDF salt, revision, and a hash of the write capability in SQLite.

## Docker

Build and start the application in the background:

```bash
docker compose up -d --build
```

Mindrop will be available at <http://localhost:8080>. SQLite data is kept in
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
