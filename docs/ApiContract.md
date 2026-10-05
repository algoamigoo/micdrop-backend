# MicDrop API Contract

**Base URL:** `http://localhost:3000/api/v1`
**Health check:** `GET /healthz`

## 1. Response Envelope

Successful JSON responses are wrapped in `data`:

```json
{
  "data": {
    "post_id": 101,
    "user_id": "rudra",
    "body": "Things you don't want to hear from your surgeon",
    "prompt_upvotes": 23,
    "response_count": 45,
    "viewer_vote": "upvote",
    "created_at": "2026-09-20T12:00:00Z",
    "updated_at": "2026-09-20T12:00:00Z"
  }
}
```

Errors are returned as:

```json
{
  "error": "prompt not found"
}
```

The current implementation does **not** return machine-readable error codes.

## 2. Authentication / Identity

Auth is Google OAuth + JWT sessions. There are two token purposes:

| Token | `purpose` claim | Lifetime | Grants |
|---|---|---|---|
| Session | `session` | 72h | All authenticated endpoints |
| Onboarding | `onboarding` | 15m | Only `POST /auth/complete-signup` |

Flow:

1. `GET /auth/google/login` → redirects to Google consent.
2. `GET /auth/google/callback` → redirects to the frontend:
   - existing user: `?token=<session JWT>`
   - new Google identity: `?onboarding=<onboarding JWT>`
3. New users call `POST /auth/complete-signup` with the onboarding token to pick a `user_id`, and receive a session JWT.

Authenticated requests send the session token:

```http
Authorization: Bearer <session JWT>
```

Read endpoints (`GET /prompts`, `GET /prompts/{postID}`, `GET /prompts/{postID}/responses`,
`GET /users/{userID}[/prompts|/responses]`) are public but viewer-aware:
when a valid session token is present they include the caller's
`viewer_vote`; otherwise `viewer_vote` is `null`. Invalid tokens on reads
are ignored (anonymous view), never rejected.

## 3. Endpoints

| Endpoint | Method | Auth | Request | Response |
|---|---|---|---|---|
| `/auth/google/login` | GET | None (redirect) | — | `307` to Google, sets the `micdrop_oauth_state` cookie |
| `/auth/google/callback` | GET | None (redirect) | Query `code`, `state` | `307` to frontend with `?token=` or `?onboarding=` |
| `/auth/complete-signup` | POST | Onboarding JWT | `{ user_id }` | `200` `{ token, user }` |
| `/auth/me` | GET | Session JWT | — | `200` User |
| `/users/me` | PATCH | Session JWT | `{ user_name?, bio?, links? }` | `200` User |
| `/users/{userID}` | GET | Optional | Path `userID` | `200` `{ user, stats, follows }` |
| `/users/{userID}/prompts` | GET | Optional | Query `limit`, `offset` | `200` Prompt[] |
| `/users/{userID}/responses` | GET | Optional | Query `limit`, `offset` | `200` Response[] |
| `/users/{userID}/followers` | GET | Optional | Query `limit`, `offset` | `200` User[] |
| `/users/{userID}/following` | GET | Optional | Query `limit`, `offset` | `200` User[] |
| `/users/{userID}/follow` | PUT | Session JWT | — | `204` (idempotent) |
| `/users/{userID}/follow` | DELETE | Session JWT | — | `204` (idempotent) |
| `/prompts` | POST | Session JWT | `{ body }` | `201` Prompt |
| `/prompts` | GET | Optional | Query `sort`, `limit`, `offset` | `200` Prompt[] |
| `/prompts/{postID}` | GET | Optional | Path `postID` | `200` Prompt |
| `/prompts/{postID}` | PATCH | Session JWT (author) | `{ body }` | `200` Prompt |
| `/prompts/{postID}` | DELETE | Session JWT (author) | — | `204` |
| `/prompts/{postID}/vote` | PUT | Session JWT | `{ vote }` | `200` Prompt |
| `/prompts/{postID}/responses` | POST | Session JWT | `{ body }` | `201` Response |
| `/prompts/{postID}/responses` | GET | Optional | Query `limit`, `offset` | `200` Response[] |
| `/responses/{responseID}` | PATCH | Session JWT (author) | `{ body }` | `200` Response |
| `/responses/{responseID}` | DELETE | Session JWT (author) | — | `204` |
| `/responses/{responseID}/vote` | PUT | Session JWT | `{ vote }` | `200` Response |
| `/healthz` | GET | None | None | `200 { "status": "ok" }` |

## 4. Data Models

### User

```json
{
  "user_id": "rudra",
  "user_name": "comedy_fan_42",
  "bio": "Punchlines before breakfast.",
  "links": [{ "type": "github", "url": "https://github.com/..." }],
  "prompt_score": 42,
  "response_score": 157,
  "total_score": 199,
  "created_at": "2026-09-01T10:30:00Z",
  "updated_at": "2026-09-26T15:45:00Z"
}
```

