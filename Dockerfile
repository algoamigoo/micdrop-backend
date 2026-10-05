# ---- builder ----
FROM golang:1.27-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# server binary + goose for startup migrations
RUN CGO_ENABLED=0 GOOS=linux go build -o /micdrop ./cmd/server && \
    GOBIN=/tmp/goosebin go install github.com/pressly/goose/v3/cmd/goose@v3.28.0 && \
    cp /tmp/goosebin/goose /goose

# ---- runner ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /micdrop /app/micdrop
COPY --from=builder /goose /usr/local/bin/goose
COPY migrations ./migrations
EXPOSE 3000
# Railway/Render inject PORT and DATABASE_URL at runtime.
# Migrate first (GOOSE_DBSTRING falls back to DATABASE_URL), then start.
# A failed migration aborts the boot: never serve traffic against a stale schema.
CMD ["sh", "-c", "export GOOSE_DRIVER=${GOOSE_DRIVER:-postgres}; export GOOSE_DBSTRING=${GOOSE_DBSTRING:-$DATABASE_URL}; /usr/local/bin/goose -dir ./migrations up || exit 1; exec /app/micdrop"]
