# MicDrop — Technical Design Document

## 1. Overview

MicDrop backend is a Go HTTP API backed by PostgreSQL. It uses Chi for routing, `sqlx` over `pgx` for database access, goose-style SQL migrations, and `slog` for structured JSON logging.

The MVP is API-only. Identity is mocked through `user_id` in request bodies and `X-User-ID` for votes.

## 2. Architecture

```text
Client
  |
  v
Chi Router
  |
  v
Handlers
  |
  v
Repository
  |
  v
sqlx + pgxpool
  |
  v
PostgreSQL
```

Key packages:

| Package | Responsibility |
|---|---|
| `cmd/server` | Process startup, config load, DB connect, HTTP server, graceful shutdown |
| `internal/config` | Environment-based configuration |
| `internal/db` | pgxpool + sqlx connection setup |
| `internal/router` | Chi routes, middleware, CORS, request logging |
| `internal/handlers` | HTTP decoding/encoding, status codes, validation |
| `internal/repository` | SQL queries, transactions, DB error mapping |
| `internal/models` | API/database structs |
| `migrations` | PostgreSQL schema |

## 3. Technology Stack

- Language: Go
- Router: `go-chi/chi/v5`
- Middleware: Chi middleware, CORS
- Database: PostgreSQL
- Driver: `pgx/v5` with `pgxpool`
- SQL helper: `sqlx`
- Migrations: goose-style SQL files
- Logging: `log/slog` JSON handler
- Config: environment variables

## 4. Data Model

### users

```sql
CREATE TABLE users (
    user_id        VARCHAR(50) PRIMARY KEY,
    user_name      VARCHAR(50) NOT NULL,
    prompt_score   INTEGER     NOT NULL DEFAULT 0,
    response_score INTEGER     NOT NULL DEFAULT 0,
    total_score    INTEGER     NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### prompts

```sql
CREATE TABLE prompts (
    post_id        BIGSERIAL    PRIMARY KEY,
    user_id        VARCHAR(50)  NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    body           VARCHAR(280) NOT NULL,
    prompt_upvotes INTEGER      NOT NULL DEFAULT 0,
    response_count INTEGER      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### responses

```sql
CREATE TABLE responses (
    response_id      BIGSERIAL    PRIMARY KEY,
    post_id          BIGINT       NOT NULL REFERENCES prompts(post_id) ON DELETE CASCADE,
    user_id          VARCHAR(50)  NOT NULL REFERENCES users(user_id)  ON DELETE CASCADE,
    body             VARCHAR(280) NOT NULL,
    response_upvotes INTEGER      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### prompt_votes

```sql
CREATE TABLE prompt_votes (
    user_id    VARCHAR(50) NOT NULL REFERENCES users(user_id)   ON DELETE CASCADE,
    post_id    BIGINT      NOT NULL REFERENCES prompts(post_id) ON DELETE CASCADE,
    vote_type  VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);
```

### response_votes

```sql
CREATE TABLE response_votes (
    user_id     VARCHAR(50) NOT NULL REFERENCES users(user_id)       ON DELETE CASCADE,
    response_id BIGINT      NOT NULL REFERENCES responses(response_id)   ON DELETE CASCADE,
    vote_type   VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, response_id)
);
```

## 5. Identity and Auth

MVP identity is not authenticated.

- `POST /users` receives `user_id` in the JSON body.
- `POST /prompts` receives `user_id` in the JSON body.
- `POST /prompts/{postID}/responses` receives `user_id` in the JSON body.
- Vote endpoints read `X-User-ID` header.

Future Phase 2 should replace this with JWT/session auth and use a trusted `user_id` from middleware.

## 6. API Routing

Router mounts all domain routes under `/api/v1`.

```text
GET  /healthz

POST /api/v1/users
GET  /api/v1/users/{userID}

POST /api/v1/prompts
GET  /api/v1/prompts
GET  /api/v1/prompts/{postID}

POST /api/v1/prompts/{postID}/upvote
POST /api/v1/prompts/{postID}/downvote

POST /api/v1/prompts/{postID}/responses
GET  /api/v1/prompts/{postID}/responses

