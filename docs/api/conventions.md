# API conventions (REST + WebSocket)

- **Roadmap task:** 0.4.1
- **Status:** Accepted (2026-10-15) — delegated decisions recorded as D-050
- **Applies to:** every endpoint in `rest.md` and every message in `websocket.md`
- **Related:** ADR-0002 (net/http, coder/websocket), ADR-0003 (dio, RealtimeClient), schema-identity §1 (session sync), errors.md, limits.md

---

## 1. Base
| Item | Rule |
|---|---|
| Base URL | `https://api.brainvolt.app/v1` (prod), `http://localhost:8080/v1` (local). Host names finalised in Phase 9 deployment docs. |
| Transport | HTTPS only in deployed environments (TLS 1.2+). HTTP only on localhost. |
| WebSocket URL | `wss://api.brainvolt.app/v1/ws` (same host, same version prefix) |
| Format | JSON (`application/json; charset=utf-8`), UTF-8. No XML, no form posts. |
| Field names | `snake_case` in JSON (matches DB and glossary code names). Dart DTOs map to camelCase via `json_serializable`. |
| Enum values | lowercase `snake_case` strings exactly as in the glossary (`quick_duel`, `shared_answer`, …). Unknown enum values from the server must be tolerated by clients (forward compatibility). |
| IDs | UUIDv7 strings (`"0192…"`); never integers. |
| Times | RFC 3339 UTC with milliseconds, e.g. `"2026-10-15T15:42:07.123Z"`. Exception: **WebSocket timing fields** (`show_at`, `deadline_at`, `ts`) are **Unix milliseconds** (integers) for cheap math on the client. |
| Durations | integer milliseconds, field suffix `_ms`. |
| Money | never sent as floats; purchases are store-side (commerce). |
| Nulls | omitted fields and `null` are equivalent on input; output includes keys with `null` only where the field exists but has no value. |

## 2. Versioning
- Major version in the path (`/v1`). Breaking changes → `/v2`, with `/v1` kept until all supported app versions migrate.
- Non-breaking changes (new fields, new endpoints, new enum values) ship inside `v1`.
- The app sends `X-App-Version: 1.4.2+87` and `X-Platform: android|ios`. The server can answer **`426 UPGRADE_REQUIRED`** (error code `APP_VERSION_UNSUPPORTED`) with the minimum version, so old clients are forced to update when rules change (e.g. a new ruleset).

## 3. Authentication
| Item | Rule |
|---|---|
| REST | `Authorization: Bearer <Firebase ID token>` on every endpoint except `/healthz`, `/readyz`, `/v1/meta/*` (public config) and store/ad webhooks (which use their own signature verification). |
| Verification | Firebase Admin SDK: signature, `aud` = Firebase project `brainvolt`, `iss`, expiry. Token revocation checked (`checkRevoked`) only on sensitive endpoints (session, account deletion, purchases). |
| Mapping | `firebase_uid` → `user_id` via cache `id:fuid:*` (redis-keys §7); first call must be `POST /v1/session` (schema-identity §5). |
| Status checks | `suspended` users: only `GET /v1/me`, sanction notice, appeals, account deletion allowed → others `403 ACCOUNT_SUSPENDED`. Deleted users: `401`. |
| Tier checks | restricted-only actions → `403 ACCOUNT_UPGRADE_REQUIRED` (schema-identity §2.2). |
| Roles | staff endpoints under `/v1/admin/*` require `moderator` or `admin`; `403 FORBIDDEN` otherwise. |
| WebSocket | token sent in the **first message** (`sys.auth`), not in the URL (URLs end up in logs). See websocket.md §2. |
| Emulator | local dev: backend reads `FIREBASE_AUTH_EMULATOR_HOST` (ADR-0003). |

## 4. Localization
- `Accept-Language: tr` or `en` on every request. Unknown or missing → profile locale → `tr` (D-001).
- Server-generated user-facing text (error `message`, notification texts, question text) is returned in that locale. Clients still show their own translated text for known error codes; `message` is a fallback.

