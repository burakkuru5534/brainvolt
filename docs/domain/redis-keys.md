# Redis key catalogue

- **Roadmap task:** 0.2.9
- **Status:** Accepted (2026-10-15)
- **Redis version:** 8.10 (ADR-0001). Client: go-redis v9 (ADR-0002).
- **Related:** `backend-modules.md` §3–§4 (live match authority, module-prefixed keys), all `schema-*.md` documents.
- Values marked **(P)** were proposed and accepted (D-032, §9).

---

## 1. Ground rules

1. **Postgres is the system of record; Redis holds live, derived or cached data.** Every Redis key is either (a) live state that only exists while something is happening (a match, a queue ticket, a connection), or (b) rebuildable from Postgres (leaderboards, caches). §8 lists what happens if Redis is wiped.
2. **Every key starts with its owning module's prefix** and only that module reads/writes it (`backend-modules.md` §4.4):
   `id:` identity · `content:` content · `game:` game · `rt:` realtime · `lobby:` lobby · `mm:` matchmaking · `rank:` ranking · `mod:` moderation · `rl:` shared rate-limiter in `platform`.
3. **Hash tags `{…}`** group keys that are touched together in one Lua script / transaction (e.g. all keys of one match share `{m:<id>}`), so the layout already works if we move to Redis Cluster later.
4. **Every key has a TTL** unless the table says *none* and explains why. No unbounded growth.
5. IDs are UUIDv7 strings; timestamps are Unix **milliseconds** of server time.
6. Server config: AOF `appendfsync everysec` + RDB snapshots; **`maxmemory-policy noeviction`** — live match state must never be silently evicted; memory alerts instead (Phase 9).

---

## 2. `game:` — live matches  (writer: the match owner only, `backend-modules.md` §3.4)

| Key | Type | Content | TTL |
|---|---|---|---|
| `game:{m:<match_id>}:owner` | STRING | owning instance id; acquired with `SET NX PX`, renewed every 3 s | **10 s** (lease) |
| `game:{m:<match_id>}:state` | HASH | `status`, `phase` (lobby / question / reveal / between_rounds), `round_no`, `seq` (last event number), `config` (JSON snapshot), `started_at`, `ruleset_version` | none while live; **1 h** after completed/cancelled |
| `game:{m:<match_id>}:participants` | HASH | `user_id` → JSON {side, seat, locale, status, connected, last_seen_ms, disconnects} | same as `:state` |
| `game:{m:<match_id>}:scores` | HASH | `side:1`, `side:2`, `user:<id>` → running score | same |
| `game:{m:<match_id>}:round` | HASH | current round: `question_id`, `revision`, `option_order`, `correct_option_id` (**server only, never published**), `dispatched_at`, `deadline_at`, `answering_user_id` | same |
| `game:{m:<match_id>}:answers:<round_no>` | HASH | `user_id` → JSON {option_id, received_ms, response_ms}; written with `HSETNX` so the **first** accepted answer is final | same |
| `game:{m:<match_id>}:events` | STREAM | ordered client-visible events (`seq` = stream entry) for reconnect/resume (0.4.6) | `MAXLEN ~ 1000`; same TTL |
| `game:{m:<match_id>}:rejects` | HASH | `user_id` → count of rejected submissions (cheat-signal threshold, D-030) | same |
| `game:user:<user_id>:live` | STRING | the user's current live match id; `SET NX` enforces **one live match per user** | **3 h** safety TTL; deleted at match end |
| `game:live` | ZSET | match_id → last owner heartbeat ms; sweeper finds orphaned matches whose owner lease expired | none (members removed at end) |
| pub/sub `game:ch:inst:<instance_id>` | channel | match intents forwarded to the instance owning the match (D-066) | — |

**As built (D-066):** the owner keeps the complete engine state as one JSON value in `:state` (fields `engine`, `meta` — question texts and player cards —, `seats` — user → session holding the seat — and `status`), written after every step; `:participants`, `:scores`, `:round`, `:answers:<n>` and `:rejects` are covered by it and not written separately. The correct option therefore stays in Redis on the server side only, as before.

---

## 2a. `game:` — Daily Challenge solo runs (D-040; Phase 8)

| Key | Type | Content | TTL |
|---|---|---|---|
| `game:{d:<run_id>}:state` | HASH | user, day, kind, position, show_at / deadline of current question, option order, `correct_option_id` (server only), running correct count & total time, owner instance | none while running; **1 h** after end |
| `game:{d:<run_id>}:owner` | STRING | owner lease (same rules as matches) | **10 s** |
| `game:daily:<yyyy-mm-dd>:<user_id>:official` | STRING | run id of the running/completed official run (fast check; Postgres unique index is authoritative) | **48 h** |

