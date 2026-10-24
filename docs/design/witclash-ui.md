# BrainVolt UI modernization (D-099)

Designed under the working name "WitClash"; the owner kept the name
**BrainVolt** (launcher name, texts, word mark "Brain" + "Volt", the
BV mark in `mobile/assets/brand/brainvolt_logo.svg`).

The app moves from the D-085 look (light/dark) to the new design system: one dark theme, a deep navy/indigo star field with a
purple → orange glow, frosted glass cards, purple → orange gradient
buttons, and the "BrainVolt" word mark. Platform identifiers
(`com.brainvolt.app`, the Dart package `brainvolt`, Firebase project, API
hosts) do not change; only the user-visible name does.

## Approach

Everything flows from shared pieces, so no screen gets a one-off patch:

| Piece | Where | What it does |
| --- | --- | --- |
| Tokens | `lib/app/theme.dart` (`BvColors.witclash`) | background, glass fill/border, ink, muted, badge, gradient (purple → magenta → orange), accent, gold, success, danger, glow, surface for dialogs |
| Theme | `buildTheme()` | single dark `ThemeData`: transparent scaffolds/app bars, glass inputs, gradient-friendly buttons, dialogs/sheets/snack bars/menus/chips/tabs/segmented buttons/switches/nav bar in BrainVolt colours |
| Background | `WcBackground` (`lib/core/widgets/wc_background.dart`) | gradient + still star field; sits under the navigator (`MaterialApp.builder`) and is also wrapped around every Material page transition, so every route — and any future one — gets it |
| Kit | `lib/core/widgets/bv.dart` | `BvCard` (glass), `GradientButton`, `IconBadge`, `ActionCard`, `ChoiceTile`, `OptionTile`, `AvatarBubble`, `EmptyState`, `BrandMark` / `BrandLogo` / `BrandHero`, `SectionTitle(action:)`, `GradientProgressBar`, `Pill`, `GlassSection`, `TileCard`, `WcSegmented`, `StatBox`, `CountdownRing`, `NoticeBar`, `LoadingView`, `ErrorView` |
| Strings | `app_tr.arb`, `app_en.arb` | "BrainVolt" → "BrainVolt" in user-visible text |
| Platform | `AndroidManifest.xml` `android:label`, `Info.plist` `CFBundleDisplayName` | launcher name "BrainVolt" (ids unchanged) |

The bottom bar (`MainNavBar`) sits on Home, Tournaments, Leaderboards and
Profile. A tap goes to that screen as the new root (`context.go`); Back on
a tab other than Home returns Home. The screens keep their routes, so deep
links and pushes from elsewhere still work (then with a back arrow).

The light/dark choice in Settings is removed: BrainVolt is dark only. The
stored `theme_mode` preference is ignored.

## Screen inventory

Taken from `lib/app/router.dart` and every `Scaffold`, dialog and bottom
sheet in `lib/`. ✅ = moved to the BrainVolt design (see the final
checklist below for how each one was verified).

### Routes

