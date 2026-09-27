# docs/ApiContract.md

# MicDrop API Contract

**Base URL:** `http://localhost:3000/api/v1`  
**Health check:** `GET /healthz`

> Important: the previous contract used `/v1`, `user_name` in vote bodies, nested prompt+responses, `new_upvote_count`, and `/leaderboard`. Those are not implemented by the current router. This contract reflects the current MVP.

## 1. Response Envelope

Successful JSON responses are wrapped in `data`:

```json
{
  "data": {
    "post_id": 101,
    "user_id": "user_123",
    "body": "Things you don't want to hear from your surgeon",
    "prompt_upvotes": 23,
    "response_count": 45,
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

MVP uses mock identity.

- User creation: `user_id` is sent in the JSON body.
- Prompt creation: `user_id` is sent in the JSON body.
- Response creation: `user_id` is sent in the JSON body.
- Voting: `X-User-ID` header is required.

Example:

```http
X-User-ID: user_123
```

There is no token validation.

## 3. Endpoints

| Endpoint | Method | Auth | Request | Response |
|---|---|---|---|---|
| `/users` | POST | Body `user_id` | `{ user_id, user_name? }` | `201` User |
| `/users/{userID}` | GET | None | Path `userID` | `200` User |
| `/prompts` | POST | Body `user_id` | `{ user_id, body }` | `201` Prompt |
| `/prompts` | GET | None | Query `sort`, `limit`, `offset` | `200` Prompt[] |
| `/prompts/{postID}` | GET | None | Path `postID` | `200` Prompt |
| `/prompts/{postID}/upvote` | POST | `X-User-ID` | No body | `200` Prompt |
| `/prompts/{postID}/downvote` | POST | `X-User-ID` | No body | `200` Prompt |
| `/prompts/{postID}/responses` | POST | Body `user_id` | `{ user_id, body }` | `201` Response |
| `/prompts/{postID}/responses` | GET | None | Query `limit`, `offset` | `200` Response[] |
| `/responses/{responseID}/upvote` | POST | `X-User-ID` | No body | `200` Response |
| `/responses/{responseID}/downvote` | POST | `X-User-ID` | No body | `200` Response |
| `/healthz` | GET | None | None | `200 { "status": "ok" }` |

## 4. Data Models

### User

```json
{
  "user_id": "user_123",
  "user_name": "comedy_fan_42",
  "prompt_score": 42,
  "response_score": 157,
  "total_score": 199,
  "created_at": "2026-09-01T10:30:00Z",
  "updated_at": "2026-09-26T15:45:00Z"
}
```

### Prompt

```json
{
  "post_id": 101,
  "user_id": "user_123",
  "body": "Things you don't want to hear from your surgeon",
  "prompt_upvotes": 23,
  "response_count": 45,
  "created_at": "2026-09-20T12:00:00Z",
  "updated_at": "2026-09-20T12:00:00Z"
}
```

> Current implementation does not join `user_name` into Prompt responses. Clients should resolve `user_id` through `/users/{userID}` if needed.

### Response

```json
{
  "response_id": 501,
  "post_id": 101,
  "user_id": "user_456",
  "body": "Don't worry, I've done this a thousand times... on a simulator.",
  "response_upvotes": 67,
  "created_at": "2026-09-20T13:15:00Z",
  "updated_at": "2026-09-20T13:15:00Z"
}
```

## 5. Endpoint Details

### POST `/users`

Create or return an existing user.

Request:

```json
{
  "user_id": "user_123",
  "user_name": "comedy_fan_42"
}
```

Notes:

- `user_id` is required.
- `user_name` is optional; if empty, repository defaults it to `user_id`.
- Existing users are not renamed.

Response `201`:

```json
{
  "data": {
    "user_id": "user_123",
    "user_name": "comedy_fan_42",
    "prompt_score": 0,
    "response_score": 0,
    "total_score": 0,
    "created_at": "2026-09-26T16:00:00Z",
    "updated_at": "2026-09-26T16:00:00Z"
  }
}
```

### GET `/users/{userID}`

Response `200`:

```json
{
  "data": {
    "user_id": "user_123",
    "user_name": "comedy_fan_42",
    "prompt_score": 42,
    "response_score": 157,
    "total_score": 199,
    "created_at": "2026-09-01T10:30:00Z",
    "updated_at": "2026-09-26T15:45:00Z"
  }
}
```

Errors:

- `404` user not found.

### POST `/prompts`

Request:

```json
{
  "user_id": "user_123",
  "body": "Things you don't want to hear from your surgeon"
}
```

Response `201`:

```json
{
  "data": {
    "post_id": 102,
    "user_id": "user_123",
    "body": "Things you don't want to hear from your surgeon",
    "prompt_upvotes": 0,
    "response_count": 0,
    "created_at": "2026-09-26T16:00:00Z",
    "updated_at": "2026-09-26T16:00:00Z"
  }
}
```

Errors:

- `400` invalid request body.
- `400` `user_id` or `body` missing.
- `500` if `user_id` does not exist or DB constraint fails.

### GET `/prompts`

Query parameters:

| Param | Default | Notes |
|---|---|---|
| `sort` | `newest` | `newest` or `top`; other values fall back to newest |
| `limit` | `10` | Must be `1..50`; invalid values ignored |
| `offset` | `0` | Must be `>= 0`; invalid values ignored |

Response `200`:

```json
{
  "data": [
    {
      "post_id": 101,
      "user_id": "user_123",
      "body": "Things you don't want to hear from your surgeon",
      "prompt_upvotes": 23,
      "response_count": 45,
      "created_at": "2026-09-20T12:00:00Z",
      "updated_at": "2026-09-20T12:00:00Z"
    }
  ]
}
```

> No `total_count` is returned by the current implementation.

### GET `/prompts/{postID}`

Response `200`: single Prompt object.

Errors:

- `400` invalid `postID`.
- `404` prompt not found.

### POST `/prompts/{postID}/upvote`

Headers:

```http
X-User-ID: user_456
```

No body.

Response `200`: updated Prompt object.

Errors:

- `401` missing `X-User-ID`.
- `400` invalid `postID`.
- `404` prompt not found.
- `409` user already voted.
- `500` internal error.

### POST `/prompts/{postID}/downvote`

Same as upvote, but records a downvote and decrements the prompt score.

### POST `/prompts/{postID}/responses`

Request:

```json
{
  "user_id": "user_456",
  "body": "Don't worry, I've done this a thousand times... on a simulator."
}
```

Response `201`:

```json
{
  "data": {
    "response_id": 502,
    "post_id": 101,
    "user_id": "user_456",
    "body": "Don't worry, I've done this a thousand times... on a simulator.",
    "response_upvotes": 0,
    "created_at": "2026-09-26T16:05:00Z",
    "updated_at": "2026-09-26T16:05:00Z"
  }
}
```

Errors:

- `400` invalid request body.
- `400` missing `user_id` or `body`.
- `404` user not found.
- `404` prompt not found.

### GET `/prompts/{postID}/responses`

Query parameters:

| Param | Default | Notes |
|---|---|---|
| `limit` | `20` | Must be `1..100`; invalid values ignored |
| `offset` | `0` | Must be `>= 0`; invalid values ignored |

Response `200`:

```json
{
  "data": [
    {
      "response_id": 501,
      "post_id": 101,
      "user_id": "user_456",
      "body": "Don't worry, I've done this a thousand times... on a simulator.",
      "response_upvotes": 67,
      "created_at": "2026-09-20T13:15:00Z",
      "updated_at": "2026-09-20T13:15:00Z"
    }
  ]
}
```

> No `total_count` is returned by the current implementation.

### POST `/responses/{responseID}/upvote`

Headers:

```http
X-User-ID: user_789
```

No body.

Response `200`: updated Response object.

Errors:

- `401` missing `X-User-ID`.
- `400` invalid `responseID`.
- `404` response not found.
- `409` user already voted.
- `500` internal error.

### POST `/responses/{responseID}/downvote`

Same as upvote, but records a downvote and decrements the response score.

## 6. Error Status Mapping

| HTTP Status | Condition |
|---|---|
| `400` | Invalid JSON, missing required field, invalid ID format, invalid vote type |
| `401` | Missing `X-User-ID` on vote endpoint |
| `404` | Prompt, response, or user not found |
| `409` | User already voted on the item |
| `500` | Internal server error |

## 7. Planned Endpoints

These are not implemented in the current router:

- `GET /leaderboard`
- `GET /responses/{responseID}`
- `GET /users/{userID}/prompts`
- `GET /users/{userID}/responses`
- `GET /prompts?sort=hot|best|controversial`
- Paginated responses with `total_count`, `limit`, `offset`
- Error responses with machine-readable `code`