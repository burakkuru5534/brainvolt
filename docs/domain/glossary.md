# Glossary — canonical domain terms

- **Roadmap task:** 0.2.1
- **Status:** Accepted (2026-10-15; revised same day per D-016a, D-018, D-019)
- **Purpose:** one meaning per word, used identically in docs, database names, Go/Dart code, API fields and UI text.

**How to read this**
- **Code name** = the identifier used in code, DB columns, API fields and enum values (`snake_case` in DB/API, `PascalCase` types in Go/Dart).
- **TR** = the Turkish UI label (Turkish is the primary language, D-001). English UI label = the English term unless noted.
- Entries marked **(working definition)** describe intent only; exact rules are fixed in the referenced Phase 0 task and this glossary is updated then.
- Words to **avoid** are listed so they don't creep back in.

---

## 1. Accounts & identity  *(module: identity, profile)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| User | `user` | Kullanıcı | A BrainVolt account. Identified internally by `user_id` (UUIDv7). One user = one Firebase UID, for life. |
| Firebase UID | `firebase_uid` | — | Firebase Auth's ID for the account. Never shown to players; never used as a foreign key outside `identity`. |
| Sign-in method | `auth_provider` | Giriş yöntemi | `anonymous`, `google`, `apple`, `password` (D-013). A user may have several linked. |
| Guest | `account_tier = guest` | Misafir | User signed in only anonymously (D-013). |
| Unverified | `account_tier = unverified` | Doğrulanmamış | User whose only non-anonymous method is email/password and the email is not yet verified (D-017). |
| Full account | `account_tier = full` | Tam hesap | User with a verified email, Google or Apple method linked. |
| Restricted account | (derived: `guest` or `unverified`) | Kısıtlı hesap | Can play casual and ranked, including random-queue 2v2/3v3/4v4 on temporary sides; **not** on public leaderboards; cannot add friends, nor create or lead teams. Post-match "add friend" / "form team" actions prompt an account upgrade (D-016, D-016a, D-017). |
| Account upgrade | `account_link` | Hesap yükseltme | Linking a Google / Apple / email credential to a guest's Firebase user. Same UID, same `user_id`, no progress lost (D-013a). |
| Link conflict | `link_conflict` | — | The credential already belongs to another user. App warns that guest progress will be lost, then signs into the existing account; no merge (D-015). |
| Account status | `account_status` | Hesap durumu | `active`, `suspended`, `deleted`. Independent of tier. |
| Role | `role` | Rol | `player`, `moderator`, `admin`. |
| Profile | `profile` | Profil | Public face of a user: display name, avatar, country, locale. |
| Display name | `display_name` | Kullanıcı adı | Name shown to others. Unique handle rules fixed in 0.2.2. |
| Locale | `locale` | Dil | `tr` or `en` (D-002). Drives UI text and question translation choice. |
| Country | `country_code` | Ülke | ISO 3166-1 alpha-2; used by the Country leaderboard scope. |
| Device | `device` | Cihaz | An installed app instance with an FCM token (Phase 9; stored from Phase 1). |

## 2. Content  *(module: content, creator)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Category | `category` | Kategori | Top-level topic (e.g. History, Science). |
| Sub-category | `subcategory` | Alt kategori | Topic inside a category (e.g. History → Ottoman Empire). Every question belongs to exactly one sub-category. |
| Question | `question` | Soru | One multiple-choice item: prompt + options + exactly one correct option, in ≥1 locale. |
| Option | `option` | Seçenek / Şık | One answer choice of a question. *Avoid:* "choice", "alternative". |
| Correct option | `correct_option_id` | Doğru şık | Server-side only. Never sent to a client before the reveal. |
| Translation | `question_translation` | Çeviri | The text of a question and its options in one locale. |
| Difficulty | `difficulty` | Zorluk | Scale fixed in 0.2.3 (and later calibrated from answer statistics). |
| Source | `source` | Kaynak | Where a question came from (OpenTDB, BrainVolt team, a creator). Mandatory (D-006). |
| License | `license` | Lisans | Terms under which a sourced question is used, with attribution text. Mandatory (D-006). |
| Verification status | `verification_status` | Doğrulama durumu | `draft`, `in_review`, `verified`, `rejected`. Only `verified` questions can appear in ranked matches (D-007). |
| Question pool | `question_pool` | Soru havuzu | The set of questions eligible for a match after filters (locale, category, status, recently seen). |
| Question set | `question_set` | Soru seti | A fixed, ordered list of questions reused by several matches (Daily Challenge, Creator Quiz, tournament rounds). |
| Daily set | `daily_set` | Günün seti | The question set assigned to one calendar day for Daily Challenge. |
| Creator quiz | `creator_quiz` | Kullanıcı yarışması | A question set authored by a player (Phase 8). |