| # | Route | Screen (file) | Status |
| --- | --- | --- | --- |
| 1 | `/splash` | `SplashScreen` (auth/splash_screen.dart) | ✅ |
| 2 | `/welcome` | `AgeGateScreen` (auth/age_gate_screen.dart) | ✅ |
| 3 | `/account-status` | `AccountStatusScreen` (auth/account_status_screen.dart) | ✅ |
| 4 | `/` | `HomeScreen` (home/home_screen.dart) — top bar, player card, daily card, mode grid, shortcuts, stats, bottom bar | ✅ |
| 5 | `/settings` | `SettingsScreen` (profile/settings_screen.dart) — glass sections; appearance option removed | ✅ |
| 6 | `/profile` | `ProfileScreen` (profile/profile_screen.dart) — header, stat boxes, achievements, edit form | ✅ |
| 7 | `/account` | `AccountScreen` (auth/account_screen.dart) | ✅ |
| 8 | `/account/email-sign-up` | `EmailAuthScreen(signUp: true)` | ✅ |
| 9 | `/account/email-sign-in` | `EmailAuthScreen(signUp: false)` | ✅ |
| 10 | `/account/verify-email` | `VerifyEmailScreen` | ✅ |
| 11 | `/credits` | `CreditsScreen` (content/credits_screen.dart) | ✅ |
| 12 | `/friends` | `FriendsScreen` (social/friends_screen.dart) | ✅ |
| 13 | `/challenges` | `ChallengesScreen` (lobby/challenges_screen.dart) | ✅ |
| 14 | `/modes` | `ModePickerScreen` (match/mode_picker_screen.dart) — mode tiles, topic chips, gradient search | ✅ |
| 15 | `/teams` | `TeamsScreen` (team/teams_screen.dart) | ✅ |
| 16 | `/teams/:id` | `TeamScreen` (team/teams_screen.dart) | ✅ |
| 17 | `/rooms/:id` | `RoomScreen` (lobby/room_screen.dart) — private room and matchmaking lobby bodies — gradient code card, side headings, seat cards, ready/start; matchmaking lobby | ✅ |
| 18 | `/leaderboards` | `LeaderboardsScreen` (progress/leaderboards_screen.dart) — scope tabs, podium with crown, ranked rows, pinned own row | ✅ |
| 19 | `/tournaments` | `TournamentsScreen` (tournament/tournament_screens.dart) | ✅ |
| 20 | `/tournaments/new` | `CreateTournamentScreen` | ✅ |
| 21 | `/tournaments/:id` | `TournamentScreen` | ✅ |
| 22 | `/quests` | `QuestsScreen` (progress/quests_screen.dart) — quest cards, gradient progress, empty state | ✅ |
| 23 | `/badges` | `BadgesScreen` (progress/badges_screen.dart) — medal grid, empty state | ✅ |
| 24 | `/daily` | `DailyScreen` (daily/daily_screen.dart) | ✅ |
| 25 | `/daily/runs/:id` | `DailyRunScreen` (daily/daily_run_screen.dart) | ✅ |
| 26 | `/sanctions` | `SanctionsScreen` (moderation/sanctions_screen.dart) | ✅ |
| 27 | `/quizzes` | `QuizzesScreen` (creator/quizzes_screen.dart) | ✅ |
| 28 | `/quizzes/new` | `QuizEditorScreen` (creator/quiz_editor_screen.dart) | ✅ |
| 29 | `/quizzes/:id` | `QuizEditorScreen(id)` | ✅ |
| 30 | `/inbox` | `InboxScreen` (notifications/inbox_screen.dart) | ✅ |
| 31 | `/chat/team/:id` | `ChatScreen` (chat/chat_screen.dart) | ✅ |
| 32 | `/chat/couple/:id` | `ChatScreen` | ✅ |
| 33 | `/settings/notifications` | `NotificationSettingsScreen` | ✅ |
| 34 | `/play` | `QueueScreen` (match/queue_screen.dart) — queue and practice — radar with brand mark, elapsed ring, topic pill, error view | ✅ |
| 35 | `/match/:id` | `MatchScreen` (match/match_screen.dart) — question, options, countdown, reveal, round transitions, team proposals — glass scoreboard with VS, countdown ring, gradient start countdown, option tiles, feedback card, explanation card, end emblem | ✅ |
| 36 | `/match/:id/result` | `ResultScreen` (match/result_screen.dart) — victory/defeat/draw, Play Again, room return — hero result card, awards card, round cards, team lines | ✅ |

### Dialogs, sheets and overlays

