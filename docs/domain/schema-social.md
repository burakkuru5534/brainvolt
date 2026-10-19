# Schema — social, lobby, team (friends, blocks, challenges, invite codes, private rooms, teams, couples, rematch)

- **Roadmap task:** 0.2.5
- **Status:** Accepted (2026-10-15)
- **Modules / Postgres schemas:** `social`, `lobby`, `team`
- **Decisions applied:** D-016 / D-016a / D-017 (restricted accounts), D-018 (couples), D-023 (unverified creator quizzes only in private rooms among friends)
- Conventions: as in `schema-identity.md`. Numeric limits marked **(P)** were proposed and accepted as-is (D-027, §6).

---

## 1. Who may do what

| Action | guest / unverified | full | Notes |
|---|:-:|:-:|---|
| Send / accept friend request, add friend by code | ✗ → upgrade prompt | ✓ | D-016 |
| Block / unblock a user | ✓ | ✓ | safety feature, available to everyone |
| Send direct challenge | ✓ | ✓ | target's `challenges_from` setting applies; rate-limited |
| Rematch / Revenge from the results screen | ✓ | ✓ | with the opponents of that match |
| Create private room | ✓ | ✓ | |
| Join private room by invite code | ✓ | ✓ | except rooms restricted to the host's friends (§3.4) |
| Create / lead a team, be captain | ✗ | ✓ | D-016a |
| Be a member of a persistent team | ✗ | ✓ | team membership is a persistent social tie |
| Form a couple profile | ✗ | ✓ | |
| Random-queue team formats on temporary sides | ✓ | ✓ | D-016a |

Blocks are enforced everywhere: a blocked user cannot send the blocker friend requests, challenges, room invites, team/couple invites or chat messages, and matchmaking never puts them in the same match (Phase 5).

---

## 2. `social` schema

```sql
CREATE SCHEMA social;

CREATE TABLE social.friend_requests (
  id            uuid        PRIMARY KEY,
  from_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  to_user_id    uuid        NOT NULL REFERENCES identity.users(id),
  via           text        NOT NULL CHECK (via IN ('search','invite_code','post_match','profile')),
  status        text        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','accepted','declined','cancelled','expired')),
  created_at    timestamptz NOT NULL,
  responded_at  timestamptz,
  expires_at    timestamptz NOT NULL,                       -- created_at + 30 days (P)
  CHECK (from_user_id <> to_user_id)
);
-- at most one pending request per unordered pair
CREATE UNIQUE INDEX friend_requests_pending_pair_uq
  ON social.friend_requests (LEAST(from_user_id,to_user_id), GREATEST(from_user_id,to_user_id))
  WHERE status = 'pending';
CREATE INDEX friend_requests_inbox_idx ON social.friend_requests (to_user_id) WHERE status = 'pending';
CREATE INDEX friend_requests_outbox_idx ON social.friend_requests (from_user_id) WHERE status = 'pending';

-- Stored as TWO directed rows per friendship (A→B and B→A), written in one transaction:
-- "friends of X" is a single index range scan, which Friends leaderboards rely on.
CREATE TABLE social.friendships (
  user_id     uuid        NOT NULL REFERENCES identity.users(id),
  friend_id   uuid        NOT NULL REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, friend_id),
  CHECK (user_id <> friend_id)
);

CREATE TABLE social.blocks (
  blocker_id  uuid        NOT NULL REFERENCES identity.users(id),
  blocked_id  uuid        NOT NULL REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL,
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX blocks_blocked_idx ON social.blocks (blocked_id);
```

Rules:
- Accepting a request inserts both friendship rows + sets `accepted` in one transaction; if the reverse request is pending, sending one auto-accepts both.
- Blocking deletes any friendship rows and cancels pending requests between the pair (same transaction), and emits `social.UserBlocked` so `lobby`, `team`, `chat`, `matchmaking` react.
- Friend cap: **300** per user (P). Outgoing requests: **50 per day** (P).
- `social` exposes `AreFriends(a,b)`, `FriendIDs(user)`, `IsBlocked(a,b)` (either direction) to other modules.

