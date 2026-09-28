# MicDrop — Product Requirements Document

## 1. Overview

MicDrop is a crowdsourced comedy platform inspired by the improv game “Scenes From a Hat” and Reddit-style community voting. Users post comedic prompts/setups, others submit one-liner responses/punchlines, and the community upvotes or downvotes submissions to surface the funniest content.

The MVP is an API-first backend. It supports users, prompts, responses, voting, denormalized karma scores, and basic sorting/pagination. Authentication is mocked in Phase 1.

## 2. Goals

- Let users create/upsert a lightweight profile.
- Let users create comedic prompts.
- Let users submit one-liner responses to prompts.
- Let users upvote/downvote prompts and responses.
- Enforce one vote per user per prompt/response.
- Track user karma via `prompt_score`, `response_score`, and `total_score`.
- Provide list endpoints with basic sorting and pagination.
- Keep vote and response writes atomic.

## 3. Non-Goals for MVP (Phase 1)

- Real authentication/authorization (Google OAuth, JWTs).
- User profile pages listing a user's prompts/responses.
- Comment threads.
- Editing or deleting prompts/responses.
- Changing or retracting votes.
- Full hot/best/controversial ranking algorithms.
- Leaderboard API (repo method exists, but route is not wired).
- Moderation/reporting.
- Notifications.
- Media uploads.

## 4. Target Users

- Comedy writers and comedians who want quick joke feedback.
- Comedy fans who want to riff and vote.
- Communities running asynchronous improv/comedy games.

## 5. Core User Flows

1. User creates or upserts a profile with `user_id` and `user_name`.
2. User creates a prompt.
3. Other users browse prompts and submit responses.
4. Users upvote/downvote prompts and responses.
5. Scores update atomically.
6. Users browse prompt responses sorted by top/newest behavior.

## 6. Functional Requirements

### 6.1 User Management

- Users are identified by `user_id` (string).
- `user_name` is a display name.
- `POST /api/v1/users` creates or returns a user.
- `GET /api/v1/users/{userID}` returns a user profile and scores.
- Scores:
  - `prompt_score`: net votes on the user’s prompts.
  - `response_score`: net votes on the user’s responses.
  - `total_score`: `prompt_score + response_score`.

### 6.2 Prompts

- A prompt belongs to exactly one user.
- A prompt has `post_id`, `user_id`, `body`, `prompt_upvotes`, `response_count`, timestamps.
- `POST /api/v1/prompts` creates a prompt.
- `GET /api/v1/prompts` lists prompts.
  - Supported sort in MVP: `newest` (default) and `top`.
- `GET /api/v1/prompts/{postID}` fetches one prompt.
- Prompts can be upvoted/downvoted.

### 6.3 Responses

- A response belongs to exactly one prompt and one user.
- A response has `response_id`, `post_id`, `user_id`, `body`, `response_upvotes`, timestamps.
- `POST /api/v1/prompts/{postID}/responses` creates a response.
- `GET /api/v1/prompts/{postID}/responses` lists responses for a prompt.
- Responses can be upvoted/downvoted.

### 6.4 Voting

- A user may cast at most one vote per prompt.
- A user may cast at most one vote per response.
- Duplicate vote returns `409 Conflict`.
- Vote writes are transactional:
  - Insert vote row.
  - Update denormalized counter.
  - Update author score.
- Vote identity is passed through the `X-User-ID` header for MVP.

### 6.5 Sorting and Pagination

- Prompts:
  - `sort=newest` — creation date descending.
  - `sort=top` — `prompt_upvotes DESC, created_at DESC`.
  - Other values fall back to newest.
- Responses:
  - Ordered by `response_upvotes DESC, created_at ASC`.
- Pagination is supported through `limit` and `offset`.
- MVP list responses return arrays only, not `total_count`.

## 7. API Summary

Base URL: `http://localhost:3000/api/v1`

| Endpoint | Method | Purpose |
|---|---|---|
| `/users` | POST | Create/upsert user |
| `/users/{userID}` | GET | Get user profile |
| `/prompts` | POST | Create prompt |
| `/prompts` | GET | List prompts |
| `/prompts/{postID}` | GET | Get prompt |
| `/prompts/{postID}/upvote` | POST | Upvote prompt |
| `/prompts/{postID}/downvote` | POST | Downvote prompt |
| `/prompts/{postID}/responses` | POST | Create response |
| `/prompts/{postID}/responses` | GET | List responses |
| `/responses/{responseID}/upvote` | POST | Upvote response |
| `/responses/{responseID}/downvote` | POST | Downvote response |
| `/healthz` | GET | Health check |

## 8. Acceptance Criteria

- Creating a user returns a user object.
- Creating a prompt with an existing `user_id` returns `201 Created`.
- Creating a response with an existing `user_id` and `post_id` returns `201 Created`.
- Duplicate vote returns `409 Conflict`.
- Missing `X-User-ID` on vote endpoints returns `401 Unauthorized`.
- Invalid `postID`/`responseID` returns `400 Bad Request`.
- Missing prompt/response/user returns `404 Not Found`.
- Prompt response creation increments `prompts.response_count`.
- Vote creation updates both the target counter and author score.
- Duplicate votes are prevented by database primary keys.

## 9. Future Requirements (Phase 2+)

- Real authentication (Google OAuth) and JWT/session identity.
- User profile pages (`GET /users/{userID}/prompts`, `GET /users/{userID}/responses`).
- `GET /responses/{responseID}`.
- Leaderboard endpoint (`GET /leaderboard`).
- Hot/Best/Controversial ranking.
- Total counts in paginated responses.
- Vote change/retraction.
- Self-vote prevention.
- Rate limiting.
- Moderation and reporting.