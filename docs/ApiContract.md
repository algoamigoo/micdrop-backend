# MicDrop API Contract

**Base URL:** `localhost:3000/v1`

---

## REST Endpoints Overview

| Endpoint | Method | Purpose | Request Body | Response |
| --- | --- | --- | --- | --- |
| `/prompts` | GET | List all prompts with pagination and sorting | - | Array of prompts, total_count, limit, offset |
| `/prompts` | POST | Create a new prompt | `{ user_name, body }` | Created prompt object |
| `/prompts/{post_id}` | GET | Get a specific prompt with all responses | - | Prompt object + array of responses |
| `/responses` | POST | Submit a response to a prompt | `{ user_name, post_id, body }` | Created response object |
| `/responses/{response_id}` | GET | Get a specific response | - | Response object |
| `/prompts/{post_id}/upvote` | POST | Upvote a prompt | `{ user_name }` | `{ success, new_upvote_count }` |
| `/prompts/{post_id}/downvote` | POST | Downvote a prompt | `{ user_name }` | `{ success, new_upvote_count }` |
| `/responses/{response_id}/upvote` | POST | Upvote a response | `{ user_name }` | `{ success, new_upvote_count }` |
| `/responses/{response_id}/downvote` | POST | Downvote a response | `{ user_name }` | `{ success, new_upvote_count }` |
| `/users/{user_name}` | GET | Get user profile and stats | - | User object with recent prompts/responses |
| `/leaderboard` | GET | Get ranked users by total score | - | Array of users ranked by karma |

---

## Query Parameters

### `GET /prompts`
- `sort` (default: `hot`): `hot`, `top`, `newest`, `controversial`
- `limit` (default: `20`): Results per page (1-100)
- `offset` (default: `0`): Pagination offset

---

## Data Models

### User
```json
{
  "user_id": 1,
  "user_name": "comedy_fan_42",
  "prompt_score": 42,
  "response_score": 157,
  "total_score": 199,
  "created_at": "2026-09-01T10:30:00Z",
  "updated_at": "2026-09-26T15:45:00Z"
}
```
**Fields:**
- `user_id`: Unique integer identifier
- `user_name`: Unique username (3-50 chars)
- `prompt_score`: Net upvotes (up - down) on all prompts by this user
- `response_score`: Net upvotes (up - down) on all responses by this user
- `total_score`: Sum of `prompt_score` + `response_score`
- `created_at`: User creation timestamp
- `updated_at`: Last update timestamp

### Prompt
```json
{
  "post_id": 101,
  "user_id": 1,
  "user_name": "comedy_fan_42",
  "body": "Things you don't want to hear from your surgeon",
  "prompt_upvotes": 23,
  "response_count": 45,
  "created_at": "2026-09-20T12:00:00Z",
  "updated_at": "2026-09-20T12:00:00Z"
}
```
**Fields:**
- `post_id`: Unique integer identifier
- `user_id`: ID of the user who created the prompt
- `user_name`: Username of the creator
- `body`: The prompt text (1-280 characters)
- `prompt_upvotes`: Net upvotes (up - down) on this prompt
- `response_count`: Total number of responses to this prompt
- `created_at`: Creation timestamp
- `updated_at`: Last update timestamp

### Response
```json
{
  "response_id": 501,
  "post_id": 101,
  "user_id": 2,
  "user_name": "comedian_jane",
  "body": "Don't worry, I've done this a thousand times... on a simulator.",
  "response_upvotes": 67,
  "created_at": "2026-09-20T13:15:00Z",
  "updated_at": "2026-09-20T13:15:00Z"
}
```
**Fields:**
- `response_id`: Unique integer identifier
- `post_id`: ID of the prompt this response answers
- `user_id`: ID of the user who created the response
- `user_name`: Username of the creator
- `body`: The response/punchline text (1-280 characters)
- `response_upvotes`: Net upvotes (up - down) on this response
- `created_at`: Creation timestamp
- `updated_at`: Last update timestamp

### Vote (Internal, not returned directly)
```sql
-- Prompt votes
CREATE TABLE prompt_votes (
  user_name VARCHAR(50) NOT NULL,
  post_id BIGINT NOT NULL,
  vote_type VARCHAR(10) NOT NULL,  -- 'upvote' or 'downvote'
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (user_name, post_id)
);

-- Response votes
CREATE TABLE response_votes (
  user_name VARCHAR(50) NOT NULL,
  response_id BIGINT NOT NULL,
  vote_type VARCHAR(10) NOT NULL,  -- 'upvote' or 'downvote'
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (user_name, response_id)
);
```
*Note:* Vote entries are keyed by `user_name` (not `user_id`) in Phase 1 for simplicity. The `PRIMARY KEY` ensures atomic, duplicate-free voting.

