import 'package:flutter/foundation.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../features/auth/domain/session.dart';
import '../features/auth/presentation/account_screen.dart';
import '../features/auth/presentation/account_status_screen.dart';
import '../features/auth/presentation/age_gate_screen.dart';
import '../features/auth/presentation/email_auth_screen.dart';
import '../features/auth/presentation/splash_screen.dart';
import '../features/auth/presentation/verify_email_screen.dart';
import '../features/chat/presentation/chat_screen.dart';
import '../features/content/presentation/credits_screen.dart';
import '../features/creator/presentation/quiz_editor_screen.dart';
import '../features/creator/presentation/quizzes_screen.dart';
import '../features/daily/presentation/daily_answers_screen.dart';
import '../features/daily/presentation/daily_run_screen.dart';
import '../features/daily/presentation/daily_screen.dart';
import '../features/home/presentation/home_screen.dart';
import '../features/lobby/presentation/challenges_screen.dart';
import '../features/lobby/presentation/room_screen.dart';
import '../features/match/domain/match_repository.dart';
import '../features/match/presentation/match_screen.dart';
import '../features/match/presentation/mode_picker_screen.dart';
import '../features/match/presentation/queue_screen.dart';
import '../features/match/presentation/result_screen.dart';
import '../features/moderation/presentation/sanctions_screen.dart';
import '../features/notifications/presentation/inbox_screen.dart';
import '../features/notifications/presentation/notification_settings_screen.dart';
import '../features/pass/presentation/season_pass_screen.dart';
import '../features/profile/presentation/guide_screen.dart';
import '../features/profile/presentation/profile_screen.dart';
import '../features/profile/presentation/settings_screen.dart';
import '../features/progress/presentation/badges_screen.dart';
import '../features/progress/presentation/leaderboards_screen.dart';
import '../features/progress/presentation/quests_screen.dart';
import '../features/social/presentation/friends_screen.dart';
import '../features/social/presentation/player_profile.dart';
import '../features/store/presentation/plus_screen.dart';
import '../features/store/presentation/volt_screen.dart';
import '../features/team/presentation/teams_screen.dart';
import '../features/tournament/presentation/tournament_screens.dart';
import 'session_controller.dart';

part 'router.g.dart';

/// Route paths.
abstract final class Routes {
  static const splash = '/splash';
  static const ageGate = '/welcome';
  static const accountStatus = '/account-status';
  static const home = '/';
  static const settings = '/settings';
  static const profile = '/profile';
  static const account = '/account';
  static const emailSignUp = '/account/email-sign-up';
  static const emailSignIn = '/account/email-sign-in';
  static const verifyEmail = '/account/verify-email';
  static const credits = '/credits';
  static const friends = '/friends';
  static String player(String id) => '/players/$id';
  static const challenges = '/challenges';
  static const modes = '/modes';
  static const teams = '/teams';
  static const leaderboards = '/leaderboards';
  static const tournaments = '/tournaments';
  static const tournamentNew = '/tournaments/new';
  static String tournament(String id) => '/tournaments/$id';
  static const quests = '/quests';
  static const badges = '/badges';
  static const daily = '/daily';
  static const dailyAnswers = '/daily/answers';
  static const sanctions = '/sanctions';
  static const quizzes = '/quizzes';
  static const inbox = '/inbox';
  static String teamChat(String id, String title) =>
      Uri(path: '/chat/team/$id', queryParameters: {'title': title}).toString();
  static String coupleChat(String id, String title) => Uri(
    path: '/chat/couple/$id',
    queryParameters: {'title': title},
  ).toString();
  static const notificationSettings = '/settings/notifications';
  static const guide = '/settings/guide';
  static const plus = '/plus';
  static const store = '/store';
  static const seasonPass = '/season-pass';
  static const quizNew = '/quizzes/new';
  static String quiz(String id) => '/quizzes/$id';
  static String dailyRun(String id) => '/daily/runs/$id';
  static String room(String id) => '/rooms/$id';
  static String team(String id) => '/teams/$id';
  static const play = '/play';
  static const practice = '/play?practice=1';
  static String playMode(QueueMode m) => Uri(
    path: play,
    queryParameters: {
      'type': m.matchType,
      'mode': ?m.categoryMode,
      'cat': ?m.categoryId,
      if (m.format != '1v1') 'format': m.format,
      if (m.partyUserIds.isNotEmpty) 'party': m.partyUserIds.join(','),
      'team': ?m.teamId,
    },
  ).toString();
  static String match(String id) => '/match/$id';
  static String matchResult(String id) => '/match/$id/result';
}

