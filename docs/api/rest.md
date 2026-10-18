# REST endpoint catalogue

- **Roadmap task:** 0.4.2
- **Status:** Accepted (2026-10-15)
- **Conventions:** conventions.md (auth, errors, pagination, idempotency). Error codes: errors.md. Limits: limits.md.
- **Legend:** **Auth** — `pub` public · `any` any signed-in user · `full` full account only (else `ACCOUNT_UPGRADE_REQUIRED`) · `mod` moderator · `adm` admin. **Idem** — `✓` = `Idempotency-Key` required. **Ph** — phase that implements it.
- Real-time actions inside a match or room (ready, answer, picks) are **WebSocket** messages (websocket.md), not REST.
- Request/response bodies here list the essential fields; full JSON schemas are generated from Go types and published as OpenAPI 3.1 (`docs/api/openapi.yaml`) in Phase 1 and kept in sync by CI.

---

## 1. identity & session
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/session` | any | — | 1 | Sync after sign-in/link/verify. Body `{age_confirmed, installation_id, platform, app_version, locale, country_hint?}` → `{user, profile, tier, status, flags}` (`flags`: `created`, `restricted`, `email_verification_required`, `deletion_pending`, `suspended` — D-059). Creates user on first call (schema-identity §5.1). `AGE_CONFIRMATION_REQUIRED` if missing. |
| POST | `/v1/session/abandon-guest` | any (guest) | ✓ | 1 | Link-conflict path (D-015): mark this guest abandoned. |
| GET | `/v1/me` | any | — | 1 | Own user, profile, settings, tier, sanctions summary, entitlements. |
| POST | `/v1/me/deletion` | any | ✓ | 1 | Request account deletion (14-day cooling-off, D-022). |
| DELETE | `/v1/me/deletion` | any | — | 1 | Cancel pending deletion. |
| POST | `/v1/me/devices` | any | — | 1 | Register/refresh device & FCM token `{installation_id, fcm_token?, platform, app_version, locale}`. |
| DELETE | `/v1/me/devices/{installation_id}` | any | — | 1 | Sign-out on this device (revokes push). |

## 2. profile
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/profiles/{user_id}` | any | — | 1 | Public profile (no email), league, level, badges, stats (respecting privacy). |
| GET | `/v1/profiles/by-username/{username}` | any | — | 5 | Lookup for friend search. |
| PATCH | `/v1/me/profile` | any | — | 1 | `display_name`, `avatar_id`, `country_code` (30-day limit), `locale`. Username change: `full` only, 30-day limit. |
| GET / PATCH | `/v1/me/settings` | any | — | 1 | privacy (`challenges_from`, `show_online_status`, `allow_friend_requests`), sound/music/haptics. |
| GET | `/v1/me/history?type=&cursor=` | any | — | 4 | Own match history (and Daily runs with `type=daily`). |
| GET | `/v1/me/stats` | any | — | 7 | Aggregated stats (accuracy, per-category, per-mode). |

## 3. content
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/categories` | any | — | 2 | Categories + sub-categories (localized), eligible-question counts per mode for UI hints. ETag. |
| GET | `/v1/questions?subcategory_id=&status=&locale=&translation_status=&include_retired=&limit=&cursor=` | mod | — | 2 | Question listing for review tools, newest first (never exposes correct answers to players). `locale` limits to questions with that translation and shows its prompt; `translation_status` needs `locale` (D-063). |
| GET | `/v1/questions/{id}` | mod | — | 2 | Full question incl. translations, options, correct option, revisions, stats. |
| POST | `/v1/questions` | mod | ✓ | 2 | Create draft question (with source/license, D-006). |
| PATCH | `/v1/questions/{id}` | mod | — | 2 | Edit (bumps revision; used questions: options immutable, schema-content §4.3). |
| POST | `/v1/questions/{id}/submit` | mod | — | 2 | draft → in_review. |
| POST | `/v1/questions/{id}/review` | mod | — | 2 | `{decision: approve|reject, locale?, reason?}` question- or translation-level (D-007). |
| POST | `/v1/questions/{id}/retire` | adm | — | 2 | Retire (repeating it is harmless). Every mutating question endpoint answers with the full question (D-063). |
| GET | `/v1/credits` | pub | — | 2 | Sources & attribution text for the Credits screen (CC BY-SA, D-005/D-006). |

## 4. game (matches & daily)
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/matches/current` | any | — | 4 | The caller's live match (if any) + WS stream to resume. |
| GET | `/v1/matches/{id}` | any (participant, or public summary) | — | 4 | Result screen data: sides, participants, rounds, answers (after completion), breakdowns. |
| GET | `/v1/matches/{id}/card` | any | — | 9 | Data for the shareable result card. |
| GET | `/v1/daily/today` | any | — | 8 | Today's daily info: date, status of caller's official attempt, practice allowed, board preview. |
| POST | `/v1/daily/runs` | any | ✓ | 8 | Start a run `{kind: official|practice}` → `{run_id, ws_stream}`. `DAILY_ATTEMPT_USED` / `DAILY_ATTEMPT_IN_PROGRESS`. |
| GET | `/v1/daily/runs/{id}` | any (owner) | — | 8 | Run result. |
| POST | `/v1/daily/{date}/beat-my-score` | any | ✓ | 8 | Send "beat my score" to friends `{friend_ids[]}`. |
| GET | `/v1/meta/modes` | any | — | 3 | Mode matrix for UI (allowed combinations, defaults, windows) — generated from the validator (0.5). |