---

## Endpoint Details

### `GET /prompts`
List all prompts with sorting and pagination.

*Example Request:*
```http
GET /prompts?sort=hot&limit=20&offset=0
```

*Example Response (200 OK):*
```json
{
  "prompts": [
    {
      "post_id": 101,
      "user_id": 1,
      "user_name": "comedy_fan_42",
      "body": "Things you don't want to hear from your surgeon",
      "prompt_upvotes": 23,
      "response_count": 45,
      "created_at": "2026-09-20T12:00:00Z",
      "updated_at": "2026-09-20T12:00:00Z"
    },
    {
      "post_id": 100,
      "user_id": 2,
      "user_name": "dark_humor_fan",
      "body": "Rejected names for a daycare center",
      "prompt_upvotes": 18,
      "response_count": 32,
      "created_at": "2026-09-19T15:30:00Z",
      "updated_at": "2026-09-19T15:30:00Z"
    }
  ],
  "total_count": 1250,
  "limit": 20,
  "offset": 0
}
```

---

### `POST /prompts`
Create a new prompt.

*Example Request:*
```json
{
  "user_name": "comedy_fan_42",
  "body": "Things you don't want to hear from your surgeon"
}
```

**Constraints:**
- `user_name`: Required, 3-50 characters
- `body`: Required, 1-280 characters

*Example Response (201 Created):*
```json
{
  "post_id": 102,
  "user_id": 1,
  "user_name": "comedy_fan_42",
  "body": "Things you don't want to hear from your surgeon",
  "prompt_upvotes": 0,
  "response_count": 0,
  "created_at": "2026-09-26T16:00:00Z",
  "updated_at": "2026-09-26T16:00:00Z"
}
```

*Error Response (400 Bad Request):*
```json
{
  "error": "Missing required field: body",
  "code": "MISSING_FIELD"
}
```

---

### `GET /prompts/{post_id}`
Get a specific prompt with all responses.

*Example Request:*
```http
GET /prompts/101?response_sort=hot
```

**Query Parameters:**
- `response_sort` (default: `hot`): `hot`, `top`, `newest`, `best`, `controversial`

*Example Response (200 OK):*
```json
{
  "post_id": 101,
  "user_id": 1,
  "user_name": "comedy_fan_42",
  "body": "Things you don't want to hear from your surgeon",
  "prompt_upvotes": 23,
  "response_count": 2,
  "created_at": "2026-09-20T12:00:00Z",
  "updated_at": "2026-09-20T12:00:00Z",
  "responses": [
    {
      "response_id": 501,
      "post_id": 101,
      "user_id": 2,
      "user_name": "comedian_jane",
      "body": "Don't worry, I've done this a thousand times... on a simulator.",
      "response_upvotes": 67,
      "created_at": "2026-09-20T13:15:00Z",
      "updated_at": "2026-09-20T13:15:00Z"
    },
    {
      "response_id": 500,
      "post_id": 101,
      "user_id": 3,
      "user_name": "funny_writer",
      "body": "Can you sign this waiver?",
      "response_upvotes": 52,
      "created_at": "2026-09-20T12:45:00Z",
      "updated_at": "2026-09-20T12:45:00Z"
    }
  ]
}
```

---

### `POST /responses`
Submit a response to a prompt.

*Example Request:*
```json
{
  "user_name": "comedian_jane",
  "post_id": 101,
  "body": "Don't worry, I've done this a thousand times... on a simulator."
}
```

**Constraints:**
- `user_name`: Required, 3-50 characters
- `post_id`: Required, must refer to an existing prompt
- `body`: Required, 1-280 characters

*Example Response (201 Created):*
```json
{
  "response_id": 502,
  "post_id": 101,
  "user_id": 2,
  "user_name": "comedian_jane",
  "body": "Don't worry, I've done this a thousand times... on a simulator.",
  "response_upvotes": 0,
  "created_at": "2026-09-26T16:05:00Z",
  "updated_at": "2026-09-26T16:05:00Z"
}
```

---

### `POST /prompts/{post_id}/upvote`
Upvote a prompt. Atomic operation (enforced by DB UNIQUE constraint).

*Example Request:*
```json
{
  "user_name": "another_user"
}
```

*Example Response (200 OK):*
```json
{
  "success": true,
  "message": "Upvote recorded",
  "new_upvote_count": 24
}
```

*Error Response (400 Bad Request) — user already voted:*
```json
{
  "error": "You have already voted on this prompt",
  "code": "ALREADY_VOTED"
}
```

---

### `POST /prompts/{post_id}/downvote`
Downvote a prompt. Atomic operation.

