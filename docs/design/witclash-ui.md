# WitClash UI modernization (D-099)

The app moves from the BrainVolt look (D-085, light/dark) to the **WitClash**
design system: one dark theme, a deep navy/indigo star field with a
purple → orange glow, frosted glass cards, purple → orange gradient
buttons, and the "WitClash" word mark. Platform identifiers
(`com.brainvolt.app`, the Dart package `brainvolt`, Firebase project, API
hosts) do not change; only the user-visible name does.

## Approach

Everything flows from shared pieces, so no screen gets a one-off patch:

| Piece | Where | What it does |
| --- | --- | --- |
| Tokens | `lib/app/theme.dart` (`BvColors.witclash`) | background, glass fill/border, ink, muted, badge, gradient (purple → magenta → orange), accent, gold, success, danger, glow, surface for dialogs |
| Theme | `buildTheme()` | single dark `ThemeData`: transparent scaffolds/app bars, glass inputs, gradient-friendly buttons, dialogs/sheets/snack bars/menus/chips/tabs/segmented buttons/switches/nav bar in WitClash colours |
| Background | `WcBackground` (`lib/core/widgets/wc_background.dart`) | gradient + star field painter; wrapped around every routed page by the theme's page transition builder, so every route (and any future one) gets it |
| Kit | `lib/core/widgets/bv.dart` | `BvCard` (glass), `GradientButton`, `IconBadge`, `ActionCard`, `ChoiceTile`, `OptionTile`, `AvatarBubble`, `EmptyState`, `BrandLogo` (WitClash word mark + W/bolt mark) |
| Strings | `app_tr.arb`, `app_en.arb` | "BrainVolt" → "WitClash" in user-visible text |
| Platform | `AndroidManifest.xml` `android:label`, `Info.plist` `CFBundleDisplayName` | launcher name "WitClash" (ids unchanged) |

The light/dark choice in Settings is removed: WitClash is dark only. The
stored `theme_mode` preference is ignored.

## Screen inventory

Taken from `lib/app/router.dart` and every `Scaffold`, dialog and bottom
sheet in `lib/`. Status: ⬜ not started, 🟨 tokens/kit only (inherits the
shared theme, layout not yet reviewed), ✅ updated and verified.

### Routes

| # | Route | Screen (file) | Status |
| --- | --- | --- | --- |
| 1 | `/splash` | `SplashScreen` (auth/splash_screen.dart) | 🟨 |
| 2 | `/welcome` | `AgeGateScreen` (auth/age_gate_screen.dart) | 🟨 |
| 3 | `/account-status` | `AccountStatusScreen` (auth/account_status_screen.dart) | 🟨 |
| 4 | `/` | `HomeScreen` (home/home_screen.dart) | 🟨 |
| 5 | `/settings` | `SettingsScreen` (profile/settings_screen.dart) | 🟨 |
| 6 | `/profile` | `ProfileScreen` (profile/profile_screen.dart) | 🟨 |
| 7 | `/account` | `AccountScreen` (auth/account_screen.dart) | 🟨 |
| 8 | `/account/email-sign-up` | `EmailAuthScreen(signUp: true)` | 🟨 |
| 9 | `/account/email-sign-in` | `EmailAuthScreen(signUp: false)` | 🟨 |
| 10 | `/account/verify-email` | `VerifyEmailScreen` | 🟨 |
| 11 | `/credits` | `CreditsScreen` (content/credits_screen.dart) | 🟨 |
| 12 | `/friends` | `FriendsScreen` (social/friends_screen.dart) | 🟨 |
| 13 | `/challenges` | `ChallengesScreen` (lobby/challenges_screen.dart) | 🟨 |
| 14 | `/modes` | `ModePickerScreen` (match/mode_picker_screen.dart) | 🟨 |
| 15 | `/teams` | `TeamsScreen` (team/teams_screen.dart) | 🟨 |
| 16 | `/teams/:id` | `TeamScreen` (team/teams_screen.dart) | 🟨 |
| 17 | `/rooms/:id` | `RoomScreen` (lobby/room_screen.dart) — private room and matchmaking lobby bodies | 🟨 |
| 18 | `/leaderboards` | `LeaderboardsScreen` (progress/leaderboards_screen.dart) | 🟨 |
| 19 | `/tournaments` | `TournamentsScreen` (tournament/tournament_screens.dart) | 🟨 |
| 20 | `/tournaments/new` | `CreateTournamentScreen` | 🟨 |
| 21 | `/tournaments/:id` | `TournamentScreen` | 🟨 |
| 22 | `/quests` | `QuestsScreen` (progress/quests_screen.dart) | 🟨 |
| 23 | `/badges` | `BadgesScreen` (progress/badges_screen.dart) | 🟨 |
| 24 | `/daily` | `DailyScreen` (daily/daily_screen.dart) | 🟨 |
| 25 | `/daily/runs/:id` | `DailyRunScreen` (daily/daily_run_screen.dart) | 🟨 |
| 26 | `/sanctions` | `SanctionsScreen` (moderation/sanctions_screen.dart) | 🟨 |
| 27 | `/quizzes` | `QuizzesScreen` (creator/quizzes_screen.dart) | 🟨 |
| 28 | `/quizzes/new` | `QuizEditorScreen` (creator/quiz_editor_screen.dart) | 🟨 |
| 29 | `/quizzes/:id` | `QuizEditorScreen(id)` | 🟨 |
| 30 | `/inbox` | `InboxScreen` (notifications/inbox_screen.dart) | 🟨 |
| 31 | `/chat/team/:id` | `ChatScreen` (chat/chat_screen.dart) | 🟨 |
| 32 | `/chat/couple/:id` | `ChatScreen` | 🟨 |
| 33 | `/settings/notifications` | `NotificationSettingsScreen` | 🟨 |
| 34 | `/play` | `QueueScreen` (match/queue_screen.dart) — queue and practice | 🟨 |
| 35 | `/match/:id` | `MatchScreen` (match/match_screen.dart) — question, options, countdown, reveal, round transitions, team proposals | 🟨 |
| 36 | `/match/:id/result` | `ResultScreen` (match/result_screen.dart) — victory/defeat/draw, Play Again, room return | 🟨 |