---

## 3. `lobby` schema

Live room state (members, seats, ready flags) is ephemeral and lives in **Redis** (keys in 0.2.9); Postgres keeps the durable records below for history, invites and audit.

```sql
CREATE SCHEMA lobby;

-- ---------- Invite codes (shared by rooms, challenges, teams, couples, friends) ----------
CREATE TABLE lobby.invite_codes (
  code          text        PRIMARY KEY CHECK (code ~ '^[2-9A-HJ-NP-Z]{6}$'),   -- 6 chars, no 0/O/1/I (P)
  target_kind   text        NOT NULL CHECK (target_kind IN ('room','challenge','team','couple','friend')),
  target_id     uuid        NOT NULL,                       -- room id / challenge id / team id / couple id / user id
  created_by    uuid        NOT NULL REFERENCES identity.users(id),
  max_uses      int         CHECK (max_uses >= 1),          -- NULL = unlimited until expiry
  use_count     int         NOT NULL DEFAULT 0,
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  created_at    timestamptz NOT NULL
);
CREATE INDEX invite_codes_target_idx ON lobby.invite_codes (target_kind, target_id);
-- Expired codes are deleted by a worker job after 30 days so codes can be reused.

-- ---------- Private rooms ----------
CREATE TABLE lobby.rooms (
  id                  uuid        PRIMARY KEY,
  host_user_id        uuid        NOT NULL REFERENCES identity.users(id),
  status              text        NOT NULL DEFAULT 'open'
                      CHECK (status IN ('open','starting','in_match','closed')),
  match_config        jsonb       NOT NULL,                 -- draft config; validated by game (mode matrix, 0.5) before start
  creator_quiz_id     uuid,                                 -- plain id (creator module, 0.2.8)
  friends_of_host_only boolean    NOT NULL DEFAULT false,   -- forced true for unverified creator quizzes (D-023)
  current_match_id    uuid        REFERENCES game.matches(id),
  matches_played      int         NOT NULL DEFAULT 0,       -- rooms can host several matches in a row
  created_at          timestamptz NOT NULL,
  last_activity_at    timestamptz NOT NULL,
  closed_at           timestamptz,
  close_reason        text        CHECK (close_reason IN ('host_closed','idle','all_left','moderation'))
);
CREATE INDEX rooms_open_idx ON lobby.rooms (last_activity_at) WHERE status <> 'closed';

-- ---------- Challenges: direct, rematch, revenge ----------
CREATE TABLE lobby.challenges (
  id                  uuid        PRIMARY KEY,
  kind                text        NOT NULL CHECK (kind IN ('direct','rematch','revenge')),
  challenger_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  challenger_team_id  uuid,                                 -- set when a team/couple challenges as a unit
  challenger_couple_id uuid,
  target_user_id      uuid        REFERENCES identity.users(id),
  target_team_id      uuid,
  target_couple_id    uuid,
  match_config        jsonb       NOT NULL,                 -- validated by game at send time and again at start
  parent_match_id     uuid        REFERENCES game.matches(id),   -- required for rematch & revenge
  status              text        NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','accepted','declined','cancelled','expired','started','failed')),
  match_id            uuid        REFERENCES game.matches(id),   -- set when the match is created
  created_at          timestamptz NOT NULL,
  expires_at          timestamptz NOT NULL,
  responded_at        timestamptz,
  CHECK (num_nonnulls(target_user_id, target_team_id, target_couple_id) = 1),
  CHECK ((kind = 'direct') = (parent_match_id IS NULL))
);
CREATE INDEX challenges_target_user_idx ON lobby.challenges (target_user_id) WHERE status = 'pending';
CREATE INDEX challenges_parent_idx      ON lobby.challenges (parent_match_id) WHERE parent_match_id IS NOT NULL;
CREATE INDEX challenges_challenger_idx  ON lobby.challenges (challenger_user_id) WHERE status = 'pending';
```

