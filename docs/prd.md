# MicDrop — Product Requirements Document

## 1. Overview

MicDrop is a crowdsourced comedy platform inspired by the improv game “Scenes From a Hat” and Reddit-style community voting. Users post comedic prompts/setups, others submit one-liner responses/punchlines, and the community upvotes or downvotes submissions to surface the funniest content.

The MVP is an API-first backend. It supports users, prompts, responses, voting, denormalized karma scores, and basic sorting/pagination. Authentication is Google OAuth with JWT sessions.

## 2. Goals

- Let users sign in with Google and pick a username.
- Let users create comedic prompts.
- Let users submit one-liner responses to prompts.
- Let users upvote/downvote prompts and responses, change, or retract votes.
- Track user karma via `prompt_score`, `response_score`, and `total_score` (self-votes excluded).
- Show each viewer their own vote state (`viewer_vote`) on every item.
- Provide list endpoints with basic sorting and pagination.
- Keep vote and response writes atomic.

## 3. Non-Goals for MVP (Phase 1)

- Comment threads.
- Editing or deleting prompts/responses.
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

1. User signs in with Google and picks a username (new accounts) or lands in the app (returning accounts).
2. User creates a prompt (starts at 1 with the author's auto-upvote).
3. Other users browse prompts and submit responses.
4. Users upvote/downvote prompts and responses; re-clicking toggles off, opposite-click flips.
5. Scores and karma update atomically.
6. Users browse prompt responses sorted by top/newest behavior.

## 6. Functional Requirements

### 6.1 User Management

- Users are identified by `user_id` (string, chosen at signup, immutable).
- `user_name` is a display name (defaults to `user_id`).
- Signup is Google OAuth: `GET /auth/google/login` → `GET /auth/google/callback` → `POST /auth/complete-signup` with the onboarding token.
- Session JWTs (`purpose: "session"`, 72h) authenticate writes via `Authorization: Bearer`.
- `GET /api/v1/users/{userID}` returns a user profile with stats.
- `GET /api/v1/users/{userID}/prompts` and `/responses` list a user's items.
- `PATCH /api/v1/users/me` edits display name, bio, and links.
- Scores:
  - `prompt_score`: net votes on the user's prompts, excluding self-votes.
  - `response_score`: net votes on the user's responses, excluding self-votes.
  - `total_score`: `prompt_score + response_score`.

### 6.2 Prompts

- A prompt belongs to exactly one user.
- A prompt has `post_id`, `user_id`, `body`, `prompt_upvotes`, `response_count`, `viewer_vote`, timestamps.
- `POST /api/v1/prompts` creates a prompt (author auto-upvoted, starts at 1).
- `GET /api/v1/prompts` lists prompts.
  - Supported sort in MVP: `newest` (default) and `top`.
- `GET /api/v1/prompts/{postID}` fetches one prompt.
- Prompts can be upvoted/downvoted.

### 6.3 Responses

- A response belongs to exactly one prompt and one user.
- A response has `response_id`, `post_id`, `user_id`, `body`, `response_upvotes`, `viewer_vote`, timestamps.
- `POST /api/v1/prompts/{postID}/responses` creates a response (author auto-upvoted, starts at 1).
- `GET /api/v1/prompts/{postID}/responses` lists responses for a prompt.
- Responses can be upvoted/downvoted.

### 6.4 Voting

- Voting is desired-state: `PUT /prompts/{postID}/vote` and
  `PUT /responses/{responseID}/vote` take `{"vote": "upvote"|"downvote"|"none"}`.
- Clicking the current direction clears the vote; clicking the opposite flips it.
- Authors may vote on their own items; creating an item auto-records the
  author's upvote (items start at 1).
- Self-votes move the displayed counter but never author karma.
- Vote writes are transactional (lock parent row, diff old vs new, apply delta).
- Every read carries the viewer's own state as `viewer_vote` (`null` when anonymous).

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
| `/auth/google/login` | GET | Start Google OAuth |
| `/auth/google/callback` | GET | OAuth callback (issues session or onboarding token) |
| `/auth/complete-signup` | POST | Pick username, receive session token |
| `/auth/me` | GET | Current user |
| `/users/me` | PATCH | Edit own profile |
| `/users/{userID}` | GET | Get user profile + stats |
| `/users/{userID}/prompts` | GET | List user's prompts |
| `/users/{userID}/responses` | GET | List user's responses |
| `/prompts` | POST | Create prompt |
| `/prompts` | GET | List prompts |
| `/prompts/{postID}` | GET | Get prompt |
| `/prompts/{postID}/vote` | PUT | Set vote (`upvote`/`downvote`/`none`) |
| `/prompts/{postID}/responses` | POST | Create response |
| `/prompts/{postID}/responses` | GET | List responses |
| `/responses/{responseID}/vote` | PUT | Set vote (`upvote`/`downvote`/`none`) |
| `/healthz` | GET | Health check |

## 8. Acceptance Criteria

- Completing signup with a valid onboarding token returns a session token and user.
- Creating a prompt with a valid session returns `201 Created` with count 1 and author `viewer_vote`.
- Creating a response with an existing `post_id` returns `201 Created`.
- Setting the same vote twice is a stable no-op (no error).
- Missing/invalid token on protected endpoints returns `401 Unauthorized`.
- Invalid `postID`/`responseID` or `vote` value returns `400 Bad Request`.
- Missing prompt/response/user returns `404 Not Found`.
- Prompt response creation increments `prompts.response_count`.
- Votes update both the target counter and (non-self) author karma.
- Reads include the caller's `viewer_vote`, `null` when anonymous.

## 9. Future Requirements (Phase 2+)

- `GET /responses/{responseID}`.
- Leaderboard endpoint (`GET /leaderboard`).
- Hot/Best/Controversial ranking.
- Total counts in paginated responses.
- Rate limiting.
- Moderation and reporting.