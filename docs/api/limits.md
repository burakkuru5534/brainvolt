# Rate limits & payload limits

- **Roadmap task:** 0.4.8
- **Status:** Accepted (2026-10-15) — consolidates D-027, D-030, D-031, D-032; new values delegated (D-052)
- **Implementation:** `platform` rate limiter on Redis counters `rl:<name>:<subject>:<window>` (redis-keys §7.1); money-related limits checked in Postgres. Exceeding a limit → `429 RATE_LIMITED` / `sys.error RATE_LIMITED` with retry hint; WS flood → close 4008.

## 1. Request rate limits
| Name | Limit | Subject | Source |
|---|---|---|---|
| API general | 60 requests / 10 s | user | D-032 |
| API per IP | 300 requests / min | IP | D-032 |
| WebSocket messages | 20 / s (burst), 600 / min | connection | D-032 + delegated |
| `POST /v1/session` | 20 / h | installation | D-032 |
| Unauthenticated endpoints (`/v1/meta`, `/v1/credits`) | 60 / min | IP | delegated |
| Friend requests | 50 / day | user | D-027 |
| Direct challenges (incl. "beat my score") | 20 / h | user | D-027, D-040 |
| Room creation | 10 / h | user | delegated |
| Room joins by code (wrong codes) | 10 failed / 10 min | user + IP | delegated (brute-force protection) |
| Invite code creation | 30 / day | user | delegated |
| Matchmaking ticket creation | 30 / 10 min | user | delegated |
| Chat messages | 5 / 10 s | user | D-031 |
| Quick-chat phrases in a match | 3 / 5 s | participant | delegated |
| Reports | 20 / day | user | D-030 |
| Push notifications received | 10 / h | user | D-031 |
| Rewarded ads | 5 / day | user (Postgres) | D-031 |
| Interstitial ads | ≤ 1 per 3 matches | user (Postgres) | D-031 |
| Profile edits (`PATCH /v1/me/profile`) | 10 / h | user | delegated |
| Creator quiz writes | 60 / h | user | delegated |
| Admin endpoints | 120 / min | staff user | delegated |

## 2. Payload limits
| Item | Limit |
|---|---|
| REST request body | 64 KB (creator quiz save: 256 KB) |
| WebSocket client message | 4 KB |
| WebSocket server message | 256 KB (snapshots) |
| Chat message body | 300 chars (D-031) |
| Report comment / appeal text | 500 / 1 000 chars (D-030) |
| Display name / username | 2–24 / 3–20 chars (schema-identity) |
| Team name / tag | 3–24 / 2–4 chars (schema-social) |
| Question prompt / option / explanation | 300 / 120 / 500 chars (schema-content) |
| Pagination `limit` | 1–100 (default 20) |
| Idempotency key retention | 24 h |

## 3. Timeouts
| Item | Value |
|---|---|
| HTTP server read / write / idle | 10 s / 15 s / 60 s |
| WS `sys.auth` deadline | 5 s |
| WS dead-connection detection | 3 missed pongs (6 s in play, 60 s idle) |
| Outbound calls (Firebase, stores, FCM) | 5 s, 2 retries with backoff (in River jobs, not in request path where possible) |
| DB statement timeout (API requests) | 3 s (`statement_timeout`); worker jobs 60 s |