## 3. Match structure  *(module: game)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Match | `match` | Maç | One complete game from creation to `completed` or `cancelled`. |
| Match configuration | `match_config` | Maç ayarları | Immutable settings of a match: type, format, team answer mode, category mode, rated flag, question count, timers. Validated against the mode matrix (0.5). |
| Match type | `match_type` | Maç türü | `quick_duel`, `ranked`, `sudden_death`, `marathon` — see §4. |
| Format | `format` | Biçim | Player format: `1v1`, `2v2`, `3v3`, `4v4`, `couple` — see §4a. |
| Side | `side` | Taraf | One competing unit **inside a match**. In 1v1 a side has one participant; in 3v3 a side has three. Sides are what win, lose or draw. |
| Participant | `participant` | Oyuncu | A user's seat in a specific match, on exactly one side. |
| Team | `team` | Takım | A **persistent** group profile (2–4 users) that exists outside matches (module `team`). A team *plays as* a side; a side is not necessarily a team (matchmaking may form a side from strangers). *Never use "team" for an in-match side.* Restricted accounts cannot create or lead a team (D-016, D-016a). |
| Temporary side | `side.team_id = null` | Geçici takım | A side formed by matchmaking from individual queue entries (possibly strangers, possibly guests). Exists only for one match. In UI it may be called "takım"; in data and code it is a side with no team profile (D-016a). |
| Couple | `couple` | Çift | A persistent two-user profile used for Couple Contest. |
| Party | `party` | Grup | Users queueing together for matchmaking (Phase 5). Temporary. |
| Rated match | `is_rated = true` | Dereceli | Changes rating (Glicko-2). Only match type `ranked` is rated. |
| Casual match | `is_rated = false` | Dereceli olmayan | Never changes rating (D-004). SP/XP effect fixed in 0.3.10. |
| Round | `round` | Tur | One question asked to all sides (or to one side's player, depending on team answer mode) and resolved. |
| Question slot | `match_question` | — | A question assigned to a specific round of a specific match, with its order number. |
| Answer window | `answer_window` | Cevap süresi | Server-defined time during which answers to the current round are accepted. |
| Deadline | `deadline_at` | — | Server timestamp when the answer window closes. Clients display it; the server enforces it. |
| Grace window | `grace_ms` | — | Small server-side allowance for network delay after `deadline_at` (value fixed in 0.3.3). |
| Answer | `answer` | Cevap | A participant's submitted option for a round, timestamped **by the server on receipt**. One per participant per round (unless team mode says otherwise). |
| Response time | `response_ms` | Cevap süresi | Server-measured time from question dispatch to answer receipt (latency-adjusted per 0.3.3). Used for speed bonus. |
| Reveal | `reveal` | Cevap açıklama | Moment the server publishes the correct option and round scores. |
| Score | `score` | Puan | Points earned **inside one match**. Computed only by the server. *Not* the same as rating or SP. |
| Speed bonus | `speed_bonus` | Hız bonusu | Extra score for faster correct answers (0.3.4). |
| Answer streak | `answer_streak` | Doğru serisi | Consecutive correct answers **within a match** (0.3.4). Different from play streak (§7). |
| Result | `match_result` | Sonuç | Final outcome per side: `win`, `loss`, `draw`, plus final scores and stats. |
| Winner | `winner_side` | Kazanan | Side with the winning result; null on draw or cancel. |
| Forfeit | `forfeit` | Hükmen mağlubiyet | A side loses by leaving or failing to reconnect in time (0.3.9). |
| Disconnect | `disconnect` | Bağlantı kopması | A participant's socket drops; match continues per 0.3.9 rules. |
| Reconnect window | `reconnect_window` | Yeniden bağlanma süresi | Time a disconnected participant has to resume before forfeit rules apply. |
| Snapshot | `match_snapshot` | — | Full current match state the server sends on (re)connect. |
| Match owner | `match_owner` | — | The single server process that runs a live match (single-writer, `backend-modules.md` §3.4). |
| Rematch | `rematch` | Tekrar oyna | New match with the same participants and configuration, proposed after a match ends; both sides must accept. |

## 4. Match types  *(rules: 0.3.5)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Quick Duel | `quick_duel` | Hızlı Düello | Short casual match with a fixed number of questions. |
| Ranked | `ranked` | Dereceli | Rated match: verified questions only (D-007), rating and Season Points change. |
| Sudden Death | `sudden_death` | Ani Ölüm | Continues until a side answers wrong (or slower) while the other is right (working definition). |
| Marathon | `marathon` | Maraton | Long match with many questions; endurance format (working definition). |

## 4a. Player formats  *(D-018)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| 1v1 | `1v1` | 1'e 1 | One participant per side. |
| 2v2 / 3v3 / 4v4 | `2v2`, `3v3`, `4v4` | 2'ye 2 / 3'e 3 / 4'e 4 | Two, three or four participants per side. A side is either a persistent team or a temporary side formed by matchmaking (D-016a). |
| Couple Contest | `couple` | Çiftler Yarışması | Specialised **2v2** format in which each side is a **couple profile** (two paired users). Not a category mode. Specific rules: 0.3.8. |

## 5. Team answer modes  *(rules: 0.3.6; apply when a side has >1 participant)*

| Term | Code name | TR | Definition (confirmed D-019) |
|---|---|---|---|
| Shared Answer | `shared_answer` | Ortak Cevap | The side submits **one collective decision** per question (D-019). How the decision is reached and submitted: 0.3.6. |
| Individual Turns | `individual_turns` | Sırayla | Questions **cycle through the side's members in order**; the member whose turn it is answers for the side (D-019). |
| Fast Team Round | `fast_team_round` | Hızlı Takım Turu | **All members answer simultaneously**; the side's round score comes from a predefined **aggregation rule** — e.g. fastest correct, majority, or total correct (D-019). Which rule(s) and whether it is configurable: 0.3.6. |

## 6. Category modes  *(rules: 0.3.7; full features: Phase 8)*

| Term | Code name | TR | Definition (working) |
|---|---|---|---|
| Blind Challenge | `blind_challenge` | Kör Meydan Okuma | Category is hidden until each question is shown. |
| Category Master | `category_master` | Kategori Ustası | All questions come from one chosen category/sub-category. |
| Mixed Battle | `mixed_battle` | Karışık Savaş | Questions drawn across several categories (e.g. each side contributes picks). |
| Daily Challenge | `daily_challenge` | Günlük Meydan Okuma | Everyone plays the same daily set. |
| Creator Quiz | `creator_quiz` | Kullanıcı Yarışması | Match played on a published creator quiz. |
| Revenge Match | `revenge_match` | Rövanş | Challenge issued by the losing side to the same opponent(s) after a match, with its own rules (vs. a plain rematch). |

## 7. Ranking & progression  *(modules: ranking, progression; formulas: 0.3.10)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Rating | `rating` | Beceri puanı | Glicko-2 skill estimate (D-003), made of **rating** μ, **deviation** RD, **volatility** σ. Changed only by rated matches. |
| MMR | (alias of rating) | — | "Matchmaking rating". In docs and code always say **rating**; "MMR" only in player-facing or roadmap text. |
| Rating bucket | `rating_bucket` | — | Which rating a match affects (e.g. solo vs team); defined in 0.3.10. |
| Season | `season` | Sezon | A fixed time period for SP, leagues and season leaderboards. |
| Season Points (SP) | `season_points` | Sezon Puanı | Leaderboard currency earned in a season; reset at season end. Separate from rating. |
| League | `league` | Lig | Tier from SP/rating: Bronze, Silver, Gold, Platinum, Diamond (exact list & thresholds 0.3.10). TR: Bronz, Gümüş, Altın, Platin, Elmas. |
| Seasonal reset | `season_reset` | Sezon sıfırlama | End-of-season process: archive standings, distribute rewards, reset SP, soft-reset rating deviation. |
| XP | `xp` | Deneyim (XP) | Progression points from playing; never reset; drives level. |
| Level | `level` | Seviye | Derived from total XP. |
| Play streak | `play_streak` | Oyun serisi | Consecutive days with at least one completed match. Different from answer streak. |
| Badge | `badge` | Rozet | Permanent achievement. |
| Quest | `quest` | Görev | Time-limited objective (daily / weekly / seasonal) with a reward. |
| Leaderboard | `leaderboard` | Sıralama | Ranked list defined by 4 filters: **period** (`daily`, `weekly`, `monthly`, `all_time`), **entity** (`individual`, `team`, `couple`), **scope** (`friends`, `country`, `global`), **category** (optional). Restricted accounts excluded from public scopes (D-016). |

## 8. Social & matchmaking  *(modules: social, team, lobby, matchmaking)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Friend | `friendship` | Arkadaş | Mutual, accepted relationship. Full accounts only (D-016). |
| Friend request | `friend_request` | Arkadaşlık isteği | Pending request; becomes a friendship on accept. |
| Block | `block` | Engelleme | One-way; blocked users can't challenge, invite or message the blocker. |
| Direct challenge | `challenge` | Meydan okuma | Invitation from one user/side to a specific user/side to play a configured match. |
| Invite code | `invite_code` | Davet kodu | Short shareable code that lets someone join a room or challenge. |
| Private room | `room` | Özel oda | Pre-match space where invited users gather, pick config and get ready. |
| Host | `room_host` | Oda sahibi | User who created a room; can change config before start. |
| Queue | `mm_queue` | Sıra | Matchmaking waiting list for one (match type, format, rating bucket) combination. |
| Ticket | `mm_ticket` | — | One user's or party's entry in a queue. |
| Search range | `search_range` | Arama aralığı | Allowed rating difference for a ticket; **expands** over waiting time (Phase 5). |
| Ready | `ready` | Hazır | Participant has confirmed they are ready; all ready → match starts. |

## 9. Moderation & anti-cheat  *(module: moderation)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Report | `report` | Şikayet | A user's complaint about a question (wrong, offensive, unclear) or a user. |
| Review | `review` | İnceleme | Moderator decision on a report or a question in `in_review`. |
| Sanction | `sanction` | Yaptırım | Action on a user: warning, ranked ban, chat mute, suspension. |
| Cheat signal | `cheat_signal` | Hile sinyali | Automatically detected suspicious pattern (e.g. impossible response times). Signals feed review; they never change a result automatically without a rule from 0.3.9/Phase 8. |

## 10. Platform & monetization  *(modules: notification, chat, tournament, commerce)*

| Term | Code name | TR | Definition |
|---|---|---|---|
| Notification | `notification` | Bildirim | Push (FCM) and/or in-app message to a user. |
| Team chat | `chat_message` | Takım sohbeti | Messages within a team's channel. |
| Tournament | `tournament` | Turnuva | Organised competition of many matches with registration and a bracket. |
| Bracket | `bracket` | Eşleşme tablosu | Structure deciding who plays whom in each tournament round. |
| Premium | `subscription` | Premium | Paid subscription granting entitlements. **Never affects match outcome** (no pay-to-win). |
| Entitlement | `entitlement` | Hak | A permission granted by premium or purchase (e.g. no ads, cosmetic access). |
| Cosmetic | `cosmetic` | Kozmetik | Visual-only item (avatar frame, theme, emote). |
| Shareable result card | `result_card` | Paylaşılabilir sonuç kartı | Image summarising a match result, generated on the device from server results. |

## 11. Architecture terms

| Term | Definition |
|---|---|
| Module | A business area under `backend/internal/modules/` (`backend-modules.md`). |
| Domain event | Fact published by a module after a state change (e.g. `game.MatchCompleted`). |
| Outbox | Delivery of domain events through River jobs inserted in the same DB transaction (ADR-0002). |
| Server authority | The rule that only the backend decides state, timers, correctness, scores and winners. |

---

## Words to avoid
| Avoid | Use instead | Why |
|---|---|---|
| "team" for an in-match group | **side** | Team is a persistent profile. |
| "player" in data models | **user** (account) / **participant** (in a match) | "Player" is ambiguous; fine in UI text. |
| "points" alone | **score** (in match), **SP** (season), **XP** (progression), **rating** (skill) | Four different numbers. |
| "streak" alone | **answer streak** or **play streak** | Two different counters. |
| "game" for one match | **match** | `game` is the module name. |
| "level" for difficulty | **difficulty** | Level is XP-based. |
| "answer" for an option | **option** | An answer is a submission. |
| "rematch" for Revenge Match | keep them separate | Different flows and rules. |

## Open points (to settle in the referenced tasks)
- Working definitions in §4 and §6 are confirmed or corrected in 0.3.5 and 0.3.7. §4a and §5 confirmed by D-018 / D-019; detailed rules in 0.3.6 and 0.3.8.
- Fast Team Round aggregation rule(s) — 0.3.6.
- League list (Bronze → Diamond, any tiers or divisions in between) — 0.3.10.