### 3.1 Challenge lifetimes (P)
| Kind | Expires | Meaning |
|---|---|---|
| `rematch` | **30 s** after the results screen opens | "Play again" with the same participants and same config; all must accept |
| `revenge` | **24 h** after the parent match ended | Issued only by the **losing** side of the parent match; Revenge Match rules in 0.3.7 |
| `direct` | **24 h** | Sent to an online or offline target (push notification in Phase 9) |

On accept, `lobby` asks `game` to create the match in `waiting`; the ready flow and its timeout (0.3.1) decide whether it starts. If the challenger is no longer available, the match is cancelled with `not_all_ready` and the challenge becomes `failed`.

### 3.2 Direct-challenge permissions
Target's `profile.settings.challenges_from`: `everyone` / `friends` / `nobody`. Blocked pairs always rejected. Rate limit **20 direct challenges per hour** per user (P).

### 3.3 Rooms
- Host creates a room → gets an invite code (`target_kind = 'room'`, expires with the room).
- Members join with the code, the host assigns sides/seats (or "shuffle"), everyone marks ready → `lobby` asks `game` to create the match from `match_config` and the seating.
- After the match the room returns to `open` for another match (`matches_played + 1`); idle rooms close after **15 min** (P).
- Room size limit = format size (max 8 for 4v4).

### 3.4 Unverified Creator Quiz rooms (D-023)
If `creator_quiz_id` points to a quiz that is not verified:
- `friends_of_host_only = true` is forced and cannot be turned off.
- Joining requires being a **friend of the host** (`social.AreFriends`) — guests/unverified accounts therefore cannot join (they have no friends, D-016).
- `game` creates such matches as unrated, `origin = 'private_room'`, `category_mode = 'creator_quiz'`, excluded from leaderboards, SP and quests.

---

## 4. `team` schema

```sql
CREATE SCHEMA team;

CREATE TABLE team.teams (
  id               uuid        PRIMARY KEY,
  name             citext      NOT NULL CHECK (char_length(name) BETWEEN 3 AND 24),
  tag              citext      NOT NULL CHECK (tag ~ '^[A-Za-z0-9]{2,4}$'),   -- short label, e.g. [BVX]
  emblem_id        text        NOT NULL,                       -- built-in emblem preset (D-021 spirit)
  captain_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  status           text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','disbanded')),
  created_at       timestamptz NOT NULL,
  updated_at       timestamptz NOT NULL,
  disbanded_at     timestamptz
);
CREATE UNIQUE INDEX teams_name_active_uq ON team.teams (name) WHERE status = 'active';
CREATE UNIQUE INDEX teams_tag_active_uq  ON team.teams (tag)  WHERE status = 'active';

CREATE TABLE team.memberships (
  team_id    uuid        NOT NULL REFERENCES team.teams(id),
  user_id    uuid        NOT NULL REFERENCES identity.users(id),
  role       text        NOT NULL CHECK (role IN ('captain','member')),
  joined_at  timestamptz NOT NULL,
  left_at    timestamptz,
  leave_reason text      CHECK (leave_reason IN ('left','kicked','team_disbanded','account_deleted')),
  PRIMARY KEY (team_id, user_id, joined_at)
);
CREATE UNIQUE INDEX memberships_current_uq ON team.memberships (team_id, user_id) WHERE left_at IS NULL;
CREATE UNIQUE INDEX memberships_one_captain_uq ON team.memberships (team_id) WHERE left_at IS NULL AND role = 'captain';
CREATE INDEX memberships_user_idx ON team.memberships (user_id) WHERE left_at IS NULL;

CREATE TABLE team.invites (
  id               uuid        PRIMARY KEY,
  team_id          uuid        NOT NULL REFERENCES team.teams(id),
  invited_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  invited_by       uuid        NOT NULL REFERENCES identity.users(id),
  status           text        NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','accepted','declined','cancelled','expired')),
  created_at       timestamptz NOT NULL,
  expires_at       timestamptz NOT NULL,                       -- 7 days (P)
  responded_at     timestamptz
);
CREATE UNIQUE INDEX team_invites_pending_uq ON team.invites (team_id, invited_user_id) WHERE status = 'pending';
CREATE INDEX team_invites_user_idx ON team.invites (invited_user_id) WHERE status = 'pending';

CREATE TABLE team.couples (
  id            uuid        PRIMARY KEY,
  user_low_id   uuid        NOT NULL REFERENCES identity.users(id),   -- smaller uuid of the pair
  user_high_id  uuid        NOT NULL REFERENCES identity.users(id),
  requested_by  uuid        NOT NULL REFERENCES identity.users(id),
  name          citext      CHECK (char_length(name) BETWEEN 3 AND 24),  -- optional couple name
  status        text        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','active','declined','dissolved','expired')),
  created_at    timestamptz NOT NULL,
  requested_at  timestamptz NOT NULL,                     -- the current request (expires after 7 days; reset on reactivation)
  activated_at  timestamptz,
  dissolved_at  timestamptz,
  last_dissolved_at timestamptz,                          -- kept when the same pair reactivates the profile (couple-contest.md §6)
  CHECK (user_low_id < user_high_id),
  CHECK (requested_by IN (user_low_id, user_high_id))
);
-- one row per pair: forming the same couple again reactivates it (D-047)
CREATE UNIQUE INDEX couples_pair_uq ON team.couples (user_low_id, user_high_id);
-- each user can be in at most one pending-or-active couple
CREATE UNIQUE INDEX couples_low_live_uq  ON team.couples (user_low_id)  WHERE status IN ('pending','active');
CREATE UNIQUE INDEX couples_high_live_uq ON team.couples (user_high_id) WHERE status IN ('pending','active');
```

