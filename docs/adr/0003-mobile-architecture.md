# ADR-0003 — Mobile app architecture

- **Status:** Accepted
- **Date:** 2026-10-15
- **Roadmap task:** 0.1.6
- **Related:** ADR-0001 (Flutter 3.47 / Dart 3.13), `repo-layout.md` (`mobile/` tree), `backend-modules.md` (feature ↔ module mapping), D-001/D-002 (TR first, TR+EN)

## Context
The app is a **thin, real-time client**: the backend owns game state, timers, answers and scores. The client must (1) authenticate with Firebase, (2) call REST for everything outside a live match, (3) hold one WebSocket for lobbies, matches, presence and chat, survive network drops and resume a match, (4) render server state quickly and never compute an authoritative result, (5) ship in Turkish and English. It must stay testable as ~15 feature areas grow.

## Decision

### Libraries
| Concern | Choice | Version at decision* | Why |
|---|---|---|---|
| State management **and** dependency injection | **`flutter_riverpod`** + **`riverpod_generator`** (`@riverpod` annotations) + `riverpod_lint` | 3.4.x | Compile-safe providers, async state (`AsyncValue`) for loading/error/data, easy overriding in tests, auto-dispose of match/lobby state when screens close. Doubles as DI → no separate service locator. |
| Navigation | **`go_router`** (flutter.dev) | 18.x | Official, declarative, deep links (invite codes, challenge links, shared result cards), auth redirect guard. Feature-complete & maintained by the Flutter team. |
| REST client | **`dio`** | 5.11.x | Interceptors for Firebase ID token injection & refresh-on-401, request IDs, `Accept-Language`, timeouts, cancellation. |
| WebSocket transport | **`web_socket_channel`** (tools.dart.dev) wrapped by our own `RealtimeClient` in `core/realtime/` | 3.0.x | Official, minimal transport. All protocol logic (auth handshake, heartbeat, seq/ack, reconnect with backoff, resume) lives in our code so it matches the contract in 0.4.3–0.4.6 exactly and is unit-testable with a fake channel. |
| Models / JSON | **`freezed`** + **`json_serializable`** (via `build_runner`) | 4.0.x | Immutable models, unions for WS event types and match states, exhaustive `switch`. |
| Firebase | **`firebase_core`**, **`firebase_auth`**; **`firebase_messaging`** (Phase 9); configured with the FlutterFire CLI | latest compatible | Sign-in and FCM only (ADR-0001). |
| Localization | **`flutter_localizations`** + **`gen-l10n`** (ARB files `app_tr.arb` = template, `app_en.arb`) | SDK | Official, type-safe generated strings. Turkish is the template language (D-001). Device locale picks TR/EN; user can override in settings; chosen locale sent to backend (`Accept-Language`, profile). |
| Small local storage | **`shared_preferences`** | latest | Locale override, onboarding flags, sound settings. No game data is stored locally (server is the source of truth). |
| Logging | **`logging`** (tools.dart.dev) | latest | Levelled logs, silenced in release builds. |
| Testing | **`flutter_test`**, **`mocktail`**, **`integration_test`** (Phase 9 E2E) | SDK / latest | Unit, widget and on-device tests; provider overrides replace real clients. |
| Lint | **`flutter_lints`** + `riverpod_lint` + stricter rules in `analysis_options.yaml` | latest | Run in CI as `flutter analyze` with zero warnings allowed. |

\* Exact versions pinned in `pubspec.lock` in Phase 1 (ADR-0001 policy).

### Code structure (feature-first, three layers)
```
lib/features/<feature>/
├── data/          DTOs (freezed/json), REST & WS message adapters, repository implementations
├── domain/        entities, value objects, repository interfaces — pure Dart, no Flutter imports
└── presentation/  screens, widgets, Riverpod notifiers/controllers (UI state only)
```
Rules:
1. `presentation` → `domain` ← `data`. Widgets never call `dio` or the WebSocket directly.
2. A feature may import another feature's **`domain`** only; never its `data` or `presentation`. Shared pieces move to `core/`.
3. `core/` never imports `features/`.
4. Feature folders mirror backend modules (`backend-modules.md` §1): `auth`, `profile`, `home`, `content` (category browse), `lobby`, `match`, `results`, `matchmaking`, `friends`, `teams`, `leaderboard`, `progression`, `creator`, `moderation` (report), `notifications`, `chat`, `tournament`, `store`.

