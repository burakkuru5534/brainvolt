# WebSocket protocol

- **Roadmap tasks:** 0.4.3 (handshake, envelope, heartbeat), 0.4.4 (client → server), 0.4.5 (server → client), 0.4.6 (reconnect / resume)
- **Status:** Accepted (2026-10-15) — delegated decisions D-051
- **Server:** `platform/wsserver` + `realtime` module (ADR-0002, `backend-modules.md` §3.3). **Client:** `core/realtime/RealtimeClient` (ADR-0003).
- **Builds on:** conventions.md, timing.md (ping, show_at, grace — D-036), round-lifecycle.md, match-lifecycle.md, edge-cases.md (reconnect windows — D-048), category-modes.md §5 (daily runs), redis-keys.md §2/§2a/§4, D-032 (2 min resume, 20 msg/s)

---

## 1. Principles
1. **One socket per signed-in session**, shared by all features (lobby, match, daily run, chat, presence, notifications).
2. **Clients send intents; the server sends facts.** No client message ever carries a score, correctness or authoritative time.
3. **Ordered, gap-free server streams** with sequence numbers so a reconnecting client can resume exactly where it was.
4. JSON text frames, UTF-8, max **4 KB** per client message (limits.md); server frames may be larger (snapshots).
5. Unknown message types are ignored by both sides (forward compatibility), and logged.

## 2. Connection & authentication (0.4.3)

```
Client                                   Server
  │  GET /v1/ws  (WebSocket upgrade)        │   headers: X-App-Version, X-Platform, Accept-Language
  │ ───────────────────────────────────────▶│
  │  sys.auth {token, installation_id,      │   must arrive within 5 s, else close 4001
  │            resume?}                     │
  │ ───────────────────────────────────────▶│   verify Firebase token, status, version
  │                     sys.welcome {session_id, resume_token,
  │                       server_time, heartbeat_ms, user_id, live: {...}}
  │ ◀───────────────────────────────────────│
```
- `sys.auth.resume` (optional) = `{session_id, resume_token, streams: {"<stream>": last_seq}}` → see §6.
- `sys.welcome.live` lists anything the user is currently part of: `match_id`, `room_id`, `daily_run_id`, so the app can jump straight back in.
- **Token refresh:** Firebase ID tokens expire after ~1 h. The client sends `sys.reauth {token}` before expiry; the server closes with **4003** if the token it holds expires and no reauth arrived.
- **One session per installation:** a new `sys.auth` from the same `installation_id` replaces the old socket (old one closed with 4009). Several installations of the same user may be connected (e.g. phone + tablet); only **one** of them may hold the match seat — the newest connection to `match.join` takes it (old one gets `match.seat_moved`).

### 2.1 Close codes
| Code | Meaning | Client action |
|---|---|---|
| 1000 | normal close | — |
| 1001 | server going away (deploy) | reconnect immediately with resume |
| 4000 | protocol error (malformed, too large) | reconnect; report bug |
| 4001 | auth timeout / missing `sys.auth` | reconnect |
| 4002 | invalid token | refresh token, reconnect |
| 4003 | token expired | refresh token, reconnect |
| 4004 | account suspended / deleted | stop, show notice |
| 4008 | rate limit exceeded (D-032: 20 msg/s) | back off 5 s, reconnect |
| 4009 | replaced by a newer session of this installation | don't reconnect |
| 4026 | app version unsupported | force update |

## 3. Envelope (0.4.3)

**Client → server**
```json
{ "t": "match.answer", "id": "c-17", "d": { "match_id": "…", "round_no": 4, "option_id": "…" } }
```
| Field | Meaning |
|---|---|
| `t` | message type `<area>.<action>` |
| `id` | client message id (unique per session; echoed in `sys.ack` / `sys.error`) |
| `d` | payload |