Rules:
- Team size **2–4** members (one captain). Membership limit: a user may belong to at most **3** active teams (P).
- Only full accounts can create, join, captain or form a couple; enforced in `team` service (`ACCOUNT_UPGRADE_REQUIRED`).
- Team invites only to **friends** of the inviter or via a team invite code (P).
- Lineup: for a 2v2 or 3v3 match a team fields a **subset** of its members; the lineup is fixed at match creation (`game.match_participants`).
- Captain leaving → captaincy passes to the longest-standing member; last member leaving → team `disbanded`.
- Couple: one user requests, the other accepts (`pending → active`, request expires after 7 days (P)); either member can dissolve.
- Names and tags pass the profanity/impersonation filter (Phase 8; basic list from Phase 5).
- Team/couple history stays in `game.match_sides.team_id / couple_id`, so disbanded teams keep their past results.

---

## 5. Events produced
| Event | Producer | Main consumers |
|---|---|---|
| `social.FriendRequestSent` | social | notification |
| `social.FriendshipCreated` / `FriendshipRemoved` | social | ranking (friends boards), notification |
| `social.UserBlocked` | social | lobby, team, chat, matchmaking |
| `lobby.ChallengeSent` / `ChallengeAccepted` / `ChallengeDeclined` / `ChallengeExpired` | lobby | notification |
| `lobby.RoomClosed` | lobby | realtime |
| `team.TeamCreated` / `TeamDisbanded` | team | ranking, chat, notification |
| `team.MemberJoined` / `MemberLeft` / `CaptainChanged` | team | chat, ranking, notification |
| `team.CoupleRequested` / `CoupleFormed` / `CoupleDissolved` | team | ranking, notification |

---

## 6. Limits (accepted as proposed — D-027)
| # | Item | Proposal |
|---|---|---|
| 1 | Friend cap / outgoing friend requests per day | 300 / 50 |
| 2 | Friend request expiry | 30 days |
| 3 | Invite code format | 6 characters, no look-alike characters (0/O/1/I) |
| 4 | Rematch window / Revenge window / Direct challenge expiry | 30 s / 24 h / 24 h |
| 5 | Direct challenges per hour | 20 |
| 6 | Idle private room closes after | 15 min |
| 7 | Team size / teams per user | 2–4 members / max 3 active teams |
| 8 | Team invites | friends of the inviter or via team invite code; expire after 7 days |
| 9 | Couple request expiry; one active couple per user | 7 days; yes |
