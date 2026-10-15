# Rating, Season Points, leaderboard points, leagues, XP & seasons

- **Roadmap task:** 0.3.10
- **Status:** Accepted (2026-10-15) — delegated decision **D-049** (decision policy in ROADMAP)
- **Owner modules:** `ranking` (rating, SP, leagues, leaderboards, seasons), `progression` (XP, levels, streaks, badges, quests)
- **Builds on:** D-003 (Glicko-2), D-004 (casual unrated), D-016 (restricted accounts hidden), D-023/D-031/D-044 (creator quizzes earn no SP/leaderboard), D-028/D-029 (entities, buckets, periods), D-030 (voids), D-038 (Ranked matchmaking only), D-040/D-046 (Daily Challenge isolated), D-041 (rating + RD), D-047 (couple dual rating), D-048 (forfeits, leaver protection)
- Every number below is part of **ruleset 1** and lives in code as configuration (`Ruleset`), pinned by tests. Changing it later = new ruleset version.

---

## 1. Glicko-2 rating

### 1.1 Parameters (standard values from Glickman's paper)
| Parameter | Value |
|---|---|
| Initial rating μ (display scale) | **1500** |
| Initial RD | **350** |
| Initial volatility σ | **0.06** |
| System constant τ | **0.5** |
| Convergence tolerance ε | 0.000001 |
| Minimum RD | **50** (prevents ratings from freezing) |
| Maximum RD | 350 |

### 1.2 Rating period
- Each **rated match is its own rating period** for every rated entity in it (standard for online games).
- **Inactivity:** before each update (and when ratings are read for matchmaking), RD grows as if each full idle day were an empty Glicko-2 period: `φ' = min(φ_max, √(φ² + σ² · idle_days))` on the Glicko-2 scale. With σ = 0.06 this brings a settled RD back towards 350 over months of inactivity.

### 1.3 Who is updated, against whom
| Match | Updated ratings (bucket) | Opponent used for each update |
|---|---|---|
| Ranked 1v1 | each user (`solo`) | the other user |
| Ranked 2v2/3v3/4v4, temporary sides | each user (`team`) | **composite opponent** = opposing side's members: μ = mean of their `team` ratings, RD = √(mean RD²) |
| Ranked team match, side is a persistent team | each user (`team`) as above **and** the team entity (`team`) | team entity vs. opposing team entity if the other side is a team, otherwise vs. the composite opponent |
| Ranked Couple Contest | couple entity (`couple`) vs. opposing couple entity; each partner's personal `couple` rating vs. composite of the opposing partners (D-047) | as stated |

Outcome scores: **win 1, draw 0.5, loss 0** (forfeit = loss / win, D-048).
New couple / new team entity rating: μ = mean of members' personal ratings of that bucket, RD = **200**, σ = 0.06.

### 1.4 Visibility
- The raw rating is **hidden** from players; players see **league, division and SP** (industry norm; avoids rating anxiety). The profile can show "rated matches played".
- Matchmaking uses μ and RD (D-041); rules in Phase 5.

---

## 2. Season Points (SP) — Ranked only

| Result | SP change |
|---|---|
| Win | **+30** |
| Draw | **+10** |
| Loss | **−15** |
| Win-streak bonus (3rd+ consecutive ranked win) | **+5** per win, max +15 (3rd: +5, 4th: +10, 5th+: +15) |
| Leaver protection: loss while a teammate left/forfeited (D-048) | **0** instead of −15 |
| Voided match (D-030) | the match's SP fully reversed |

