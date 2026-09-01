# syntax=docker/dockerfile:1

FROM node:24-alpine@sha256:e67514e5d0f6c46656005e1b693b2ec9d52e80b641307de684d4a015ba7a4eaf AS frontend

WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html svelte.config.js tsconfig.json vite.config.ts ./
COPY web ./web
RUN npm run build


FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS backend

RUN apk add --no-cache build-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal
COPY --from=frontend /src/internal/handler/frontend/dist ./internal/handler/frontend/dist

RUN CGO_ENABLED=1 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/singlepage \
    .


FROM alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce AS runtime

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S singlepage \
    && adduser -S -G singlepage singlepage \
    && mkdir -p /data \
    && chown singlepage:singlepage /data

WORKDIR /app

COPY --from=backend --chown=singlepage:singlepage /out/singlepage ./singlepage

USER singlepage

EXPOSE 8080
VOLUME ["/data"]

ENV SINGLEPAGE_HTTP_LISTEN=:8080 \
    SINGLEPAGE_METRICS_LISTEN=127.0.0.1:9090 \
    SINGLEPAGE_SQLITE_DSN=/data/data.db

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1

ENTRYPOINT ["/app/singlepage"]