*Example Request:*
```json
{
  "user_name": "another_user"
}
```

*Example Response (200 OK):*
```json
{
  "success": true,
  "new_upvote_count": 22
}
```

---

### `POST /responses/{response_id}/upvote`
Upvote a response. Atomic operation.

*Example Request:*
```json
{
  "user_name": "another_user"
}
```

*Example Response (200 OK):*
```json
{
  "success": true,
  "message": "Upvote recorded",
  "new_upvote_count": 68
}
```

---

### `POST /responses/{response_id}/downvote`
Downvote a response. Atomic operation.

*Example Request:*
```json
{
  "user_name": "another_user"
}
```

*Example Response (200 OK):*
```json
{
  "success": true,
  "new_upvote_count": 66
}
```

---

### `GET /users/{user_name}`
Get a user's public profile.

*Example Request:*
```http
GET /users/comedy_fan_42
```

*Example Response (200 OK):*
```json
{
  "user_id": 1,
  "user_name": "comedy_fan_42",
  "prompt_score": 42,
  "response_score": 157,
  "total_score": 199,
  "created_at": "2026-09-01T10:30:00Z",
  "updated_at": "2026-09-26T15:45:00Z",
  "recent_prompts": [
    {
      "post_id": 101,
      "user_id": 1,
      "user_name": "comedy_fan_42",
      "body": "Things you don't want to hear from your surgeon",
      "prompt_upvotes": 23,
      "response_count": 45,
      "created_at": "2026-09-20T12:00:00Z"
    }
  ],
  "recent_responses": [
    {
      "response_id": 501,
      "post_id": 101,
      "user_id": 1,
      "user_name": "comedy_fan_42",
      "body": "Don't worry, I've done this a thousand times... on a simulator.",
      "response_upvotes": 67,
      "created_at": "2026-09-20T13:15:00Z"
    }
  ]
}
```

---

### `GET /leaderboard`
Get ranked users by total score.

*Example Request:*
```http
GET /leaderboard?sort_by=total_score&limit=10&offset=0
```

*Example Response (200 OK):*
```json
{
  "users": [
    {
      "user_id": 5,
      "user_name": "top_comedian",
      "prompt_score": 180,
      "response_score": 1240,
      "total_score": 1420,
      "created_at": "2026-06-15T08:00:00Z",
      "updated_at": "2026-09-26T10:00:00Z"
    },
    {
      "user_id": 3,
      "user_name": "funny_writer",
      "prompt_score": 95,
      "response_score": 1050,
      "total_score": 1145,
      "created_at": "2026-07-01T09:30:00Z",
      "updated_at": "2026-09-26T09:30:00Z"
    }
  ],
  "total_count": 500,
  "limit": 10,
  "offset": 0
}
```

---

## Error Codes

| Code | HTTP Status | Meaning |
| --- | --- | --- |
| `MISSING_FIELD` | 400 | Required field is missing |
| `INVALID_BODY` | 400 | Body exceeds 280 characters |
| `ALREADY_VOTED` | 400 | User has already voted on this item |
| `USER_NOT_FOUND` | 404 | User with given username does not exist |
| `PROMPT_NOT_FOUND` | 404 | Prompt with given ID does not exist |
| `RESPONSE_NOT_FOUND` | 404 | Response with given ID does not exist |

---

## Sorting Algorithms

- **`hot`**: Time-decay ranking. Fresh prompts/responses with strong upvote momentum rank high. Older items decay over time.
- **`top`**: Highest net upvotes (up - down) first, all-time.
- **`best`**: Confidence-weighted ranking using Wilson score interval. Accounts for the number of votes and the ratio of upvotes to downvotes.
- **`newest`**: Creation date descending (newest first).
- **`controversial`**: Net score close to zero. Highlights items with many upvotes AND many downvotes.

---

## Notes

- **Atomicity:** Vote endpoints enforce duplicate-prevention via `PRIMARY KEY (user_name, post_id)` or `PRIMARY KEY (user_name, response_id)` in the database. The `INSERT` is atomic — no app-layer checks needed.
- **User identification:** Phase 1 uses `user_name` as the primary identifier for voting/attribution (simpler than managing `user_id` in the session). In Phase 2, we'll add authentication and use `user_id` internally.
- **Pagination:** All list endpoints (`/prompts`, `/leaderboard`) return `total_count`, `limit`, and `offset` for client-side pagination.
- **Timestamps:** All timestamps are in ISO 8601 format with timezone (UTC).
- **Response count:** The `response_count` field in Prompt objects is denormalized (cached) for fast feed renders. It's updated each time a response is created.
- **Karma score:** Both `prompt_score` and `response_score` are denormalized counters. They're updated in the same transaction as a vote is recorded.