| # | Surface | File | Status |
| --- | --- | --- | --- |
| D1 | Confirm dialog, snack bars | core/widgets/dialogs.dart | ✅ |
| D2 | Share result card + dialog | core/share/result_card.dart | ✅ |
| D3 | Topic picker sheet, topic chips | match/topic_picker.dart | ✅ |
| D4 | Room settings dialog (type, format, team mode, topic) | match/format_picker.dart | ✅ |
| D5 | Party picker dialog | match/mode_picker_screen.dart | ✅ |
| D6 | Quick chat phrase sheet | match/match_screen.dart | ✅ |
| D7 | Play Again panel | match/play_again.dart | ✅ |
| D8 | Room return banner | lobby/room_return.dart | ✅ |
| D9 | Room create/join sheet | lobby/room_screen.dart (`_RoomSheet`) | ✅ |
| D10 | Challenge type dialog | lobby/challenge_dialog.dart | ✅ |
| D11 | Incoming challenge overlay | lobby/challenge_listener.dart | ✅ |
| D12 | Add friend dialog | social/friends_screen.dart | ✅ |
| D13 | Create team dialog, member picker | team/teams_screen.dart | ✅ |
| D14 | Tournament lineup sheet | tournament/tournament_screens.dart | ✅ |
| D15 | Report dialog | moderation/report_dialog.dart | ✅ |
| D16 | Appeal dialog | moderation/sanctions_screen.dart | ✅ |
| D17 | Delete quiz confirm | creator/quizzes_screen.dart | ✅ |
| D18 | Chat message actions / confirm | chat/chat_screen.dart | ✅ |
| D19 | Account flows (link, sign out, delete) | auth/account_flows.dart | ✅ |

### Shared building blocks

| # | Component | File | Status |
| --- | --- | --- | --- |
| C1 | Theme tokens and `ThemeData` | app/theme.dart | ✅ |
| C2 | Page background | core/widgets/wc_background.dart | ✅ |
| C3 | Component kit + brand mark | core/widgets/bv.dart | ✅ |
| C4 | Progress widgets (XP bar, level, streak, quest rows) | progress/progress_widgets.dart | ✅ |
| C6 | Bottom bar (Home, Tournaments, Leaderboards, Profile) | home/main_nav.dart | ✅ |
| C5 | Error / loading text | auth/error_text.dart, match/play_error_text.dart | ✅ |

Profile: the mockup's "favourite categories" chips are not shown — the
API has no per-category statistics for a player yet (needs a backend
endpoint first).

Not present in the app (nothing to update): paywalls, subscriptions,
purchases, help center, onboarding carousel.

## Final checklist (2026-10-24)

How each surface was verified:

- **S** — screenshot test (`test/goldens/screens_golden_test.dart`,
  iPhone size, Turkish, dark), reviewed by eye. Goldens are dark only now.
- **L** — layout test (`test/layout/small_screen_test.dart`): the screen at
  360 × 740 dp with 130 % text must lay out without overflow. Runs with the
  normal test suite.
- **T** — widget tests that drive the screen's behaviour (unchanged
  business logic, navigation, keys).
- Every screen: no hard-coded legacy colours (only the medal colours on
  the podium and the Marathon tile tint are literal), background from
  `WcBackground`, text in Nunito from the theme, loading / empty / error
  states through `LoadingView` / `EmptyState` / `ErrorView`.

