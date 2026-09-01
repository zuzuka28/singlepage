# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend

WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html svelte.config.js tsconfig.json vite.config.ts ./
COPY web ./web
RUN npm run build


FROM golang:1.24-alpine AS backend

RUN apk add --no-cache build-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal
COPY --from=frontend /src/web/dist ./web/dist

RUN CGO_ENABLED=1 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/mindrop \
    .


FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S mindrop \
    && adduser -S -G mindrop mindrop \
    && mkdir -p /data \
    && chown mindrop:mindrop /data

WORKDIR /app

COPY --from=backend --chown=mindrop:mindrop /out/mindrop ./mindrop

USER mindrop

EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1

ENTRYPOINT ["/app/mindrop"]
CMD ["-listen", ":8080", "-db", "/data/data.db"]