POST /api/v1/responses/{responseID}/upvote
POST /api/v1/responses/{responseID}/downvote
```

Middleware order:

1. `RequestID`
2. `Recoverer`
3. `Timeout(30s)`
4. CORS
5. Request logger

## 7. Transactions and Consistency

### Create Response

Repository starts a transaction:

1. Insert into `responses`.
2. Update `prompts.response_count = response_count + 1`.
3. Commit.

Foreign key errors (`23503`):
- `responses_user_id_fkey` → `ErrUserNotFound`.
- Otherwise → `ErrPromptNotFound`.

### Vote on Prompt

Transaction:

1. Insert into `prompt_votes`.
2. Update `prompts.prompt_upvotes`.
3. Update author `users.prompt_score` and `users.total_score`.
4. Commit.
5. Fetch updated prompt outside transaction.

Duplicate vote is detected through PostgreSQL unique violation `23505` and mapped to `ErrAlreadyVoted`.
Foreign key violation `23503` is mapped to `ErrPromptNotFound`.

### Vote on Response

Transaction:

1. Insert into `response_votes`.
2. Update `responses.response_upvotes`.
3. Update author `users.response_score` and `users.total_score`.
4. Commit.
5. Fetch updated response.

Duplicate vote is detected through `23505` and mapped to `ErrAlreadyVoted`.
Foreign key violation `23503` is mapped to `ErrResponseNotFound`.

## 8. Error Handling

Repository errors:

| Error | HTTP |
|---|---|
| `ErrPromptNotFound` | 404 |
| `ErrResponseNotFound` | 404 |
| `ErrUserNotFound` | 404 |
| `ErrAlreadyVoted` | 409 |
| `ErrInvalidVoteType` | 400 |
| default | 500 |

Current error body:

```json
{
  "error": "prompt not found"
}
```

## 9. Configuration

Environment variables:

| Variable | Default |
|---|---|
| `PORT` | `3000` |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/micdrop?sslmode=disable` |
| `ALLOWED_ORIGINS` | `*` |
| `AUTO_MIGRATE` | `true` |
| `LOG_LEVEL` | `info` |

`AUTO_MIGRATE` is currently loaded but not used.

## 10. Observability

- Structured JSON logs via `slog`.
- One log line per request:
  - method
  - path
  - status
  - bytes
  - duration_ms
  - request_id
- Health endpoint: `GET /healthz`.

## 11. Migrations

Migrations are goose-style SQL files:

- `20260926124313_users.sql`
- `20260926125644_prompts.sql`
- `20260926125846_responses.sql`
- `20260926130234_prompt_votes.sql`
- `20260926130256_response_votes.sql`

Run migrations with goose or equivalent before starting the server.

## 12. Testing Strategy

Recommended:

- Repository integration tests against real PostgreSQL via Testcontainers.
- Handler tests with `httptest`.
- Transaction tests for:
  - duplicate votes
  - missing foreign keys
  - counter updates
  - score updates
- Migration up/down tests.
- Race tests for concurrent voting.

## 13. Known Gaps / Bugs

1. `VoteOnResponse` selects `u.user_name`, but `models.Response` has no `UserName` field. `sqlx` may fail to scan the extra column.
2. `GetLeaderboard` exists but no route exposes it.
3. `ErrSelfVote` is defined but unused.
4. `AUTO_MIGRATE` is unused.
5. `internal/middleware` package exists but is empty.
6. No request body length validation; DB `VARCHAR(280)` errors become `500`.
7. `CreatePrompt` does not map missing `user_id` to `ErrUserNotFound` (returns `500`).
8. Vote foreign-key errors (`23503`) cannot distinguish missing voter from missing target, mapping both to `ErrPromptNotFound` / `ErrResponseNotFound`.
9. List endpoints do not return `total_count`.
10. Only `newest` and `top` prompt sorting are implemented.
11. No `GET /responses/{responseID}`.
12. No self-vote prevention despite PRD/error definition.
13. No authentication; `X-User-ID` is trusted.

## 14. Recommended Roadmap

### Phase 1 fixes

- Fix `VoteOnResponse` scan mismatch.
- Add body length validation.
- Map missing voter FK errors correctly.
- Add `GET /responses/{responseID}`.
- Return `total_count` for list endpoints.
- Wire `GetLeaderboard` to `GET /leaderboard`.
- Add self-vote prevention.

### Phase 2

- Real Google OAuth and JWT session implementation.
- User profile pages (`GET /users/{userID}/prompts`, `GET /users/{userID}/responses`).
- Vote changes/retractions.
- Hot/Best/Controversial ranking.
- Rate limiting.
- Moderation.