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
import '../features/content/presentation/credits_screen.dart';
import '../features/home/presentation/home_screen.dart';
import '../features/lobby/presentation/challenges_screen.dart';
import '../features/match/presentation/match_screen.dart';
import '../features/match/presentation/queue_screen.dart';
import '../features/match/presentation/result_screen.dart';
import '../features/profile/presentation/profile_screen.dart';
import '../features/profile/presentation/settings_screen.dart';
import '../features/social/presentation/friends_screen.dart';
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
  static const challenges = '/challenges';
  static const play = '/play';
  static const practice = '/play?practice=1';
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
        path: Routes.credits,
        builder: (context, state) => const CreditsScreen(),
      ),
      GoRoute(
        path: Routes.play,
        builder: (context, state) =>
            QueueScreen(practice: state.uri.queryParameters['practice'] == '1'),
      ),
      GoRoute(
        path: '/match/:id',
        builder: (context, state) =>
            MatchScreen(matchId: state.pathParameters['id']!),
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
