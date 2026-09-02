# Singlepage

Singlepage is a self-hosted application for personal notes organized as a
nested outline. It is suitable for plans, study notes, drafts, and small
knowledge bases.

Page contents are encrypted on the user's device before being sent to the
server. The server stores only encrypted data and never receives the password
or readable note contents.

## Features

- nested note structure;
- search within the opened page;
- protected links for sharing;
- password and access-link rotation;
- Markdown import and export;
- light and dark themes;
- browser and desktop versions.

Mobile applications are not currently supported. A lost password cannot be
recovered, so keep it in a safe place.

## Installation and startup

### Docker Compose

This is the recommended way to run Singlepage. Git, Docker, and Docker Compose
are required.

```bash
git clone https://github.com/zuzuka28/singlepage.git
cd singlepage
docker compose -f deploy/compose/standalone/compose.yaml up -d --build
```

Open <http://localhost:8080> after startup. Application data is preserved
between restarts.

To stop the application:

```bash
docker compose -f deploy/compose/standalone/compose.yaml down
```

### Public deployment with HTTPS

Point your domain to the server and allow incoming connections on ports 80 and
443, then run:

```bash
DOMAIN=notes.example.com \
docker compose -f deploy/compose/caddy/compose.yaml up -d --build
```

The application will be available at `https://notes.example.com`. The HTTPS
certificate is issued and renewed automatically.

### Running from source

Node.js, Go, and Make are required.

```bash
git clone https://github.com/zuzuka28/singlepage.git
cd singlepage
make install
make build
./singlepage
```

By default, the application is available at <http://127.0.0.1:8080>, and its
data is stored in `data.db` in the current directory.

### Desktop application

Build and run the desktop version from source:

```bash
make build-app
make run-app
```

Desktop data is stored in the current user's standard application-data
directory.

## Configuration

Configuration is provided through environment variables before startup. The
main settings are:

| Variable | Default | Purpose |
| --- | --- | --- |
| `SINGLEPAGE_HTTP_LISTEN` | `127.0.0.1:8080` | Application address and port |
| `SINGLEPAGE_SQLITE_DSN` | `data.db` | Data storage location |
| `SINGLEPAGE_SQLITE_MAX_BYTES` | `536870912` | Maximum storage size; `0` disables the limit |
| `SINGLEPAGE_PAGE_MAX_PAGES` | `100000` | Maximum number of pages; `0` disables the limit |

Example using a different address and data file:

```bash
SINGLEPAGE_HTTP_LISTEN=0.0.0.0:3000 \
SINGLEPAGE_SQLITE_DSN=/var/lib/singlepage/data.db \
./singlepage
```

To change the external Docker Compose port:

```bash
PORT=3000 docker compose -f deploy/compose/standalone/compose.yaml up -d --build
```

A complete configuration example is available in `.env.example`. The
application does not load this file automatically; pass the required values
through the environment when starting it.
