# Game mode combination matrix & validation rules

- **Roadmap tasks:** 0.5.1 (axes), 0.5.2 (matrix), 0.5.3 (declarative validation rules), 0.5.4 (test cases)
- **Status:** Accepted (2026-10-15) — consolidates D-004, D-016/D-016a, D-018, D-023, D-031, D-038, D-039, D-041–D-047; new choices delegated as **D-053**
- **Owner:** `game` config validator (`domain/config`), used by `game.CreateMatch`, `lobby` (room/challenge config edits), `matchmaking` (queue keys), `tournament`; exposed read-only to the app via `GET /v1/meta/modes` (rest.md §4)
- Daily Challenge is **not** a match configuration (D-040): any config with `category_mode = daily_challenge` is rejected.

---

## 1. Axes (0.5.1)

| Axis | Values | Count |
|---|---|---:|
| `match_type` | `quick_duel`, `ranked`, `sudden_death`, `marathon` | 4 |
| `format` | `1v1`, `2v2`, `3v3`, `4v4`, `couple` | 5 |
| `team_answer_mode` | none, `shared_answer`, `individual_turns`, `fast_team_round` | 4 |
| `aggregation` (Fast Team Round only) | none, `average`, `fastest_correct`, `majority` | 4 |
| `category_mode` | none, `blind_challenge`, `category_master`, `mixed_battle`, `creator_quiz`, `revenge_match` | 6 |
| `origin` | `matchmaking`, `challenge`, `private_room`, `rematch`, `revenge`, `tournament` | 6 |
| `is_rated` | derived: `true` ⇔ `match_type = ranked` (D-004) | — |

Raw combinations: 4 × 5 × 4 × 4 × 6 × 6 = **11 520**. After the configuration rules in §3: **1 178 valid**. Ranked has exactly **5** valid configurations (§2.5).

---

## 2. Matrix (0.5.2)

### 2.1 Format × team answer mode
| Format | none | Shared Answer | Individual Turns | Fast Team Round |
|---|:-:|:-:|:-:|:-:|
| 1v1 | ✓ | ✗ R01 | ✗ R01 | ✗ R01 |
| 2v2 / 3v3 / 4v4 | ✗ R01 | ✓ | ✓ | ✓ |
| couple | ✗ R01 | ✓ | ✓ | ✓ |

### 2.2 Match type × category mode
| Category mode | Quick Duel | Ranked | Sudden Death | Marathon |
|---|:-:|:-:|:-:|:-:|
| none | ✓ | ✓ | ✓ | ✓ |
| Blind Challenge | ✓ | ✗ R04 | ✓ | ✓ |
| Category Master | ✓ | ✗ R04 | ✓ | ✓ |
| Mixed Battle | ✓ | ✗ R04 | ✓ | ✓ |
| Creator Quiz | ✓ (quiz length) | ✗ R04 | ✗ R07 | ✗ R07 |
| Revenge Match | ✓ | ✗ R03/R04 | ✓ | ✓ |

### 2.3 Match type × origin
| Origin | Quick Duel | Ranked | Sudden Death | Marathon |
|---|:-:|:-:|:-:|:-:|
| matchmaking | ✓ | ✓ | ✓ | ✓ |
| challenge | ✓ | ✗ R03 | ✓ | ✓ |
| private_room | ✓ | ✗ R03 | ✓ | ✓ |
| rematch | ✓ | ✗ R03 (T01) | ✓ | ✓ |
| revenge | ✓ | ✗ R03 (T01) | ✓ | ✓ |
| tournament | ✓ | ✗ R10 | ✓ | ✓ |

### 2.4 Category mode × origin
| Category mode | matchmaking | challenge | private_room | rematch | revenge | tournament |
|---|:-:|:-:|:-:|:-:|:-:|:-:|
| none | ✓ | ✓ | ✓ | ✓ | ✗ R09 | ✓ |
| Blind Challenge | ✓ | ✓ | ✓ | ✓ | ✗ R09 | ✓ |
| Category Master | ✓ (top-level, R12) | ✓ | ✓ | ✓ | ✗ R09 | ✓ |
| Mixed Battle | ✓ (server picks 3, R13) | ✓ | ✓ | ✓ | ✗ R09 | ✓ |
| Creator Quiz | ✗ R08 | ✓ published only (R14) | ✓ (unverified: friends of host, R14) | ✗ R08 | ✗ R08/R09 | ✗ R08 |
| Revenge Match | ✗ R09 | ✗ R09 | ✗ R09 | ✗ R09 | ✓ | ✗ R09 |

### 2.5 The five Ranked configurations
| # | Format | Team mode | Aggregation | Category | Origin | Rating bucket |
|---|---|---|---|---|---|---|
| 1 | 1v1 | — | — | — | matchmaking | `solo` |
| 2 | 2v2 | Fast Team Round | average | — | matchmaking | `team` |
| 3 | 3v3 | Fast Team Round | average | — | matchmaking | `team` |
| 4 | 4v4 | Fast Team Round | average | — | matchmaking | `team` |
| 5 | couple | Fast Team Round | average | — | matchmaking | `couple` |