## 3. `mm:` — matchmaking (Phase 5)

| Key | Type | Content | TTL |
|---|---|---|---|
| `mm:queues` | SET | every queue name used so far (the matcher walks them) | none |
| `mm:q:<queue>` | ZSET | ticket_id → rating (for range search) | none (members removed when matched/expired) |
| `mm:t:<ticket_id>` | STRING | ticket JSON: user, locale, request, rating, rd, enqueued_ms, block list snapshot (the search range is computed from the wait, D-068) | **10 min** (P) |
| `mm:user:<user_id>` | STRING | ticket id — one ticket per user | **10 min** |
| `mm:lock:<queue>` | STRING | lease: only one matcher loop per queue across instances | **5 s**, renewed |
| `mm:m:<match_id>` | HASH | user_id → ticket JSON of a new matchmade match, to re-queue its ready players if the ready check fails (D-034) | **2 min** |
| `mm:missed:<user_id>` / `mm:leaves:<user_id>` | ZSET | missed ready checks / matchmade matches left (timestamps) | 30 min / 24 h |
| `mm:cooldown:<user_id>` | STRING | queue cooldown (3 missed ready checks → 5 min; 2nd leave in 24 h → 5 min, 3rd+ → 15 min) | the cooldown |

`<queue>` = mode-matrix §3.4: `<match_type>:<format>:<rating_bucket or casual>:<category_mode or default>:<category_id or ->:<locale_group>`, e.g. `ranked:1v1:solo:default:-:any`. Questions exist in both languages, so the locale group is always `any` (D-068).

---

## 4. `rt:` — connections, presence, delivery (Phase 4)

| Key | Type | Content | TTL |
|---|---|---|---|
| `rt:presence:<user_id>` | HASH | instance id → last heartbeat ms of that instance's sessions (deliveries go to instances seen in the last 60 s); online/in-match status joins in Phase 5 | **60 s**, refreshed by heartbeat |
| `rt:sess:<session_id>` | HASH | user_id, resume token hash, installation_id, instance (the client sends its last `seq` per stream when resuming) | **2 min** after disconnect (P) |
| `rt:installation:<installation_id>` | STRING | the installation's current session id — a new `sys.auth` replaces the previous session on any instance (close 4009) | **2 min**, refreshed by heartbeat |
| pub/sub `rt:ch:inst:<instance_id>` | channel | messages to deliver to sockets on that instance (routing via `rt:presence`) | — |

Online status is hidden from others when `profile.settings.show_online_status = false` (filtered when read, not when stored).

---

## 5. `lobby:` — rooms, invites, rematch (Phase 4–5)

| Key | Type | Content | TTL |
|---|---|---|---|
| `lobby:{r:<room_id>}:room` | STRING | the whole live room as JSON: host, status, config, `version` (optimistic concurrency with WATCH/MULTI; also the `room:<id>` stream seq), code, current match, members {user_id, side, seat, ready, locale} (D-067) | **1 h**, refreshed on every change; the idle sweep (15 min without activity in `lobby.rooms.last_activity_at`, D-027) closes rooms, not the TTL |
| `lobby:user:<user_id>:room` | STRING | the user's current room | same as room |
| `lobby:rematch:<match_id>` | HASH | user_id → accepted flag | until 30 s after the match ended (D-027) |
| `lobby:rematch:<match_id>:started` | STRING | guard: the rematch was created once | 30 s |

Invite codes are read from Postgres (`lobby.invite_codes`); a `lobby:code:` cache is not needed at launch volumes (D-067).

---

## 6. `rank:` — leaderboards & rating cache (Phase 7)

### 6.1 Leaderboard sorted sets
`rank:lb:<period>:<period_id>:<entity>:<scope>:<category>` → ZSET member = entity id, score = points.

| Part | Values |
|---|---|
| `period` / `period_id` | `d:2026-10-15` · `w:2026-W42` · `m:2026-10` · `s:<season_no>` · `a` (all-time, no id). Boundaries on Europe/Istanbul time, ISO week starting Monday (D-029). |
| `entity` | `user` · `team` · `couple` |
| `scope` | `global` · `country:<CC>` (friends scope is computed at read time, §6.3) |
| `category` | `all` · `cat:<category_id>` |

Examples: `rank:lb:d:2026-10-15:user:global:all`, `rank:lb:w:2026-W42:team:country:TR:all`, `rank:lb:m:2026-10:user:global:cat:01920c3a-…`.