/// The sign-in guard (roadmap 1.8.3): where [location] must go for the
/// current session state, or null to stay.
@visibleForTesting
String? sessionRedirect(AsyncValue<AppSession> session, String location) {
  String? goTo(String target) => location == target ? null : target;
  if (session.hasError || !session.hasValue) {
    return goTo(Routes.splash); // starting, or failed with a retry button
  }
  switch (session.requireValue) {
    case AgeGateRequired():
      return goTo(Routes.ageGate);
    case SignedIn(:final session):
      if (session.status == AccountStatus.suspended ||
          session.flags.deletionPending) {
        return goTo(Routes.accountStatus);
      }
      const gates = {Routes.splash, Routes.ageGate, Routes.accountStatus};
      return gates.contains(location) ? Routes.home : null;
  }
}

/// The app's router (go_router, ADR-0003), re-evaluated on every session
/// change.
@Riverpod(keepAlive: true)
GoRouter router(Ref ref) {
  final session = ValueNotifier(ref.read(sessionControllerProvider));
  ref.listen(sessionControllerProvider, (_, next) => session.value = next);
  final router = GoRouter(
    initialLocation: Routes.splash,
    refreshListenable: session,
    redirect: (context, state) =>
        sessionRedirect(session.value, state.matchedLocation),
    routes: [
      GoRoute(
        path: Routes.splash,
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: Routes.ageGate,
        builder: (context, state) => const AgeGateScreen(),
      ),
      GoRoute(
        path: Routes.accountStatus,
        builder: (context, state) => const AccountStatusScreen(),
      ),
      GoRoute(
        path: Routes.home,
        builder: (context, state) => const HomeScreen(),
      ),
      GoRoute(
        path: Routes.sanctions,
        builder: (context, state) => const SanctionsScreen(),
      ),
      GoRoute(
        path: Routes.daily,
        builder: (context, state) => const DailyScreen(),
      ),
      GoRoute(
        path: Routes.dailyAnswers,
        builder: (context, state) => const DailyAnswersScreen(),
      ),
      GoRoute(
        path: '/daily/runs/:id',
        builder: (context, state) =>
            DailyRunScreen(runId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/players/:id',
        builder: (context, state) =>
            PlayerProfileScreen(userId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: Routes.plus,
        builder: (context, state) => const PlusScreen(),
      ),
      GoRoute(
        path: Routes.seasonPass,
        builder: (context, state) => const SeasonPassScreen(),
      ),
      GoRoute(
        path: Routes.store,
        builder: (context, state) => VoltScreen(
          cosmetics: state.uri.queryParameters['tab'] == 'cosmetics',
        ),
      ),
      GoRoute(
        path: Routes.guide,
        builder: (context, state) => const GuideScreen(),
      ),
      GoRoute(
        path: Routes.leaderboards,
        builder: (context, state) => const LeaderboardsScreen(),
      ),
      GoRoute(
        path: Routes.quests,
        builder: (context, state) => const QuestsScreen(),
      ),
      GoRoute(
        path: Routes.badges,
        builder: (context, state) => const BadgesScreen(),
      ),
      GoRoute(
        path: Routes.settings,
        builder: (context, state) => const SettingsScreen(),
      ),
      GoRoute(
        path: Routes.profile,
        builder: (context, state) => const ProfileScreen(),
      ),
      GoRoute(
        path: Routes.account,
        builder: (context, state) => const AccountScreen(),
      ),
      GoRoute(
        path: Routes.emailSignUp,
        builder: (context, state) => const EmailAuthScreen(signUp: true),
      ),
      GoRoute(
        path: Routes.emailSignIn,
        builder: (context, state) => const EmailAuthScreen(signUp: false),
      ),
      GoRoute(
        path: Routes.verifyEmail,
        builder: (context, state) => const VerifyEmailScreen(),
      ),
      GoRoute(
        path: Routes.friends,
        builder: (context, state) => const FriendsScreen(),
      ),
      GoRoute(
        path: Routes.challenges,
        builder: (context, state) => const ChallengesScreen(),
      ),
      GoRoute(
        path: Routes.modes,
        builder: (context, state) => const ModePickerScreen(),
      ),
      GoRoute(
        path: Routes.teams,
        builder: (context, state) => const TeamsScreen(),
      ),
      GoRoute(
        path: '/teams/:id',
        builder: (context, state) =>
            TeamScreen(teamId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/rooms/:id',
        builder: (context, state) =>
            RoomScreen(roomId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: Routes.tournaments,
        builder: (context, state) => const TournamentsScreen(),
      ),
      GoRoute(
        path: Routes.tournamentNew,
        builder: (context, state) => const CreateTournamentScreen(),
      ),
      GoRoute(
        path: '/tournaments/:id',
        builder: (context, state) =>
            TournamentScreen(id: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/chat/team/:id',
        builder: (context, state) => ChatScreen(
          teamId: state.pathParameters['id'],
          title: state.uri.queryParameters['title'],
        ),
      ),
      GoRoute(
        path: '/chat/couple/:id',
        builder: (context, state) => ChatScreen(
          coupleId: state.pathParameters['id'],
          title: state.uri.queryParameters['title'],
        ),
      ),
      GoRoute(
        path: Routes.inbox,
        builder: (context, state) => const InboxScreen(),
      ),
      GoRoute(
        path: Routes.notificationSettings,
        builder: (context, state) => const NotificationSettingsScreen(),
      ),
      GoRoute(
        path: Routes.quizzes,
        builder: (context, state) => const QuizzesScreen(),
      ),
      GoRoute(
        path: Routes.quizNew,
        builder: (context, state) => const QuizEditorScreen(),
      ),
      GoRoute(
        path: '/quizzes/:id',
        builder: (context, state) =>
            QuizEditorScreen(quizId: state.pathParameters['id']),
      ),
      GoRoute(
        path: Routes.credits,
        builder: (context, state) => const CreditsScreen(),
      ),
      GoRoute(
        path: Routes.play,
        builder: (context, state) {
          final q = state.uri.queryParameters;
          return QueueScreen(
            practice: q['practice'] == '1',
            mode: QueueMode(
              matchType: q['type'] ?? 'quick_duel',
              categoryMode: q['mode'],
              categoryId: q['cat'],
              format: q['format'] ?? '1v1',
              partyUserIds: [
                for (final id in (q['party'] ?? '').split(','))
                  if (id.isNotEmpty) id,
              ],
              teamId: q['team'],
            ),
          );
        },
      ),
      GoRoute(
        path: '/match/:id',
        // Keyed by id: going from one match straight to the next (play again)
        // must start a fresh screen, not keep the old one's ready state.
        builder: (context, state) => MatchScreen(
          key: ValueKey(state.pathParameters['id']),
          matchId: state.pathParameters['id']!,
        ),
        routes: [
          GoRoute(
            path: 'result',
            builder: (context, state) =>
                ResultScreen(matchId: state.pathParameters['id']!),
          ),
        ],
      ),
    ],
  );
  ref.onDispose(() {
    router.dispose();
    session.dispose();
  });
  return router;
}