**Server → client**
```json
{ "t": "round.revealed", "s": "match:0192…", "seq": 57, "ts": 1760542927123, "re": "c-17", "d": { … } }
```
| Field | Meaning |
|---|---|
| `t` | event type |
| `s` | **stream** the event belongs to: `user:<id>` (personal), `match:<id>`, `room:<id>`, `daily:<run_id>`, `chat:<channel_id>` |
| `seq` | per-stream sequence number, starts at 1, no gaps; match streams = Redis `game:{m}:events` entries (redis-keys §2) |
| `ts` | server time, Unix ms |
| `re` | optional: client `id` this event answers |
| `d` | payload |

**Acknowledgements:** every client intent gets either `sys.ack {re}` (accepted) or `sys.error {re, code, message}` (rejected, code from errors.md). Facts that follow (e.g. `round.answer_received`) arrive as stream events.

## 4. Heartbeat & clock (0.4.3, D-036)
| Message | Direction | Payload | When |
|---|---|---|---|
| `sys.ping` | S → C | `{ts}` server ms | every **2 s** while the user is in a match/daily run, every **20 s** otherwise |
| `sys.pong` | C → S | `{ts, client_ts}` echo + client clock | immediately |
| — | — | server computes RTT median of last 10 (timing.md §3); client computes clock offset (timing.md §6) | — |
| `sys.connection` | S → C | `{rtt_ms, quality: good\|weak}` | when quality changes (weak = RTT > 400 ms sustained) |

No pong for **3 consecutive pings** (in match: 6 s; outside: 60 s) → server treats the socket as dead (disconnect handling, edge-cases §2).

## 5. Messages

### 5.1 Client → server (0.4.4)
| Type | Payload | Valid when | Notes |
|---|---|---|---|
| `sys.auth` | `{token, installation_id, resume?}` | first message | §2 |
| `sys.reauth` | `{token}` | any time | token refresh |
| `sys.pong` | `{ts, client_ts}` | after ping | §4 |
| `sys.subscribe` / `sys.unsubscribe` | `{stream, from_seq?}` | any | e.g. open a chat channel; server checks permission |
| `presence.set` | `{status: online\|away}` | any | app foreground/background |
| `room.join` | `{room_id}` | member | sends `room.state` to this session (joining by code is REST); after a match it also returns the room to `open` |
| `room.update_seat` | `{room_id, side_no, seat_no, user_id?}` | room `open` | to an empty seat; host can move anyone (`user_id`), members only themselves |
| `room.ready` / `room.unready` | `{room_id}` | room `open` | |
| `room.start` | `{room_id}` | host, every seat taken, everyone but the host ready | lobby asks game to create the match |
| `match.join` | `{match_id}` | participant | takes the seat on this socket (§2) |
| `match.ready` | `{match_id}` | `waiting` | auto-sent in private rooms (match-lifecycle §4) |
| `match.leave` | `{match_id}` | any live state | confirm dialog on client; final (edge-cases §3) |
| `match.pick_category` | `{match_id, round_no, category_id}` | `pre_round`, picker | Mixed Battle (D-043) |
| `match.answer` | `{match_id, round_no, option_id}` | `question`, expected answerer | 1v1, Individual Turns, Fast Team Round |
| `match.propose` | `{match_id, round_no, option_id}` | `question`, Shared Answer | live teammate picks (D-035) |
| `match.lock` | `{match_id, round_no, option_id}` | `question`, Shared Answer | locks side answer |
| `match.quick_chat` | `{match_id, phrase_id}` | live | preset phrases/emotes (all sides, D-031) |
| `rematch.respond` | `{match_id, accept: bool}` | match end / results screen "Play again", any participant (winner or loser), 60 s; not for private room (`use_room`) or tournament (`tournament`) matches | D-027, D-091 |
| `daily.join` | `{run_id}` | owner | start/continue a daily run |
| `daily.answer` | `{run_id, position, option_id}` | open question | solo run |
| `chat.send` | `{channel_id, body}` or `{channel_id, quick_id}` | member, not muted | persistent team / couple channels (D-031, D-088): 1–300 chars, blocked words masked, slurs refused (`TEXT_REJECTED`), 5 / 10 s (`RATE_LIMITED`), `SANCTIONED` while muted; quick ids `good_luck`, `well_played`, `nice`, `thanks`, `oops`, `lets_go`, `gg`, `rematch` |
| `chat.typing` | `{channel_id}` | member | throttled 1 / 3 s |