## 5. Request IDs & tracing
- Client may send `X-Request-ID` (UUID); otherwise the server generates one. Echoed in every response header and inside every error body; logged with every log line (`slog`).
- WebSocket messages carry their own `id` (client) / `seq` (server) — websocket.md §3.

## 6. Errors
Every non-2xx response has this body:
```json
{
  "error": {
    "code": "ACCOUNT_UPGRADE_REQUIRED",
    "message": "Bu işlem için hesabını yükseltmelisin.",
    "details": { "action": "friend_request.create" },
    "request_id": "0192c7e0-…"
  }
}
```
- `code`: stable, UPPER_SNAKE_CASE, from the registry in `errors.md`. Clients switch on `code`, never on `message`.
- `details`: optional structured data (field errors, retry hints).
- Validation errors: `400 VALIDATION_FAILED` with `details.fields: [{ "field": "display_name", "rule": "length", "min": 2, "max": 24 }]`.
- Rate limits: `429 RATE_LIMITED` + `Retry-After` header (seconds).

## 7. HTTP status usage
| Status | When |
|---|---|
| 200 | success with body · 201 created (with `Location`) · 202 accepted for async work · 204 success, no body |
| 400 | malformed JSON / validation |
| 401 | missing/invalid/expired token |
| 403 | authenticated but not allowed (tier, role, sanction, block) |
| 404 | resource not found **or not visible to this user** (never leak existence) |
| 409 | state conflict (already in a live match, attempt already used, duplicate) |
| 410 | expired (invite code, challenge) |
| 422 | semantically invalid config (e.g. mode-matrix violation) |
| 426 | app version too old |
| 429 | rate limited |
| 500 / 503 | server error / temporarily unavailable (with `Retry-After`) |

## 8. Pagination
- Cursor-based everywhere: `?limit=20&cursor=<opaque>`; `limit` 1–100, default 20.
- Response: `{ "items": [...], "next_cursor": "…" | null }`. Cursors are opaque (base64 of the last sort key); clients never build them.
- Leaderboards additionally accept `?around=me` (window of ±10 around the caller).

## 9. Idempotency
- Every `POST` that **creates** something or **spends** a limited resource accepts `Idempotency-Key: <UUID>` (client-generated, required for: friend requests, challenges, room creation, report, purchase verification, daily run start, quest claim, account deletion).
- The server stores key → response for **24 h** (Redis `rl:`-style key `idem:<user_id>:<key>`); a repeated request returns the stored response with header `Idempotent-Replayed: true`.
- Same key with a different body → `409 IDEMPOTENCY_KEY_REUSED`.

## 10. Concurrency control
Mutable resources edited by several people (rooms, team settings, creator quizzes) return `version` (int). Updates send `If-Match: <version>`; mismatch → `409 VERSION_CONFLICT` with the current resource.

## 11. Caching
- Static-ish data (`/v1/meta/*`, categories, leagues, badges, cosmetics catalogue) sends `ETag` + `Cache-Control: max-age=300`; clients use `If-None-Match`.
- User-specific data: `Cache-Control: no-store`.

## 12. Health & meta
| Endpoint | Purpose |
|---|---|
| `GET /healthz` | liveness (process up) |
| `GET /readyz` | readiness (Postgres + Redis reachable, migrations applied) |
| `GET /v1/meta/config` | public config: min app version, ruleset version, feature flags, season info, legal URLs |

## 13. Security headers & CORS
- `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`, no `Server` version header.
- CORS: none (mobile clients only); admin web console in a later phase gets its own allow-list.
- Request bodies over the limit → `413` (limits.md).

---

## Delegated decisions (D-050)
JSON snake_case; UUIDv7 ids; RFC 3339 times on REST, Unix ms on WebSocket timing fields; `/v1` path versioning with forced-update via 426; Firebase Bearer tokens, WS token in first message; `Accept-Language`; error envelope with stable codes; cursor pagination; `Idempotency-Key` 24 h on creating/spending POSTs; `If-Match` versions for shared resources.