## 5. matchmaking
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/matchmaking/tickets` | any | ✓ | 4–5 | Phase 4: `{match_type: quick_duel, format: 1v1, practice?}` only (first-come pairing; `practice` = dev bot, D-066). Phase 5: queue `{match_type, format, team_answer_mode?, category_mode?, category_id?, party_user_ids?, team_id?, couple_id?}` → ticket. Errors: sanctions, cooldown, already queued/in match. |
| DELETE | `/v1/matchmaking/tickets/current` | any | — | 4 | Leave queue. |
| GET | `/v1/matchmaking/tickets/current` | any | — | 4 | Status, current search range, estimated wait. |

## 6. lobby (rooms, challenges, invite codes)
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/rooms` | any | ✓ | 4 | Create private room `{match_config, creator_quiz_id?}` → room + invite code. |
| GET | `/v1/rooms/{id}` | member | — | 4 | Room state (live updates via WS). |
| PATCH | `/v1/rooms/{id}` | host | — | 4 | Change config (`If-Match`). |
| POST | `/v1/rooms/join` | any | — | 4 | `{code}` → join (friends-of-host rule for unverified quizzes, D-023). |
| POST | `/v1/rooms/{id}/leave` | member | — | 4 | Leave / host closes. |
| POST | `/v1/challenges` | any | ✓ | 5 | Direct challenge `{target_user_id|target_team_id|target_couple_id, match_config}`. |
| POST | `/v1/matches/{id}/revenge` | loser side | ✓ | 8 | Revenge challenge `{home_category_id}` (D-045). |
| GET | `/v1/challenges?box=in|out` | any | — | 5 | Pending challenges. |
| POST | `/v1/challenges/{id}/accept` · `/decline` · `/cancel` | target / challenger | — | 5 | Respond. |
| GET | `/v1/invite-codes/{code}` | any | — | 4 | Resolve a code (what it opens) before joining. |
| POST | `/v1/invite-codes` | full / any per kind | ✓ | 5 | Create codes for friend / team / couple invites. |

## 7. social
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/friends?cursor=` | full | — | 5 | Friend list with presence (respecting `show_online_status`). |
| DELETE | `/v1/friends/{user_id}` | full | — | 5 | Unfriend. |
| POST | `/v1/friend-requests` | full | ✓ | 5 | `{to_user_id | code}`. |
| GET | `/v1/friend-requests?box=in|out` | full | — | 5 | Pending requests. |
| POST | `/v1/friend-requests/{id}/accept` · `/decline` · `/cancel` | full | — | 5 | Respond. |
| GET / POST / DELETE | `/v1/blocks` · `/v1/blocks/{user_id}` | any | — | 5 | List / block / unblock. |

## 8. team & couples
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/teams` | full | ✓ | 5 | Create `{name, tag, emblem_id}`. |
| GET | `/v1/teams/{id}` | any | — | 5 | Team profile, members, standings. |
| PATCH | `/v1/teams/{id}` | captain | — | 5 | Edit (`If-Match`). |
| POST | `/v1/teams/{id}/invites` | captain/member | ✓ | 5 | Invite a friend (D-027). |
| POST | `/v1/team-invites/{id}/accept` · `/decline` | full | — | 5 | Respond. |
| POST | `/v1/teams/{id}/leave` · `/kick/{user_id}` · `/captain/{user_id}` · `/disband` | member / captain | — | 5 | Membership management. |
| GET | `/v1/me/teams` | any | — | 5 | Own teams. |
| POST | `/v1/couples` | full | ✓ | 5 | Couple request `{partner_user_id}` (7-day cooldown, reactivation, D-047). |
| POST | `/v1/couples/{id}/accept` · `/decline` · `/dissolve` | partner | — | 5 | Respond / break up. |
| GET | `/v1/couples/{id}` · `/v1/me/couple` | any | — | 5 | Couple profile. |