| Route / surface | Updated | S | L | T | Notes |
| --- | --- | --- | --- | --- | --- |
| Splash (`/splash`) | ✅ | ✅ (error state) | — | ✅ | loading state shows only for a moment |
| Welcome / age gate (`/welcome`) | ✅ | ✅ | — | ✅ | |
| Account status (`/account-status`) | ✅ | ❌ | — | ✅ (cancel deletion) | not reachable in the screenshot fakes; check on a device |
| Home (`/`) + bottom bar | ✅ | ✅ | ✅ | ✅ | |
| Settings | ✅ | ✅ | ✅ | ✅ | |
| Profile | ✅ | ✅ | ✅ | ✅ | favourite categories not shown (no API data) |
| Account | ✅ | ✅ | ✅ | ✅ | |
| Email sign-up / sign-in | ✅ | ✅ (sign-up) | ✅ | ✅ | sign-in is the same screen |
| Verify email | ✅ | ✅ | ✅ | ✅ | |
| Credits | ✅ | ✅ | ✅ | ✅ | "BrainVolt" in the list is server data (content source record) |
| Friends (+ requests tab) | ✅ | ✅ | ✅ | ✅ | |
| Challenges | ✅ | ✅ (empty) | ✅ | ✅ (via social flow) | list rows only in widget tests |
| Mode picker (`/modes`) | ✅ | ✅ | ✅ | ✅ | |
| Teams | ✅ | ✅ (guest view) | ✅ | ✅ | team list rows only in widget tests |
| Team detail (`/teams/:id`) | ✅ | ❌ | ❌ | ✅ (partly) | check on a device |
| Room + matchmaking lobby (`/rooms/:id`) | ✅ | ✅ (one-off, not a golden) | ❌ | ✅ | |
| Leaderboards | ✅ | ✅ | ✅ | ✅ | |
| Tournaments | ✅ | ✅ (empty) | ✅ | ✅ | |
| Create tournament | ✅ | ✅ | ✅ | ✅ | |
| Tournament detail | ✅ | ❌ | ❌ | ✅ | check on a device |
| Quests | ✅ | ✅ | ✅ | ✅ | |
| Badges | ✅ | ✅ | ✅ | ✅ | |
| Daily challenge | ✅ | ✅ | ✅ | ✅ | |
| Daily run (`/daily/runs/:id`) | ✅ | ❌ | ❌ | ✅ | check on a device |
| Sanctions | ✅ | ✅ (empty) | ✅ | ✅ | |
| Quizzes (mine / discover) | ✅ | ✅ (empty) | ✅ | ✅ | |
| Quiz editor (new / edit) | ✅ | ✅ (new) | ✅ | ✅ | |
| Inbox | ✅ | ✅ (empty) | ✅ | ✅ | |
| Chat (team / couple) | ✅ | ❌ | ❌ | ✅ | check on a device |
| Notification settings | ✅ | ✅ | ✅ | ✅ | |
| Queue (`/play`) | ✅ | ✅ | ✅ | ✅ | |
| Match (question, reveal) | ✅ | ✅ | ❌ | ✅ | needs a live socket; team match and Mixed Battle pick only in widget tests |
| Match end, Play Again, room return | ✅ | ❌ | ❌ | ✅ | |
| Result (`/match/:id/result`) | ✅ | ✅ | ❌ | ✅ | |
| Topic sheet, room sheet, room settings dialog | ✅ | ✅ (room sheet) | ✅ | ✅ | |
| Share card dialog | ✅ | ✅ | — | ✅ | |
| Other dialogs (confirm, add friend, create team, pick friend, party, challenge type, report, appeal, lineup, quick chat) | ✅ | ❌ | ❌ | ✅ (except lineup) | restyled through the dialog / sheet / input themes |

Not updated: none. Not verified by screenshot (listed above as ❌):
account status, team detail, tournament detail, daily run, chat, match end
and the smaller dialogs — they use the same kit and themes and pass their
widget tests, but nobody has looked at them yet; the owner checks them on
a device. iOS was not built (no Mac available).

Layout fixes found by the 360 dp / 130 % test: Home's mode and shortcut
rows now grow with the text size, the queue screen scrolls, the profile
badge strip is taller and keeps badge names to one line. The match timer
ring moves in whole seconds (stable screenshots, in step with the number).

## Later changes (owner feedback, 2026-10-24)

- Name back to **BrainVolt**. Logo: the owner's brain + bolt artwork.
  `BrandMark` shows `assets/brand/brainvolt_mark.png` (the brain crop);
  `assets/brand/brainvolt_icon_1024.png` is the store icon. Generated from
  it: Android launcher icons (legacy + adaptive: brain on #100B2E), Android
  splash (pre-12 `launch_background`, 12+ `windowSplashScreen*`), iOS app
  icons and launch screen (navy background).
- "Quizler" → **Soru Atölyesi** / Question Lab (players suggest questions
  and build quizzes there).
- Topic sheet: the categories load in the background on Home and stay for
  the session (per language), so the sheet opens with them; each topic is
  a tile with its own icon and colour (`topicIcon` / `topicColor` by the
  server's `icon_key`). Golden: `topic_sheet_dark.png`.
- Queue: a tip card under the radar changes every 4 s (11 tips about
  modes, scoring, leagues, daily challenge, quests, friends, teams).