### 5.2 Server → client (0.4.5)

**System & personal stream (`user:<id>`)**
| Type | Payload (essentials) |
|---|---|
| `sys.welcome` | `{session_id, resume_token, server_time, heartbeat_ms, live}` |
| `sys.ack` / `sys.error` | `{re}` / `{re, code, message, details?}` |
| `sys.ping` / `sys.connection` | §4 |
| `presence.changed` | `{user_id, status}` (friends, respecting privacy) |
| `notify` | `{inbox_id, category, kind, params, deep_link}` — in-app toast for an inbox item |
| `mm.status` | `{ticket_id, state: searching\|found\|expired\|cancelled, range, wait_ms}` |
| `mm.match_found` | `{match_id, ready_deadline_at}` |
| `challenge.received` / `challenge.updated` | `{challenge}` (as `GET /v1/challenges`) / `{challenge_id, status, match_id?}` |
| `sanction.applied` | `{kind, ends_at}` |

**Room stream (`room:<id>`)**
| Type | Payload |
|---|---|
| `room.state` | `{room}`: full room (members with cards, seats, ready flags, config, status, code, version = seq). Sent after **every** change (D-067: rooms are small, no deltas) |
| `room.match_starting` | `{room_id, match_id}` |
| `room.closed` | `{room_id, reason}` (`host_closed` / `all_left` / `idle`) |

**Match stream (`match:<id>`)**
| Type | Payload | Notes |
|---|---|---|
| `match.created` | config summary, sides, participants, `ready_deadline_at` | |
| `match.participant_ready` | `{user_id}` | |
| `match.starting` | `{starts_at}` | 3 s countdown |
| `match.started` | `{started_at}` | |
| `round.pre_round` | `{round_no, picker_side_no, picker_user_id?, categories: [{id, name}], allowed_category_ids, pick_deadline_at}` | Mixed Battle; names in the recipient's locale; `allowed_category_ids` leave out a 4th pick in a row; `match.snapshot` carries the same as `pre_round` |
| `round.category_picked` | `{round_no, category_id}` | |
| `round.started` | `{round_no, question: {id, prompt, category?}, options: [{id, text}] (shuffled), show_at, deadline_at, answering_user_ids}` | text in the recipient's locale; `category` omitted in Blind Challenge; **never** the correct option |
| `round.answer_received` | `{round_no, user_id}` (others) / `{round_no, user_id, option_id}` (own) | a Shared Answer lock carries `{side_no, option_id, locked: true}` and shows the option to the whole side; the snapshot's `round` adds `answering_user_ids`, `answered_user_ids` and, in Shared Answer, the own side's `proposals` and `locked_option_id` |
| `round.proposals` | `{round_no, proposals: [{user_id, option_id}]}` | own side only, Shared Answer |
| `round.locked` | `{round_no}` | |
| `round.revealed` | `{round_no, correct_option_id, explanation?, category, answers: [{user_id, option_id, response_ms, correct, points: {base, speed, streak, sync?}}], sides: [{side_no, correct, points, score}]}` | |
| `match.halftime` | `{ends_at, scores}` | Marathon |
| `match.participant_status` | `{user_id, status, reconnect_deadline_at?}` | disconnected / back / left / forfeited |
| `match.seat_moved` | `{}` | another device took the seat |
| `match.resumed` | `{reason: "takeover", voided_round_no?}` | edge-cases §4 |
| `match.completed` | `{end_reason, winner_side_no, is_draw, sides, participants, tie_break?, awards_pending: true}` | rating/SP/XP arrive later as `match.awards` (async consumers) |
| `match.awards` | ranking: `{match_id, sp_delta, season_points, league, division, league_up}` (ranked only); progression: `{match_id, xp, xp_total, level, level_up, streak, badges, quests}` | personal, two messages once ranking and progression processed (D-077) |
| `match.cancelled` | `{reason}` | |
| `match.quick_chat` | `{match_id, user_id, phrase_id}` | to every participant; not stored (D-088) |
| `rematch.offer` / `rematch.result` | `{match_id, accepted_user_ids, expires_at}` / `{match_id, accepted, new_match_id?, declined_by?, reason?}` | personal deliveries (no stream seq); the window ends 60 s after the match ended (D-091) |