### Dialogs, sheets and overlays

| # | Surface | File | Status |
| --- | --- | --- | --- |
| D1 | Confirm dialog, snack bars | core/widgets/dialogs.dart | 🟨 |
| D2 | Share result card + dialog | core/share/result_card.dart | 🟨 |
| D3 | Topic picker sheet, topic chips | match/topic_picker.dart | 🟨 |
| D4 | Room settings dialog (type, format, team mode, topic) | match/format_picker.dart | 🟨 |
| D5 | Party picker dialog | match/mode_picker_screen.dart | 🟨 |
| D6 | Quick chat phrase sheet | match/match_screen.dart | 🟨 |
| D7 | Play Again panel | match/play_again.dart | 🟨 |
| D8 | Room return banner | lobby/room_return.dart | 🟨 |
| D9 | Room create/join sheet | lobby/room_screen.dart (`_RoomSheet`) | 🟨 |
| D10 | Challenge type dialog | lobby/challenge_dialog.dart | 🟨 |
| D11 | Incoming challenge overlay | lobby/challenge_listener.dart | 🟨 |
| D12 | Add friend dialog | social/friends_screen.dart | 🟨 |
| D13 | Create team dialog, member picker | team/teams_screen.dart | 🟨 |
| D14 | Tournament lineup sheet | tournament/tournament_screens.dart | 🟨 |
| D15 | Report dialog | moderation/report_dialog.dart | 🟨 |
| D16 | Appeal dialog | moderation/sanctions_screen.dart | 🟨 |
| D17 | Delete quiz confirm | creator/quizzes_screen.dart | 🟨 |
| D18 | Chat message actions / confirm | chat/chat_screen.dart | 🟨 |
| D19 | Account flows (link, sign out, delete) | auth/account_flows.dart | 🟨 |

### Shared building blocks

| # | Component | File | Status |
| --- | --- | --- | --- |
| C1 | Theme tokens and `ThemeData` | app/theme.dart | ✅ |
| C2 | Page background | core/widgets/wc_background.dart | ✅ |
| C3 | Component kit + brand mark | core/widgets/bv.dart | ✅ |
| C4 | Progress widgets (XP bar, level, streak, quest rows) | progress/progress_widgets.dart | 🟨 |
| C5 | Error / loading text | auth/error_text.dart, match/play_error_text.dart | 🟨 |

Not present in the app (nothing to update): paywalls, subscriptions,
purchases, help center, onboarding carousel.

## Verification per screen

Each screen is checked for: background, branding/typography, cards and
controls, token use (no hard-coded legacy colours), overflow at 360 dp
width and large text, loading/empty/error states, unchanged behaviour
(widget tests). The final checklist is added at the end of this file.