### 2.6 Matchmade team formats (delegated D-053)
At launch, **every matchmade team or couple queue (ranked and casual) uses Fast Team Round + `average`** (R05/R06). Shared Answer and Individual Turns are played through private rooms, challenges, rematch/revenge and tournaments. Reason: each extra team mode multiplies the number of queues (type × format × mode × category) and would leave queues empty at launch; Fast Team Round is the only mode that works well with strangers (no coordination needed). Enabling more modes in matchmaking later is a configuration flag, not a schema change.

---

## 3. Declarative validation rules (0.5.3)

The validator evaluates these rules in order and returns **all** violations (`422 MODE_NOT_ALLOWED`, `details.rule_id`). They are stored as data in Go (`[]Rule{ID, Description, Predicate}`), rendered into `/v1/meta/modes`, and each has tests (§5).

### 3.1 Configuration rules (match config only)
```yaml
- id: R01   # D-019/D-039
  rule: format == "1v1"  <=>  team_answer_mode == null
- id: R02   # D-039
  rule: aggregation != null  <=>  team_answer_mode == "fast_team_round"
- id: R03   # D-038
  rule: match_type == "ranked"  =>  origin == "matchmaking"
- id: R04   # D-041
  rule: match_type == "ranked"  =>  category_mode == null
- id: R05   # D-053 (delegated)
  rule: origin == "matchmaking" and format != "1v1"  =>  team_answer_mode == "fast_team_round"
- id: R06   # D-039
  rule: team_answer_mode == "fast_team_round" and origin == "matchmaking"  =>  aggregation == "average"
- id: R07   # D-044
  rule: category_mode == "creator_quiz"  =>  match_type == "quick_duel"
- id: R08   # D-031/D-044
  rule: category_mode == "creator_quiz"  =>  origin in ["private_room", "challenge"]
- id: R09   # D-045
  rule: category_mode == "revenge_match"  <=>  origin == "revenge"
- id: R10   # D-031
  rule: origin == "tournament"  =>  match_type != "ranked"
- id: R11   # D-040
  rule: category_mode != "daily_challenge"
```

### 3.2 Context rules (need data beyond the config)
```yaml
- id: R12   # D-042
  rule: category_mode == "category_master" and origin == "matchmaking"  =>  category is top-level
- id: R13   # D-043
  rule: category_mode == "mixed_battle"  =>  exactly 3 distinct categories with enough eligible questions
- id: R14   # D-023/D-044
  rule: category_mode == "creator_quiz"  =>
          (quiz.status == "published")
          or (origin == "private_room" and room.friends_of_host_only and all participants are friends of host)
- id: R15   # D-045/D-046
  rule: origin == "revenge"  =>  challenger is the losing side of parent, parent ended < 24 h ago, chain length < 3,
                                 home category set, ≤ 3 consecutive questions per category
- id: R16   # D-018/D-047
  rule: format == "couple"  =>  both sides are active couples; no non-couple side
- id: R17   # D-016/D-016a
  rule: side has team_id or couple_id  =>  every member is a full account
- id: R18   # D-038, D-024, D-007
  rule: match_type == "ranked"  =>  questions MC only, verified question + verified translation for every participant locale
- id: R19   # T1 guard, match-lifecycle
  rule: enough eligible questions exist for config × locales × no-repeat window, else NOT_ENOUGH_QUESTIONS
- id: R20   # D-030 / D-034 / D-048
  rule: no participant has a blocking sanction or queue cooldown for this origin/type; no block between any opposing or team-mates pair
- id: R21   # D-044
  rule: creator quiz answer window in {10, 15, 20} s
```

### 3.3 Transformations (applied before validation)
| ID | When | Transformation |
|---|---|---|
| T01 | `origin` ∈ {`rematch`, `revenge`} and parent `match_type = ranked` | `match_type → quick_duel` with Ranked's length & question rules (10 questions, MC only, verified); unrated (D-045) |
| T02 | `origin = revenge` | `category_mode → revenge_match`; copy parent format, team mode, aggregation, window (D-045) |
| T03 | `origin = rematch` | copy the parent config exactly (after T01) |
| T04 | `origin = matchmaking`, team format | set `team_answer_mode = fast_team_round`, `aggregation = average` (D-053) |
| T05 | `team_answer_mode = shared_answer` | `answer_window_ms += 5000` (D-038) |
| T06 | `team_answer_mode = fast_team_round`, aggregation unset outside matchmaking | `aggregation = average` (default, D-039) |

### 3.4 Matchmaking queue key
`<match_type>:<format>:<rating_bucket or casual>:<category_mode or default>:<category_id or ->:<locale_group>` — one queue per valid matchmade configuration (redis-keys §3).

