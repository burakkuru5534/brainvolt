# Couple Contest rules

- **Roadmap task:** 0.3.8 (engine rules; live feature in Phase 6)
- **Status:** Draft for review (2026-10-15)
- **Owner modules:** `game` (match rules), `team` (couple profiles, schema-social §4), `ranking` (couple rating & boards), `matchmaking` (couple queues, Phase 5)
- **Builds on:** D-016 (couples need full accounts), D-018 (Couple Contest = specialised 2v2 player format), D-027 (one live couple per user, 7-day request expiry), D-028/D-029 (rating buckets solo/team/couple; team country rule), D-039 (team answer modes, same-scale rule, Ranked fixed to Average), D-041 (Ranked: no category mode; rating + RD matchmaking)
- Values marked **(P)** are proposals (§9).

---

## 1. What Couple Contest is
A **2v2 match in which each side is a couple profile** — two users who formed a couple (`team.couples.status = active`). It is a **player format** (`format = couple`, D-018), so it combines with match types, team answer modes and category modes like any other format, with the restrictions below.

| Rule | Value |
|---|---|
| Sides | exactly 2, each = one **active couple** (`match_sides.couple_id` set); both members of that couple play on that side |
| Opponents | **couples only** — a couple never plays against a temporary pair or a 2-person team (P) |
| Accounts | all four players full accounts (D-016); no block between any two of the four |
| Same user on both sides | impossible (a user is in at most one live couple, D-027) |

---

## 2. Allowed combinations

| Axis | Allowed |
|---|---|
| Match types | Quick Duel, **Ranked**, Sudden Death, Marathon |
| Team answer modes | all three: Shared Answer, Individual Turns, Fast Team Round (D-039) |
| Ranked couple matches | **Fast Team Round + Average only** (D-039), no category mode (D-041), matchmaking only (D-038) |
| Category modes (unrated) | Blind Challenge, Category Master, Mixed Battle, Creator Quiz (room rules apply), Revenge Match |
| Origins | couple matchmaking queue, direct couple-vs-couple challenge, private room (both couples present), rematch / revenge, tournaments with `entity_kind = couple` |

---

## 3. Couple-specific twist — Sync Bonus (P)

Couples are about being in tune, so Couple Contest adds one rule on top of ruleset 1:

> **Sync Bonus:** in a round where **both partners answer correctly**, the side gets **+20 points** — applied **only in Fast Team Round**, where partners answer independently without seeing each other's choice (D-035).

- Counted after the aggregation rule; the side's round points are still **capped at 260** (same-scale rule, D-039).
- Shown at reveal as "Uyum bonusu +20" / "Sync bonus +20".
- Applies in Ranked couple matches too (both couples have the same chance; it rewards consistent pairs).
- Shared Answer and Individual Turns have no sync bonus (one answer per side — nothing to sync).

Example (Fast Team Round, Average, difficulty 3, window 15 s): partner A correct at 3 000 ms (180), partner B correct at 6 000 ms (160) → average 170 + sync 20 = **190**.

---

## 4. Rating (Ranked only)

Two ratings change after a Ranked Couple Contest (both in bucket `couple`, D-029):

| Rating | Entity | Used for |
|---|---|---|
| **Couple rating** | the couple profile (`entity_kind = couple`) | couple matchmaking, Couples leaderboards, couple's league |
| **Personal couple rating** | each user (`entity_kind = user`, bucket `couple`) | profile stat; **starting rating of any new couple** = average of the two partners' personal couple ratings (RD set high so it settles fast) (P) |

Glicko-2 parameters and the team-vs-team update formula are fixed in 0.3.10. Matchmaking uses the **couple rating and RD** (D-041).

---

## 5. Leaderboards & standings
- **Couples boards** (entity `couple`) for every period / scope / category combination (D-029), fed only by matches where the side was a couple.
- **Country scope (P):** a couple appears on the country boards of **both** partners' countries (one entry per distinct country), because a couple has no captain.
- Each partner's **individual** boards and XP also update from Couple Contest matches (the same as for other team matches); the individual `team`/`solo` ratings do **not** change.
- Season standings, leagues and season rewards exist for the couple entity like for teams (schema-ranking).

---

## 6. Couple lifecycle effects on play

| Event | Effect |
|---|---|
| Couple formed (request accepted) | can queue / be challenged as a couple; rating seeded per §4 |
| Partner offline | the couple cannot queue or accept challenges; the online partner can still play other formats |
| One partner is in another live match | couple cannot start a match (one live match per user) |
| Couple **dissolved** | removed from live Couples boards; history, snapshots and past results stay (`match_sides.couple_id`) |
| Same two users form a couple **again** | the **old couple profile is reactivated** (same id, same rating & history) instead of a fresh one — prevents resetting a bad rating by dissolving and re-forming (P) |
| Cooldown after dissolving | a user can form a **new** couple with a different partner only after **7 days** (P) — prevents boosting by partner hopping |
| Partner deletes account | couple dissolved automatically (`leave_reason = account_deleted`) |
| Sanction on a partner | `ranked_ban` / `matchmaking_ban` blocks the couple from those queues; `suspension` blocks all couple play |

Schema impact (applied in this task): `team.couples` gains `last_dissolved_at` and a reactivation path; the unique "one live couple" rule is unchanged.

---

## 7. Disconnects
Couple Contest uses the general team disconnect and forfeit rules (0.3.9); with only two players per side, a side with **no connected partner** after the reconnect window forfeits.

---

## 8. Invariants & tests (→ 0.3.11)
1. Every Couple Contest side has an active `couple_id`, and its two participants are exactly that couple's members.
2. A couple never faces a non-couple side.
3. Ranked couple match ⇒ Fast Team Round + Average, no category mode, origin matchmaking.
4. Sync Bonus only in Fast Team Round, only when both partners are correct, round points ≤ 260.
5. Ranked couple match updates the couple entity rating and both partners' personal couple ratings, nothing else.
6. Re-forming the same pair reactivates the old couple id; new partner within 7 days of dissolving is rejected.

---

## 9. Proposals to confirm
1. **Couples play only couples** (no couple vs. temporary pair or 2-person team).
2. **Sync Bonus +20** when both partners are correct in Fast Team Round (within the 260 cap), also in Ranked.
3. **Two ratings:** couple rating (matchmaking, boards) + each partner's personal couple rating; new couples start from the partners' average.
4. **Country boards:** a couple counts on both partners' countries.
5. **Anti-abuse:** re-forming the same pair restores the old couple profile; 7-day wait before forming a couple with someone new.