- **League floor protection:** SP never drops below the floor of the **current league** (you can't be relegated within a season); divisions inside a league can go down.
- Only `ranked` matches give SP. Casual, private-room, challenge, rematch, revenge, tournament, creator-quiz and Daily Challenge give **0 SP** (D-040, D-044).

## 3. Leagues

| League | SP floor | Divisions |
|---|---|---|
| Bronze / Bronz | 0 | III (0–99), II (100–199), I (200–299) |
| Silver / Gümüş | 300 | III, II, I (each 166 / 167 / 167 SP) |
| Gold / Altın | 800 | III, II, I (each 266 / 267 / 267 SP) |
| Platinum / Platin | 1 600 | III, II, I (each 400 SP) |
| Diamond / Elmas | 2 800 | — (single tier; ranked by SP) |

Same table for users, teams and couples (each entity has its own standing). Every user starts each season in Bronze III.

## 4. Leaderboard points (period boards)

Feed the daily / weekly / monthly / all-time boards (not the season board, which uses SP).

| Source | Leaderboard points |
|---|---|
| Win | **20** |
| Draw | **10** |
| Loss | **0** |
| Each correct round (own correct answer; team entity: side-correct rounds) | **+2** |

**Which matches count:** only **matchmade** matches (`origin = matchmaking`, ranked or casual). Private rooms, challenges, rematches, revenge, tournaments and creator quizzes give **0** leaderboard points — this blocks leaderboard farming between friends (same reasoning as D-038). Daily Challenge has its own board (D-040).

**Category boards:** for each category played in the match: `round(result_points × rounds_in_category / total_rounds) + 2 × correct_rounds_in_category`.

Restricted accounts: written with `public = false` (D-016); team boards only for persistent teams (D-028); couples on both partners' country boards (D-047).

## 5. XP & levels

### 5.1 XP per activity
| Activity | XP |
|---|---|
| Completed head-to-head match (any origin, any type) | **20** base + **5 per own correct answer** + result bonus (**win 15 / draw 8 / loss 0**) |
| Leaver / forfeiter (D-048) | 5 per own correct answer only (rounds actually played); no base, no result bonus |
| Daily Challenge official run | **30 + 5 per correct answer** (practice: 0, D-046) |
| Creator quiz — author on own quiz | 0 (D-044); unverified quiz: 0 for everyone (D-023) |
| Tournament | match XP as above + placement bonus from the tournament's `rewards` |
| Daily quest | 50–100 (per quest definition) |
| Weekly quest | 200–400 |
| Badge earned | per badge tier: 25 / 50 / 100 |
| Rewarded ad "bonus XP after match" (D-031) | +50 % of that match's XP, max +50 |
| Cancelled match | 0 |
| Voided match (D-030) | kept (D-034) |

Premium gives **no** XP boost (no pay-to-progress, D-031 spirit).

### 5.2 Level curve
- XP needed to go from level *L* to *L+1* = **100 + 25 × (L − 1)**. Level 2 at 100 XP, level 10 at 1 800 XP total, level 50 at 34 300 XP total  level 100 at 131 175 XP total.
- **Level cap 100** (prestige/cosmetic beyond 100 can be added later).
- Level is a display/progression stat only; it never affects matchmaking or scoring.

## 6. Play streak
- A day counts if the user **completed at least one head-to-head match or an official Daily Challenge run** that day (Europe/Istanbul day, D-029). Cancelled matches don't count.
- Missing a day resets `current_days` to 0 (no streak freeze at launch).
- Milestone badges at 3, 7, 30, 100, 365 days.

## 7. Badges (initial families)
| Family | Examples (tiers) |
|---|---|
| Wins | first win; 10 / 100 / 1 000 wins |
| Accuracy | 7/7 Quick Duel; 10/10 Ranked; 25/25 Marathon |
| Speed | 5 correct answers under 2 s in one match |
| Streaks | play streak 7 / 30 / 100; answer streak 10 |
| Categories | 50 / 250 / 1 000 correct in a category (per category) |
| Social | first friend, first team match, first couple win, first revenge win |
| Seasons | peak league per season (Bronze → Diamond), season top 100 |
| Daily | 7 / 30 Daily Challenges in a row, perfect 10/10 Daily |

## 8. Quests
- **3 daily + 3 weekly** per user (D-029), drawn from active quest definitions so the three cover different objectives; premium users get **+1 daily** (D-031).
- One free **reroll** per day per user, plus rerolls via rewarded ads (D-031).
- Only matches/runs that are eligible for the objective count; practice Daily runs and unverified creator quizzes never count (D-023, D-046).

## 9. Seasons
| Item | Rule |
|---|---|
| Length | **8 weeks**, starting Monday 00:00 Europe/Istanbul |
| End | `active → closing` at `ends_at`; matches already running finish under the old season |
| Rewards (by **peak** league of the season) | season badge of that league + seasonal avatar frame (Gold and above); global top 100 per entity type: "Top 100" badge |
| SP / league reset | SP → 0, everyone restarts in Bronze III |
| Rating soft reset | μ unchanged; **RD raised to at least 150** (so early matches of the new season move ratings faster) |
| Leaderboards | season boards archived (top 1 000, D-029) |

## 10. Worked examples
- Ranked 1v1 win, 7 correct, 3rd win in a row: SP +30 +5 = **+35**; leaderboard points 20 + 14 = **34**; XP 20 + 35 + 15 = **70**.
- Casual matchmade 3v3 loss, own 4 correct (side 6 correct rounds), persistent team: SP 0; user leaderboard points 0 + 8 = **8**; team entity leaderboard points 0 + 12 = **12**; XP 20 + 20 = **40**.
- Private room Quick Duel win, 5 correct: SP 0, leaderboard 0, XP 20 + 25 + 15 = **60**.

## 11. Tests (→ 0.3.11)
- Glicko-2 implementation reproduces Glickman's published worked example (μ 1500, RD 200 vs. three opponents) to 4 decimals.
- Composite-opponent update for 3v3; draw = 0.5; forfeit = loss.
- RD never below 50 or above 350; inactivity growth applied once per idle day.
- SP floor protection at each league boundary; leaver protection gives 0; void reverses exactly.
- Level curve: totals at levels 2, 10, 50, 100.
- Leaderboard points only for `origin = matchmaking`; category share rounding sums correctly.
