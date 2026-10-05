# MicDrop — Backend

Go HTTP API for MicDrop, a crowdsourced comedy platform (setups, punchlines, community voting). Backed by PostgreSQL.

- Router: `go-chi/chi/v5` · DB: `pgx/v5` + `sqlx` · Migrations: goose-style SQL
- Auth: Google OAuth with JWT sessions (72h) + short-lived onboarding tokens (15m)
- Voting: idempotent `PUT .../vote` endpoints, server-computed `viewer_vote`, author's auto-upvote on create, karma excludes self-votes

Full endpoint reference: [`docs/ApiContract.md`](docs/ApiContract.md).
Technical design: [`docs/tdd.md`](docs/tdd.md). Product spec: [`docs/prd.md`](docs/prd.md).

## Quickstart

Prerequisites: Go 1.27+, Docker, [`goose`](https://github.com/pressly/goose) on `PATH`.

```bash
# 1. Start Postgres
make docker-up

# 2. Migrate (needs GOOSE_DRIVER=postgres and GOOSE_DBSTRING set)
make db-up

# 3. Configure
# Create .env (see Configuration below; at minimum JWT_SECRET and the
# Google OAuth credentials for login to work)

# 4. Run
make run               # or: go run cmd/server/main.go
```

Useful targets (`make`): `docker-up`, `docker-down`, `docker-logs`, `db-up`,
`db-down`, `db-status`, `db-reset` (drop + migrate fresh — dev only),
`db-create name=...`, `run`, `build`, `test`, `test-cover`.

## Configuration

All via environment (see `internal/config/config.go`):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `3000` | HTTP listen port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/micdrop?sslmode=disable` | Postgres DSN |
| `ALLOWED_ORIGINS` | `*` | CORS origins (comma-separated) |
| `LOG_LEVEL` | `info` | `slog` level |
| `GOOGLE_CLIENT_ID` | — | OAuth client id |
| `GOOGLE_CLIENT_SECRET` | — | OAuth client secret |
| `GOOGLE_REDIRECT_URL` | `http://localhost:3000/api/v1/auth/google/callback` | OAuth callback |
| `JWT_SECRET` | `supersecretjwtkey` | HMAC secret for session/onboarding tokens  (set in prod!) |
| `FRONTEND_URL` | `http://localhost:5173/auth/callback` | Where OAuth redirects land |

## Layout

| Path | Responsibility |
|---|---|
| `cmd/server` | Startup, config, DB connect, HTTP server, graceful shutdown |
| `internal/config` | Env-based config |
| `internal/db` | pgxpool + sqlx setup |
| `internal/router` | Chi routes, `RequireAuth`/`OptionalAuth`, CORS, request logging |
| `internal/handlers` | HTTP decode/validate/encode, status mapping |
| `internal/middleware` | Session JWT auth (`RequireAuth`, `OptionalAuth`) |
| `internal/repository` | SQL + transactions (`SetPromptVote`/`SetResponseVote` live here) |
| `internal/models` | API/DB structs (`viewer_vote` is a joined read-time field, not a column) |
| `migrations` | Schema. Applied migrations are immutable — change the schema with **new** files only |

## Voting in 30 seconds

- `PUT /api/v1/prompts/{id}/vote` and `PUT /api/v1/responses/{id}/vote` take
  `{"vote": "upvote"|"downvote"|"none"}` — the desired end state.
- Each vote runs one transaction: `SELECT ... FOR UPDATE` the parent,
  diff old vs new (`delta = new − old`), delete/upsert the vote row, apply
  `delta` to the counter and to author karma unless voter == author.
- Reads include `viewer_vote` for the caller (`null` when anonymous).

## Testing

```bash
make test            # go test ./...
```

Handler tests use an in-memory repository + `httptest`; middleware tests cover
session/onboarding/expired tokens. Repository integration tests against real
Postgres (Testcontainers) are still on the roadmap — see `docs/tdd.md`.
