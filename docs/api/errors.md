# Error code registry (REST + WebSocket)

- **Roadmap task:** 0.4.7
- **Status:** Accepted (2026-10-15)
- **Format:** conventions.md §6 (REST envelope) and websocket.md §3 (`sys.error {re, code, message, details}`).
- **Rules:** codes are UPPER_SNAKE_CASE, never renamed or reused; new codes may be added any time; clients must handle unknown codes generically (show `message`). Each code has TR and EN messages in `backend/internal/platform/apierror/messages_{tr,en}.go` (shared by REST and WebSocket; a test keeps `codes.go` identical to this table) and mobile ARB keys `error_<code>`.

| Code | HTTP | WS | Meaning / when |
|---|---|---|---|
| **General** | | | |
| `VALIDATION_FAILED` | 400 | ✓ | field errors in `details.fields` |
| `MALFORMED_REQUEST` | 400 | ✓ (close 4000 if repeated) | invalid JSON / unknown required shape |
| `PAYLOAD_TOO_LARGE` | 413 | close 4000 | over limits.md |
| `NOT_FOUND` | 404 | ✓ | missing or not visible |
| `METHOD_NOT_ALLOWED` | 405 | — | route exists but not for this HTTP method; `Allow` header lists the methods (added in Phase 1, D-055) |
| `FORBIDDEN` | 403 | ✓ | role/permission |
| `RATE_LIMITED` | 429 | ✓ / close 4008 | `details.retry_after_s` |
| `IDEMPOTENCY_KEY_REUSED` | 409 | — | same key, different body |
| `IDEMPOTENT_REQUEST_IN_PROGRESS` | 409 | — | same key still being processed by an earlier request; retry after `Retry-After` (added in Phase 1, D-056) |
| `VERSION_CONFLICT` | 409 | ✓ | `If-Match` mismatch; `details.current` |
| `APP_VERSION_UNSUPPORTED` | 426 | close 4026 | `details.min_version` |
| `INTERNAL` | 500 | ✓ | unexpected; includes `request_id` |
| `UNAVAILABLE` | 503 | ✓ | maintenance / dependency down; `Retry-After` |
| **Auth & account** | | | |
| `UNAUTHENTICATED` | 401 | close 4002 | missing/invalid token |
| `TOKEN_EXPIRED` | 401 | close 4003 | refresh and retry |
| `AGE_CONFIRMATION_REQUIRED` | 400 | — | first session without age gate (D-020) |
| `ACCOUNT_UPGRADE_REQUIRED` | 403 | ✓ | restricted account action (D-016/D-017); `details.action` |
| `ACCOUNT_SUSPENDED` | 403 | close 4004 | suspension (D-030) |
| `ACCOUNT_DELETION_PENDING` | 403 | ✓ | action blocked during deletion cooling-off (only cancel allowed) |
| `SANCTIONED` | 403 | ✓ | `details.kind` (`ranked_ban`, `matchmaking_ban`, `social_ban`, `chat_mute`) and `ends_at` |
| `USERNAME_TAKEN` | 409 | — | |
| `CHANGE_TOO_SOON` | 409 | — | username/country 30-day rule; `details.available_at` |
| `TEXT_REJECTED` | 422 | ✓ | blocked by word filter (D-030) |
| **Social & lobby** | | | |
| `BLOCKED` | 403 | ✓ | a block exists between the users (no detail on direction) |
| `FRIEND_LIMIT_REACHED` | 409 | — | 300 (D-027) |
| `ALREADY_FRIENDS` / `REQUEST_PENDING` | 409 | — | |
| `CHALLENGES_DISABLED` | 403 | ✓ | target's `challenges_from` setting |
| `INVITE_CODE_INVALID` / `INVITE_CODE_EXPIRED` | 404 / 410 | — | |
| `ROOM_FULL` / `ROOM_CLOSED` | 409 / 410 | ✓ | |
| `ROOM_FRIENDS_ONLY` | 403 | ✓ | unverified creator quiz room (D-023) |
| `CHALLENGE_EXPIRED` | 410 | ✓ | |
| `REVENGE_NOT_ALLOWED` | 403 | — | retired with revenge (D-091); kept so old clients still get a text |
| `TEAM_LIMIT_REACHED` / `TEAM_FULL` | 409 | — | 3 teams per user, 4 members (D-027) |
| `NOT_CAPTAIN` | 403 | — | |
| `COUPLE_EXISTS` / `COUPLE_COOLDOWN` | 409 | — | one live couple; 7-day cooldown (D-047), `details.available_at` |
| **Matchmaking & match** | | | |
| `ALREADY_IN_MATCH` | 409 | ✓ | one live match per user; `details.match_id` |
| `ALREADY_QUEUED` | 409 | — | |
| `QUEUE_COOLDOWN` | 409 | ✓ | no-show / leaver cooldown (D-034, D-048); `details.until` |
| `MODE_NOT_ALLOWED` | 422 | ✓ | combination rejected by the mode matrix (0.5); `details.rule_id` |
| `NOT_ENOUGH_QUESTIONS` | 422 | ✓ | pool too small for config/locales |
| `NO_ACTIVE_SEASON` | 503 | ✓ | ranked unavailable between seasons |
| `MATCH_NOT_FOUND` / `NOT_A_PARTICIPANT` | 404 / 403 | ✓ | |
| `MATCH_STATE_INVALID` | 409 | ✓ | intent not valid in current phase (e.g. `ready` after start) |
| `ANSWER_LATE` | — | ✓ | after `deadline_at + grace` |
| `ANSWER_DUPLICATE` | — | ✓ | already answered this round |
| `ANSWER_INVALID_OPTION` | — | ✓ | option not part of this question |
| `NOT_YOUR_TURN` | — | ✓ | Individual Turns / picker rules |
| `PICK_NOT_ALLOWED` | — | ✓ | Mixed Battle: category not offered or 4th consecutive |
| `SEAT_TAKEN_ELSEWHERE` | — | ✓ | another device holds the seat |
| **Daily Challenge** | | | |
| `DAILY_ATTEMPT_USED` | 409 | — | official attempt already completed today (D-040) |
| `DAILY_ATTEMPT_IN_PROGRESS` | 409 | — | official run running; `details.run_id` |
| `DAILY_NOT_AVAILABLE` | 503 | — | today's set not published |
| `DAILY_ANSWERS_NOT_OPEN` | 409 | — | `GET /v1/daily/answers` for today or more than 7 days back: answers open after the day (D-100) |
| `AD_REWARD_UNAVAILABLE` | 409 | `placement`, `reason` | `POST /v1/ads/rewards`: the reward cannot be earned now — `no_match_xp`, `too_late`, `already_given`, `free_reroll_left`, `no_open_quest`, `not_repairable`, `ads_off` (D-103) |
| `AD_LIMIT_REACHED` | 409 | `placement`, `limit` | `POST /v1/ads/rewards`: the daily (or, for `streak_repair`, weekly) limit of rewarded ads is used up (D-103) |
| `VOLT_INSUFFICIENT` | 409 | `price`, `balance` | `POST /v1/store/items/{code}/buy`: the Volt balance is below the item's price (D-105) |
| `ITEM_UNAVAILABLE` | 409 | `item`, `reason` | `POST /v1/store/items/{code}/buy`: the item cannot be used now (`shields_full`, `free_reroll_left`, `no_open_quest`; D-105) |
| **Content / creator / moderation** | | | |
| `QUESTION_SOURCE_REQUIRED` | 422 | — | D-006 |
| `QUESTION_IN_USE` | 409 | — | structural edit of a used question (schema-content §4.3) |
| `QUESTION_STATE_INVALID` | 409 | — | action not allowed in the question's (or translation's) current status, e.g. review of a draft, edit of a retired question; `details.status` (added in Phase 2, D-063) |
| `QUESTION_DUPLICATE` | 409 | — | same normalised prompt + options already exist; `details.question_id` (added in Phase 2, D-063) |
| `QUIZ_LIMIT_REACHED` / `QUIZ_SIZE_INVALID` | 409 / 422 | — | 20 quizzes, 1–30 questions (D-031) |
| `REPORT_LIMIT_REACHED` | 429 | — | 20/day (D-030) |
| `ALREADY_REPORTED` | 409 | — | one report per target |
| `APPEAL_EXISTS` | 409 | — | one appeal per sanction |
| `SELF_REVIEW_FORBIDDEN` | 403 | — | moderator acting on own case/appeal |
| **Commerce** | | | |
| `PURCHASE_INVALID` | 422 | — | store verification failed |
| `PURCHASE_ALREADY_PROCESSED` | 409 | — | idempotent replay of a known transaction |
| `AD_REWARD_LIMIT` | 429 | — | 5 rewarded ads/day (D-031) |
| **Progression** | | | |
| `QUEST_NOT_COMPLETED` / `QUEST_ALREADY_CLAIMED` | 409 / 409 | — | claim before completion / twice (added in Phase 7, D-077) |
| `QUEST_REROLL_LIMIT` | 429 | — | the day's free reroll is used (progression-formulas §8) |
| `QUEST_NOT_REROLLABLE` | 409 | — | completed or claimed quest, or no other quest to draw |
| **Tournament** | | | |
| `TOURNAMENT_REGISTRATION_CLOSED` / `TOURNAMENT_FULL` / `TOURNAMENT_REQUIREMENTS_NOT_MET` | 409 / 409 / 403 | — | |