`prompt_score` / `response_score` are net karma **excluding self-votes**;
`total_score` is their sum.

`links` is `null` when the user has never set any (the column is nullable), and an
array once set — including `[]` after an explicit clear via `PATCH /users/me`.
Clients must treat `null` as "no links".

### Prompt

```json
{
  "post_id": 101,
  "user_id": "rudra",
  "body": "Things you don't want to hear from your surgeon",
  "prompt_upvotes": 23,
  "response_count": 45,
  "viewer_vote": "upvote",
  "edited": false,
  "created_at": "2026-09-20T12:00:00Z",
  "updated_at": "2026-09-20T12:00:00Z"
}
```

`viewer_vote` is `"upvote"`, `"downvote"`, or `null` (anonymous, or the
viewer hasn't voted). It is always populated from the server — clients
must not cache it locally.

`edited` is derived server-side from `updated_at > created_at`: the body has been
changed since creation. It is `false` for content that was only soft-deleted, because
deleting does not touch `updated_at`.

> Current implementation does not join `user_name` into Prompt responses. Clients should resolve `user_id` through `/users/{userID}` if needed.

### Response

```json
{
  "response_id": 501,
  "post_id": 101,
  "user_id": "Jani",
  "body": "Don't worry, I've done this a thousand times... on a simulator.",
  "response_upvotes": 67,
  "viewer_vote": null,
  "edited": true,
  "created_at": "2026-09-20T13:15:00Z",
  "updated_at": "2026-09-20T13:15:00Z"
}
```

### UserStats

```json
{
  "prompt_count": 12,
  "response_count": 34
}
```

Returned alongside the user from `GET /users/{userID}` as
`{ "user": User, "stats": UserStats, "follows": FollowCounts }`.

### FollowCounts

```json
{
  "followers_count": 12,
  "following_count": 4,
  "is_following": false
}
```

`is_following` reflects the caller and is always `false` for anonymous reads.

## 5. Endpoint Details

### GET `/auth/google/login`

Sets a `micdrop_oauth_state` cookie (httpOnly, `SameSite=Lax`, 10 minutes) holding a
256-bit random value and sends the same value to Google as `state`. The callback compares
the two and clears the cookie, so each login attempt is single-use; a mismatch is a `400`.
This is what prevents login CSRF.

Redirects (`307`) to Google's consent page.

### GET `/auth/google/callback`

Exchanges `?code=` with Google, then redirects (`307`) to `FRONTEND_URL`:

- Known Google account → `?token=<session JWT>`
- New Google account → `?onboarding=<onboarding JWT>` (frontend routes to username onboarding)

Errors:

- `400` missing `code`.
- `401` token exchange or userinfo fetch failed.

### POST `/auth/complete-signup`

Creates the user row with the chosen username. Requires the onboarding JWT:

```http
Authorization: Bearer <onboarding JWT>
```

```json
{
  "user_id": "punchline_pro"
}
```

Notes:

- `user_id` must be 3–30 chars, `[a-zA-Z0-9_-]`, not reserved (`me`, `admin`, `api`, `auth`, `root`, `micdrop`, `support`).
- `user_name` defaults to `user_id`.

Response `200`:

```json
{
  "data": {
    "token": "<session JWT>",
    "user": { "...": "..." }
  }
}
```

Errors:

- `401` missing/invalid/expired onboarding token (session tokens rejected).
- `400` invalid `user_id`.
- `409` username already taken.

### GET `/auth/me`

Response `200`: the caller's User object.

Errors:

- `401` missing/invalid token.

### PATCH `/users/me`

Partial profile update. All fields optional; `links` accepts at most 3 entries
(`github`, `twitter`, `youtube`, `instagram`, `linkedin`, `website` with
`http(s)://` URLs). An empty array clears links.

Response `200`: updated User.

### GET `/users/{userID}`

Response `200`: `{ "user": User, "stats": UserStats }`.

Errors:

- `404` user not found.

### GET `/users/{userID}/prompts` & `/users/{userID}/responses`

Paginated items by author, each with the caller's `viewer_vote` when authenticated.

| Param | Default | Notes |
|---|---|---|
| `limit` | `10` | Must be `1..50`; invalid values ignored |
| `offset` | `0` | Must be `>= 0`; invalid values ignored |

### POST `/prompts`

Request:

```json
{
  "body": "Things you don't want to hear from your surgeon"
}
```

The author is taken from the session token, and the author's upvote is
recorded in the same transaction — new prompts start at `prompt_upvotes: 1`
with `viewer_vote: "upvote"` for the author. The self-vote adds no karma.

Response `201`: Prompt object.

Errors:

- `401` missing/invalid token.
- `400` invalid request body or missing `body`.

### GET `/prompts`

Query parameters:

| Param | Default | Notes |
|---|---|---|
| `sort` | `newest` | `newest` or `top`; other values fall back to newest |
| `limit` | `10` | Must be `1..50`; invalid values ignored |
| `offset` | `0` | Must be `>= 0`; invalid values ignored |

Response `200`: Array of Prompt objects with `viewer_vote`. No `total_count` is returned.

### GET `/prompts/{postID}`

Response `200`: single Prompt object with `viewer_vote`.

Errors:

- `400` invalid `postID`.
- `404` prompt not found.

### PUT `/prompts/{postID}/vote`

Idempotent vote. The client sends the **desired end state**:

```json
{
  "vote": "upvote"
}
```

`vote` is `"upvote"`, `"downvote"`, or `"none"`. Clicking the current
direction sends `"none"` (toggle off); clicking the opposite direction
flips. Retries and double-clicks converge on the same state.

Response `200`: updated Prompt object including `viewer_vote`.

Errors:

- `401` missing/invalid token.
- `400` invalid `postID`, invalid body, or invalid `vote` value.
- `404` prompt not found.

Authors may vote on their own items. Self-votes move the counter but never karma.

### POST `/prompts/{postID}/responses`

Request:

```json
{
  "body": "Don't worry, I've done this a thousand times... on a simulator."
}
```

Like prompts, the author's upvote is recorded atomically — new responses
start at `response_upvotes: 1`.

Response `201`: Response object.

Errors:

- `401` missing/invalid token.
- `400` invalid request body or missing `body`.
- `404` prompt not found (or voter user row missing).

### GET `/prompts/{postID}/responses`

Query parameters:

| Param | Default | Notes |
|---|---|---|
| `limit` | `20` | Must be `1..100`; invalid values ignored |
| `offset` | `0` | Must be `>= 0`; invalid values ignored |

Response `200`: Array of Response objects with `viewer_vote`, ordered by `response_upvotes DESC, created_at ASC`. No `total_count` is returned.

### PUT `/responses/{responseID}/vote`

Same semantics as prompt voting. Response `200`: updated Response object
including `viewer_vote`.

Errors:

- `401` missing/invalid token.
- `400` invalid `responseID`, invalid body, or invalid `vote` value.
- `404` response not found.

### PATCH `/prompts/{postID}`

Edit the body of a prompt you authored. Same body rules as creation (non-empty, <= 280
characters). Votes, counters and `response_count` are untouched by an edit.

Response `200`: updated Prompt.

Errors:

- `400` invalid or over-long `body`
- `403` not the author
- `404` no such prompt (or already deleted)

### DELETE `/prompts/{postID}`

Soft-delete a prompt **and all of its responses**. Votes, counters and karma are left
exactly as they were: the upvotes were earned, and a responder's karma is not the
prompt author's to revoke. Rows are never removed, so the decision is reversible and
auditable — see [`edit-delete-decisions.md`](./edit-delete-decisions.md).

The accepted trade-off: a disliked prompt can be deleted to shed its downvotes, and a
liked one deleted and reposted to farm karma again.

Response `204` empty.

Errors: `403` not the author · `404` no such prompt.

### PATCH `/responses/{responseID}`

Edit the body of a response you authored. Same rules and errors as the prompt variant.

Response `200`: updated Response.

### DELETE `/responses/{responseID}`

Soft-delete a response. Votes and karma are untouched; `response_count` on the parent
prompt *is* decremented, because it counts visible children rather than scoring anything.

Response `204` empty.

### PUT / DELETE `/users/{userID}/follow`

Idempotent follow / unfollow. Self-follow is a `400`; following twice is fine, and
unfollowing someone you do not follow still returns `204`.

Response `204` empty.

### GET `/users/{userID}/followers` & `/users/{userID}/following`

Public lists of User, newest follow first. Same pagination rules as the other lists
(`limit` 1-100, default 20; `offset` >= 0), so `400` on a bad value.

## 6. Error Status Mapping

| HTTP Status | Condition |
|---|---|
| `400` | Invalid JSON, missing required field, invalid ID format, invalid `vote` value, invalid profile field |
| `401` | Missing/invalid token on protected endpoints |
| `404` | Prompt, response, or user not found |
| `403` | Authenticated but not the author (edit/delete of someone else's content) |
| `409` | Username already taken (`POST /auth/complete-signup`) |
| `500` | Internal server error |

Votes no longer return `409`: setting the same state twice is a no-op.

## 7. Planned / Unimplemented Endpoints

These are not implemented in the current router:

- `GET /leaderboard` (Repository method exists, route not wired)
- `GET /responses/{responseID}` (Repository method exists, route not wired)
- `GET /prompts?sort=hot|best|controversial`
- Paginated responses with `total_count`
- Error responses with machine-readable `code`
- Notifications of any kind (deferred; no notifications table exists)
- Restoring a soft-deleted prompt/response through the API
- Edit history / "edited by" diffs