## 9. ranking & leaderboards
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/leaderboards?period=&entity=&scope=&country=&category_id=&cursor=&around=me` | any | — | 7 | Every board combination (D-029). |
| GET | `/v1/leaderboards/daily-challenge/{date}?scope=` | any | — | 8 | Daily board (D-040). |
| GET | `/v1/seasons/current` · `/v1/seasons/{id}` | any | — | 7 | Season info, time left, rewards. |
| GET | `/v1/standings/{entity_kind}/{entity_id}` | any | — | 7 | League, division, SP, record (current or `?season_id=`). |
| GET | `/v1/leagues` | any | — | 7 | League table (ETag). |

## 10. progression
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/me/progress` | any | — | 7 | XP, level, next level, play streak. |
| GET | `/v1/me/quests` | any | — | 7 | Active quests & progress. |
| POST | `/v1/me/quests/{assignment_id}/claim` | any | ✓ | 7 | Claim reward. |
| POST | `/v1/me/quests/{assignment_id}/reroll` | any | ✓ | 7 | Daily free reroll / ad reroll. |
| GET | `/v1/badges` · `/v1/me/badges` | any | — | 7 | Catalogue / earned. |

## 11. moderation
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/reports` | any | ✓ | 8 | `{target_kind, target_id, target_locale?, match_id?, reason, comment?}` (20/day, D-030). |
| GET | `/v1/me/sanctions` | any | — | 8 | Own sanctions & notices. |
| POST | `/v1/me/sanctions/{id}/appeal` | any | ✓ | 8 | One appeal per sanction. |
| GET | `/v1/admin/cases?status=&cursor=` | mod | — | 8 | Review queue (priority order). |
| GET | `/v1/admin/cases/{id}` | mod | — | 8 | Case with reports, signals, history. |
| POST | `/v1/admin/cases/{id}/actions` | mod/adm | ✓ | 8 | Take an action (D-030 rules; admin-only actions enforced). |
| POST | `/v1/admin/appeals/{id}/decide` | mod | — | 8 | Decide (not the sanction's creator). |
| GET / POST / DELETE | `/v1/admin/blocked-terms` | adm | — | 8 | Word filter. |
| POST | `/v1/admin/matches/{id}/void` | adm | ✓ | 8 | Void a completed match (D-030). |

## 12. notification
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/me/inbox?cursor=` | any | — | 9 | In-app notifications. |
| POST | `/v1/me/inbox/read` | any | — | 9 | `{ids[] | all: true}`. |
| GET / PATCH | `/v1/me/notification-preferences` | any | — | 9 | Per-category push/in-app; quiet hours (D-031). |

## 13. chat
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/chat/channels/{id}/messages?cursor=` | member | — | 9 | History (sending is via WS `chat.send`). |
| DELETE | `/v1/chat/messages/{id}` | sender / mod | — | 9 | Remove message. |
| POST | `/v1/chat/channels/{id}/read` | member | — | 9 | Move read cursor. |

## 14. tournament
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/tournaments?status=` · `/v1/tournaments/{id}` | any | — | 9 | List / detail with bracket. |
| POST | `/v1/tournaments/{id}/entries` | full | ✓ | 9 | Register (user / team / couple, with lineup). |
| DELETE | `/v1/tournaments/{id}/entries/mine` | full | — | 9 | Withdraw before start. |
| POST | `/v1/tournaments/{id}/check-in` | full | — | 9 | Check in. |
| POST / PATCH | `/v1/admin/tournaments` · `/v1/admin/tournaments/{id}` | adm | ✓ | 9 | Create / manage (D-031). |

## 15. commerce
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| GET | `/v1/store/products` · `/v1/cosmetics` | any | — | 9 | Catalogue (ETag). |
| POST | `/v1/store/purchases/verify` | any | ✓ | 9 | Send store receipt / purchase token → server-side verification → entitlements. |
| POST | `/v1/store/restore` | any | ✓ | 9 | Restore purchases. |
| GET | `/v1/me/entitlements` · `/v1/me/inventory` | any | — | 9 | What the user owns. |
| PUT | `/v1/me/equipped/{slot}` | any | — | 9 | Equip a cosmetic. |
| POST | `/webhooks/app-store` · `/webhooks/play-store` | signature | — | 9 | Store server notifications (verified, idempotent by event id). |
| GET | `/webhooks/admob/ssv` | signature | — | 9 | Rewarded-ad server-side verification callback (D-031 limits). |

## 16. creator
| Method | Path | Auth | Idem | Ph | Purpose |
|---|---|---|---|---|---|
| POST | `/v1/creator/quizzes` | full | ✓ | 8 | Create draft quiz (D-031 limits). |
| GET / PATCH / DELETE | `/v1/creator/quizzes/{id}` | author | — | 8 | Edit (`If-Match`), questions included. |
| POST | `/v1/creator/quizzes/{id}/submit` | author | — | 8 | Send to moderation. |
| GET | `/v1/creator/quizzes?status=published&category_id=&locale=&cursor=` | any | — | 8 | Public discovery (published only). |
| POST / DELETE | `/v1/creator/quizzes/{id}/like` | full | — | 8 | Like / unlike. |

---

**Counts:** ~110 endpoints across 16 groups; Phase 1 implements groups 1–2 (session, me, devices, profile, settings); later phases follow the **Ph** column.