### Server-authority rules on the client
- The client sends **intents only**: `ready`, `answer(questionId, optionId)`, `leave`, etc. It never sends scores, correctness or timing claims.
- **Correct answers are never sent before the server reveals them**; the client cannot know the right option while answering.
- Countdown timers are **display-only**: driven by server deadlines (`deadline_at` in server time) and a **clock-offset estimate** computed from heartbeat round-trips. If the server says time is up, the UI follows the server.
- Scores, round results and winners are rendered **only from server events**. Optimistic UI is limited to "answer submitted" feedback (button locked), never to correctness or score.
- On reconnect, the match screen rebuilds from the **server snapshot** (0.4.6), discarding local assumptions.

### Real-time client (`core/realtime/`)
- **One** socket per signed-in session, opened after auth, shared by all features through a provider.
- Features subscribe to typed event streams (`match.*`, `lobby.*`, `chat.*`, `presence.*`) — exposed as Riverpod stream providers.
- Reconnect: exponential backoff with jitter, ID-token refresh before re-handshake, resume with last acknowledged sequence number; app lifecycle aware (pause → keep alive briefly; resume → verify connection).
- Fully unit-tested against a fake channel (drop, duplicate, out-of-order, slow network).

### Environments
Build flavors **dev** and **prod** via `--dart-define-from-file=env/<flavor>.json` (API base URL, WS URL). Both use the **single Firebase project `brainvolt`** (D-014) with one Android and one iOS app, both `com.brainvolt.app` (D-012). To keep test sign-ups out of real user data, local development points the app and backend at the **Firebase Auth Emulator** (backend reads `FIREBASE_AUTH_EMULATOR_HOST`); only builds talking to deployed servers use the real `brainvolt` project.

## Alternatives considered
| Rejected | Reason |
|---|---|
| Bloc / Cubit | Solid, but more boilerplate per feature; Riverpod covers state **and** DI and makes test overrides simpler. |
| Provider (package) | Superseded by Riverpod by the same author; weaker compile-time safety. |
| GetX | Mixes routing/state/DI implicitly; hard to test and reason about. |
| `get_it` | Redundant once Riverpod provides DI. |
| `http` package instead of dio | No interceptors; token refresh/retry would be hand-built. |
| `socket_io_client` | Requires a Socket.IO server; our backend speaks plain WebSocket (ADR-0002). |
| Firestore / local DB (drift, Hive) for game data | Conflicts with server authority; nothing that matters is cached locally. |
| Auto-route | go_router is official and sufficient. |

## Consequences
- `build_runner` code generation is part of the workflow (`dart run build_runner build -d`); generated files are committed so CI and fresh clones build without the generator; CI verifies they are up to date.
- WS protocol bugs are caught by unit tests of `RealtimeClient`, not only by device tests.
- Contract changes (0.4) require matching Dart DTO updates in the same commit (monorepo, D-008).

## Resolved decisions
| # | Decision |
|---|----------|
| D-011 | Android + iOS ship together |
| D-012 | Application / bundle ID `com.brainvolt.app` |
| D-013 | Sign-in: Anonymous (guest), Google, Apple, Email/Password via Firebase Auth; Sign in with Apple alongside Google on iOS |
| D-013a | Guests play immediately; upgrade later by **linking** a Google / Apple / Email credential to the same Firebase user (`linkWithCredential`) → same UID → same backend user → no progress lost |
| D-014 | Single Firebase project `brainvolt` |

In-app purchase library (e.g. RevenueCat vs. `in_app_purchase`) is decided in Phase 9 with its own ADR.

## Sources
- https://pub.dev/packages/flutter_riverpod
- https://pub.dev/packages/go_router
- https://pub.dev/packages/web_socket_channel
- https://pub.dev/packages/dio
- https://pub.dev/packages/freezed