---

## 4. Derived values (not validation, but fixed by the matrix)
| Config | Derived |
|---|---|
| `is_rated` | `match_type == ranked` |
| `rating_bucket` | ranked 1v1 → `solo`; ranked 2v2–4v4 → `team`; ranked couple → `couple`; else null |
| `question_count` | by match type (D-038); creator quiz → quiz length; Sudden Death → open (cap 30) |
| `answer_window_ms` | by match type; creator quiz → author's choice; + 5 s for Shared Answer |
| True/false allowed | not ranked, not daily (D-024) |
| Leaderboard points | only `origin = matchmaking` (D-049) |
| SP | only ranked (D-049) |

---

## 5. Test cases (0.5.4)

### 5.1 Valid configurations (must pass)
| ID | Config |
|---|---|
| MX-V01 | ranked · 1v1 · — · — · matchmaking |
| MX-V02 | ranked · 3v3 · fast_team_round/average · — · matchmaking |
| MX-V03 | ranked · couple · fast_team_round/average · — · matchmaking |
| MX-V04 | quick_duel · 2v2 · shared_answer · blind_challenge · private_room |
| MX-V05 | marathon · 4v4 · individual_turns · mixed_battle · challenge |
| MX-V06 | sudden_death · 1v1 · — · category_master · matchmaking (top-level category) |
| MX-V07 | quick_duel · 3v3 · fast_team_round/majority · — · tournament |
| MX-V08 | quick_duel · 1v1 · — · creator_quiz (published, 20 s) · challenge |
| MX-V09 | quick_duel · couple · shared_answer · creator_quiz (unverified) · private_room, all friends of host |
| MX-V10 | sudden_death · 2v2 · fast_team_round/fastest_correct · revenge_match · revenge |
| MX-V11 | rematch of a ranked 1v1 → after T01: quick_duel (10 MC) · 1v1 · rematch |
| MX-V12 | quick_duel · 4v4 · fast_team_round/average · mixed_battle · matchmaking (server-picked 3 categories) |

### 5.2 Invalid configurations (must fail with the given rule)
| ID | Config | Rule |
|---|---|---|
| MX-I01 | quick_duel · 1v1 · shared_answer | R01 |
| MX-I02 | quick_duel · 2v2 · — | R01 |
| MX-I03 | quick_duel · 2v2 · shared_answer · aggregation=average | R02 |
| MX-I04 | quick_duel · 3v3 · fast_team_round · aggregation missing, validated **without** transformations (with T04/T06 applied it becomes valid) | R02 |
| MX-I05 | ranked · 1v1 · challenge | R03 |
| MX-I06 | ranked · 1v1 · private_room | R03 |
| MX-I07 | ranked · 1v1 · blind_challenge · matchmaking | R04 |
| MX-I08 | quick_duel · 2v2 · individual_turns · matchmaking (no transform) | R05 |
| MX-I09 | quick_duel · 3v3 · fast_team_round/fastest_correct · matchmaking | R06 |
| MX-I10 | marathon · creator_quiz | R07 |
| MX-I11 | quick_duel · creator_quiz · matchmaking | R08 |
| MX-I12 | quick_duel · revenge_match · challenge | R09 |
| MX-I13 | quick_duel · — · revenge | R09 (before T02) |
| MX-I14 | ranked · 1v1 · tournament | R03, R10 |
| MX-I15 | any · daily_challenge | R11 |
| MX-I16 | category_master (sub-category) · matchmaking | R12 |
| MX-I17 | mixed_battle with 2 categories | R13 |
| MX-I18 | unverified creator quiz · private_room · non-friend participant | R14 |
| MX-I19 | revenge issued by the winning side / after 24 h / 4th in chain | R15 |
| MX-I20 | couple format with one side a temporary pair | R16 |
| MX-I21 | persistent team side containing a guest | R17 |
| MX-I22 | ranked match where a participant locale lacks verified translations for selected questions | R18 |
| MX-I23 | creator quiz window 12 s | R21 |
| MX-I24 | participant with `queue_cooldown` joins matchmade match | R20 |

### 5.3 Property tests
- For all 11 520 raw combinations: validator result equals the reference predicate table (generated once, committed as test data).
- Exactly 1 178 configurations pass configuration rules R01–R11; exactly 5 are ranked.
- Every valid configuration has a defined `question_count`, `answer_window_ms`, `rating_bucket` (or null) and queue key (if matchmade).

---

## Delegated decisions (D-053)
- Matchmade team & couple queues (ranked and casual) use **Fast Team Round + average** only at launch; Shared Answer and Individual Turns via rooms, challenges, rematch/revenge and tournaments; more modes in matchmaking = future config flag.
- Tournament matches are never `ranked` (D-031 "unrated by default" made strict, since Ranked requires matchmaking).
- Rematch of a ranked match follows the same conversion as revenge (T01).
