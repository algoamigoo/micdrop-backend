# MicDrop — Technical Design Document

## 1. Overview

MicDrop backend is a Go HTTP API backed by PostgreSQL. It uses Chi for routing, `sqlx` over `pgx` for database access, goose-style SQL migrations, and `slog` for structured JSON logging.

The API is versioned under `/api/v1`. Identity is Google OAuth with JWT
sessions: the session token's `user_id` is the trusted author/voter id on
writes, and public reads are viewer-aware (they include the caller's
`viewer_vote` when a valid session token is present).

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
    google_id      VARCHAR(100) NOT NULL UNIQUE,
    bio            VARCHAR(100),
    links          JSONB,
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

Note: `viewer_vote` (`"upvote" | "downvote" | null`) is not a column. It is
a read-time `LEFT JOIN` of the viewer's vote row onto prompt/response
selects, scanned into a nullable model field.

## 5. Identity and Auth

Auth is Google OAuth with two JWT purposes (see `internal/handlers/auth.go`):

- `GET /auth/google/login` redirects to Google consent.
- `GET /auth/google/callback` exchanges the code: known Google accounts get
  a session JWT (`purpose: "session"`, 72h) via frontend `?token=`; new
  Google accounts get an onboarding JWT (`purpose: "onboarding"`, 15m) via
  `?onboarding=`.
- `POST /auth/complete-signup` (onboarding JWT only) creates the user row
  with the chosen `user_id` and returns a session JWT.

Middleware (`internal/middleware/auth.go`):

- `RequireAuth` — rejects requests without a valid session token (onboarding
  tokens rejected); injects `user_id` into context. Used on all writes.
- `OptionalAuth` — injects `user_id` when a valid session token is present,
  otherwise continues anonymously. Used on public reads so they can include
  the caller's `viewer_vote`.

## 6. API Routing

Router mounts all domain routes under `/api/v1`.

```text
GET  /healthz

GET  /api/v1/auth/google/login
GET  /api/v1/auth/google/callback
POST /api/v1/auth/complete-signup
GET  /api/v1/auth/me                        (auth)

PATCH /api/v1/users/me                      (auth)
GET  /api/v1/users/{userID}                 (optional auth)
GET  /api/v1/users/{userID}/prompts         (optional auth)
GET  /api/v1/users/{userID}/responses       (optional auth)

POST /api/v1/prompts                        (auth)
GET  /api/v1/prompts                        (optional auth)
GET  /api/v1/prompts/{postID}               (optional auth)

PUT  /api/v1/prompts/{postID}/vote          (auth, {"vote": "upvote"|"downvote"|"none"})

POST /api/v1/prompts/{postID}/responses     (auth)
GET  /api/v1/prompts/{postID}/responses     (optional auth)

PUT  /api/v1/responses/{responseID}/vote    (auth, {"vote": "upvote"|"downvote"|"none"})
```

Middleware order:

1. `RequestID`
2. `Recoverer`
3. `Timeout(30s)`
4. CORS
5. Request logger

## 7. Transactions and Consistency

### Create Prompt / Create Response

Repository starts a transaction:

1. Insert into `prompts` / `responses` with the counter preset to `1`.
2. Insert the author's auto-upvote into `prompt_votes` / `response_votes`.
3. (Responses only) bump `prompts.response_count`.
4. Commit, then re-read the row with the author as viewer
   (`viewer_vote: "upvote"`).

The self-vote moves the displayed counter but never karma. Foreign key
errors (`23503`) map `responses_user_id_fkey` → `ErrUserNotFound`,
otherwise → `ErrPromptNotFound`.

### Set Vote on Prompt / Response

Votes are idempotent: the client sends the desired end state
(`upvote` / `downvote` / `none`), and the transaction converges on it:

1. `SELECT user_id ... FOR UPDATE` on the parent row — serializes votes
   per item, returns a real 404 when missing, and yields the author id.
2. Read the voter's old vote (or none); compute `delta = new − old`
   (`upvote=+1`, `downvote=−1`, `none=0`). `delta == 0` skips all writes.
3. Delete the vote row (`none`) or upsert it.
4. Apply `delta` to the denormalized counter.
5. Apply `delta` to author karma (`prompt_score` / `response_score` +
   `total_score`) — skipped when voter and author are the same.
6. Commit, then re-read the row with the voter as viewer.

## 8. Error Handling

Repository errors:

| Error | HTTP |
|---|---|
| `ErrPromptNotFound` | 404 |
| `ErrResponseNotFound` | 404 |
| `ErrUserNotFound` | 404 |
| `ErrUserIDTaken`, `ErrGoogleIDTaken` | 409 |
| `ErrInvalidVoteType`, `ErrInvalidLink` | 400 |
| default | 500 |

Setting the same vote twice is a no-op (no 409 on votes).

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
| `LOG_LEVEL` | `info` |


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
- `20260929090105_userbio.sql` (adds `google_id`, `bio`, `links`)

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

1. `GetLeaderboard` exists but no route exposes it.
2. `GetResponseByID` exists but no `GET /responses/{responseID}` route exposes it.
3. List endpoints do not return `total_count`.
4. Only `newest` and `top` prompt sorting are implemented.
5. No rate limiting.
6. Pre-existing rows have no author auto-vote (only newly created items start at 1).
7. No notifications at all (deferred by decision; no table exists).
8. No restore path for soft-deleted content (the rows are kept, nothing exposes them).
9. Deleting is consequence-free, so delete-and-repost can farm karma (accepted trade-off,
   see `edit-delete-decisions.md`).