**Tournaments (personal deliveries, D-090):** `tournament.updated {tournament_id}` — the bracket, entries or status changed, reload; `tournament.match_ready {tournament_id, match_id}` — both sides are present, join the match.

**Daily run stream (`daily:<run_id>`)** — same round events as a match with `run_id` and `position` instead of `match_id` / `round_no`: `daily.question`, `daily.answer_received`, `daily.revealed`, `daily.completed {correct_count, total_time_ms, score, rank?}`, `daily.voided {reason}`.

**Chat stream (`chat:<channel_id>`)**: `chat.message {id, sender_id, body|quick_id, created_at}`, `chat.message_removed {id}`, `chat.typing {user_id}`.

## 6. Reconnect & resume (0.4.6)

1. On disconnect the client reconnects with exponential backoff + jitter: 0.5 s, 1 s, 2 s, 4 s … capped at 10 s (ADR-0003), refreshing the token first if it is close to expiry.
2. It sends `sys.auth {…, resume: {session_id, resume_token, streams: {"match:…": 56, "room:…": 12}}}`.
3. Server checks the resume token (valid **2 min** after disconnect, D-032; Redis `rt:sess:*`).
4. For each stream:
   - if all events after `last_seq` are still in the stream (match: Redis Stream, `MAXLEN ~ 1000`) → **replay** them in order, then continue live;
   - otherwise (or token expired) → send a **snapshot** event (`match.snapshot`, `room.state`, `daily.snapshot`) with the full current state and its `seq`, then continue live.
5. `match.snapshot` contains everything to rebuild the screen: phase, round, question & options **as shown to this user**, deadlines, own answer, visible teammates' info, scores, participant statuses. Never the correct option of an open round.
6. If the reconnect happens after the game's reconnect window (edge-cases §2) the participant is already `forfeited`; the snapshot shows that and the result.
7. Client rule: **on snapshot, discard local assumptions** (ADR-0003).

Events are idempotent on the client by `(s, seq)`: duplicates are dropped, a gap triggers `sys.subscribe {stream, from_seq}` to fetch the missing range (or a snapshot).

**Audience (D-066):** every participant receives every event of a match stream, so sequences stay gap-free; an event meant for another audience (e.g. `round.proposals` of the other side) arrives as `{match_id, round_no, redacted: true}`. Own details are added per recipient: question and option texts in the participant's language, `option_id` in `round.answer_received` for the sender only.

## 7. Fan-out across server instances
`realtime` keeps `rt:presence:<user>` → instance ids; events for a user/stream are published to `rt:ch:inst:<instance>` for each instance holding a relevant socket (redis-keys §4). Match owners publish once per stream; `realtime` handles routing.

---

## Delegated decisions (D-051)
Token in first message (5 s deadline) with `sys.reauth`; one socket per installation, one seat-holding device per match; envelope `t/id/d` and `t/s/seq/ts/re/d`; per-stream gap-free sequences; ack/error for every intent; ping 2 s in play / 20 s idle, dead after 3 missed pongs; replay-or-snapshot resume with 2 min token; awards delivered asynchronously as `match.awards`; backoff 0.5 s → 10 s.
