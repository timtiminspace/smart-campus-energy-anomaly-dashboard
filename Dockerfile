# syntax=docker/dockerfile:1.7

# ── builder ───────────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Pull in the rest of the source needed to build the server binary.
COPY cmd ./cmd
COPY internal ./internal
COPY database ./database

# Pure-Go SQLite driver (glebarez/go-sqlite) — CGO disabled for a static binary.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ── runtime ───────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S app && adduser -S app -G app \
 && mkdir -p /app/data && chown -R app:app /app

WORKDIR /app
COPY --from=builder /out/server /app/server

USER app

ENV PORT=8080 \
    DB_PATH=/app/data/campus.db \
    ML_URL=http://ml:8000

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/readings?limit=1 >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