| Period | TTL after the period ends |
|---|---|
| daily | **3 days** |
| weekly | **14 days** |
| monthly | **40 days** |
| season | **30 days** after the season closes |
| all-time | none (rebuildable from `ranking.points_ledger`) |

Top 1 000 of every board is archived to `ranking.leaderboard_snapshots` before expiry (D-029).

### 6.2 Other ranking keys
| Key | Type | Content | TTL |
|---|---|---|---|
| `rank:rating:<bucket>:<entity_kind>:<entity_id>` | HASH | rating, rd, vol, rated_matches — read by matchmaking via `RatingReader` | **1 h**, refreshed on change |
| `rank:lbdirty` | SET | board keys touched since last snapshot run | none (drained by worker) |

### 6.3 Friends boards
`ZMSCORE` of the user + up to 300 friend ids (D-027) on the matching global board, sorted in the API. No extra keys.

### 6.4 Daily Challenge board (separate family, D-040)
`rank:daily:<yyyy-mm-dd>:<scope>` → ZSET, member = user id, scope `global` or `country:<CC>` (friends at read time).
Score = `correct_count × 1 000 000 + (999 999 − total_time_ms)` — higher is better; `total_time_ms` ≤ 150 000, so the encoding is exact. Equal scores (same correct and same time) are ordered by `completed_at` in the API (category-modes §5.2).
TTL **3 days** after the day ends; top 1 000 archived to `ranking.leaderboard_snapshots` with `board_key = 'daily_challenge:<date>:<scope>'`.

---

## 7. Other modules

| Key | Type | Content | TTL |
|---|---|---|---|
| `id:fuid:<firebase_uid>` | STRING | JSON {user_id, tier, status} — avoids a DB hit on every request; invalidated on tier/status change | **10 min** |
| `content:seen:<user_id>` | ZSET | question_id → last-seen ms; questions seen in the last **30 days** are excluded from selection; trimmed to the newest **2 000** (P) | **30 days**, refreshed |
| `content:daily:<yyyy-mm-dd>` | STRING | cached daily set (ids + localized text) | **48 h** |
| `content:pool:<hash>` | SET | cached eligible-question ids for a frequent filter combination (e.g. ranked + subcategory + tr) | **10 min**, invalidated on verification changes |
| `mod:sanction:<user_id>` | HASH | active sanction kind → ends_at ms (`SanctionChecker`, D-030); key `mod:sanction:<user_id>` with field `none` cached when the user has no sanction | **10 min**, rewritten on every sanction change |
| `rl:<name>:<subject>:<window_id>` | STRING (counter) | fixed-window rate-limit counters (below) | window length |

### 7.1 Rate limits (all through `platform` rate limiter)
| Name | Limit | Source |
|---|---|---|
| `http` per user / per IP | 60 req / 10 s per user, 300 req / min per IP (P) | general API protection |
| `friendreq` | 50 / day | D-027 |
| `challenge` | 20 / hour | D-027 |
| `chat` | 5 / 10 s | D-031 |
| `push` | 10 / hour | D-031 |
| `report` | 20 / day | D-030 |
| `ws_msg` | 20 messages / s per connection (P) | protocol abuse protection |
| `session` | 20 `POST /v1/session` / hour per installation (P) | account-creation abuse |

Rewarded-ad (5/day) and interstitial (1 per 3 matches) limits are checked against Postgres `commerce.ad_rewards` / match counts, not Redis, because they involve money.

---

## 8. If Redis is lost or restarted without data

| Data | Effect | Recovery |
|---|---|---|
| Live matches | cannot continue safely | matches without a reachable owner state are **cancelled** with `server_error` from their Postgres rows (0.3.9); unrated & unscored |
| Queue tickets, rooms, rematch offers | lost | clients get a "please requeue / rejoin" event on reconnect |
| Presence, sessions | lost | rebuilt by the next heartbeat / reconnect |
| Leaderboards | lost | worker rebuilds every current board from `ranking.points_ledger` |
| Caches (auth, sanctions, rating, daily set, pools, seen-questions) | lost | reloaded from Postgres on miss; `content:seen` loss only means a question may repeat |
| Rate-limit counters | reset | acceptable |

---

## 9. Accepted proposals (D-032)
1. Matchmaking ticket lifetime **10 min** (then auto-cancel with "no match found").
2. Reconnect/resume token valid **2 min** after a disconnect (actual in-match reconnect window is a game rule, 0.3.9, and is ≤ this).
3. No-repeat window: questions seen in the last **30 days**, up to **2 000** per user.
4. General API limits: **60 requests / 10 s per user**, **300 / min per IP**; **20 WebSocket messages / s** per connection; **20 session calls / hour** per installation.