## 14. Recommended Roadmap

### Next fixes

- Add body length validation.
- Map missing voter FK errors correctly.
- Add `GET /responses/{responseID}`.
- Return `total_count` for list endpoints.
- Wire `GetLeaderboard` to `GET /leaderboard`.
- Repository integration tests (Testcontainers) + vote race tests.
- Update `docs/ApiContract.md` alongside endpoint changes.

### Later

- Hot/Best/Controversial ranking.
- Rate limiting.
- Moderation.

Testing

Phase 1: Unit Tests (Handlers & Middleware)
Where: internal/handlers/ and internal/middleware/
Tool: Go's standard testing + httptest + Mocks

We already started this, but we need to expand it to cover every edge case.

1. Auth Middleware (internal/middleware/auth_test.go)

Test: Missing Authorization header → 401 Unauthorized
Test: Header is not Bearer <token> (e.g., Basic <token>) → 401 Unauthorized
Test: Invalid JWT signature → 401 Unauthorized
Test: Expired JWT → 401 Unauthorized
Test: Valid JWT → Context contains correct user_id
2. Prompts Handler (internal/handlers/prompts_test.go)

Test CreatePrompt:
Missing JWT → 401
Empty body field → 400 Bad Request
Internal DB error (Mock returns ErrUserNotFound) → 404
Success → 201 Created and response body matches.
Test ListPrompts:
Invalid limit (e.g., -5 or 999) → Server falls back to default 10.
Invalid sort (e.g., sort=cool) → Server falls back to newest.
Success → 200 OK with array.
3. Responses Handler (internal/handlers/responses_test.go)

Test CreateResponse:
Missing JWT → 401
Invalid postID in URL (e.g., abc) → 400 Bad Request
Missing body → 400 Bad Request
DB returns ErrPromptNotFound → 404
Success → 201
Test ListResponses: Pagination logic and success.
4. Votes Handler (internal/handlers/votes_test.go)

Test UpvotePrompt / DownvotePrompt:
Missing JWT → 401
Invalid postID → 400
Mock returns ErrAlreadyVoted → 409 Conflict
Mock returns ErrPromptNotFound → 404
Success → 200
(Repeat for Responses)
5. Users Handler (internal/handlers/users_test.go)

Test ListUserPrompts / ListUserResponses:
Invalid userID format → 400
Success → 200 with list.
Phase 2: Integration Tests (Repository / Database)
Where: internal/repository/
Tool: Go's testing + Testcontainers + Real PostgreSQL

This is critical. Your app uses complex SQL transactions for voting and creating responses. We must test the actual database. Testcontainers will automatically spin up a Docker PostgreSQL container, run your migrations, test the queries, and tear it down.

1. Users Repo (internal/repository/users_test.go)

Test GetOrCreateUser:
Create a new user → User exists in DB with total_score = 0.
Call again with same user_id but different user_name → DB should ignore the new name (no-update on conflict) and return original.
Test GetUserByID:
Get existing user → Returns correct scores.
Get non-existent user → Returns ErrUserNotFound.
2. Prompts & Responses Repo (internal/repository/prompts_test.go, etc.)

Test CreatePrompt:
Valid user ID → Inserts successfully.
Invalid user ID (FK violation) → Returns ErrUserNotFound (This tests the bug fix we discussed earlier!)
Body length > 280 chars → Returns DB error.
Test CreateResponse (Transaction test):
Insert response for valid prompt → Success.
Verify prompts.response_count actually incremented by 1.
Insert response for invalid post_id → Returns ErrPromptNotFound.
Insert response for invalid user_id → Returns ErrUserNotFound.
3. Votes Repo (internal/repository/votes_test.go) - The most critical tests

Test VoteOnPrompt (Transaction test):
Valid vote → Success. Verify prompts.prompt_upvotes increased by 1. Verify author's users.total_score increased by 1.
Duplicate vote → Returns ErrAlreadyVoted. Verify counters did not change (rollback).
Self-vote → Returns ErrSelfVote (once you implement that check).
Test VoteOnResponse:
Same as above, but verify responses.response_upvotes and response_score.
Crucial Test: Verify the SQL scan bug is fixed (that it doesn't try to scan user_name into models.Response anymore).
4. Concurrency / Race Tests

Test VoteOnPrompt under load:
Spin up 50 goroutines trying to upvote the same prompt with the same user_id simultaneously.
Assert that exactly 1 succeeds, and 49 return ErrAlreadyVoted.
Assert the final prompt_upvotes is exactly 1 (not 50).
Phase 3: End-to-End (E2E) Tests (Optional for backend)
Where: cmd/server/main_test.go
Tool: httptest + Real DB

Instead of testing just the handler, you boot the entire router.New() with a real database connection and hit the actual HTTP endpoints.

Example: Hit POST /api/v1/prompts -> Hit GET /api/v1/prompts/1 -> Verify it's there.
(We can skip this if we have solid Phase 1 and Phase 2 tests, as they cover the same paths).
