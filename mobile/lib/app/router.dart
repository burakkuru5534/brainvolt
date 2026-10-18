import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../features/home/presentation/home_screen.dart';
import '../features/profile/presentation/settings_screen.dart';

part 'router.g.dart';

/// Route paths.
abstract final class Routes {
  static const home = '/';
  static const settings = '/settings';
}

/// The app's router (go_router, ADR-0003). The sign-in redirect guard is
/// added with Firebase Auth (roadmap 1.8).
@Riverpod(keepAlive: true)
GoRouter router(Ref ref) {
  final router = GoRouter(
    routes: [
      GoRoute(
        path: Routes.home,
        builder: (context, state) => const HomeScreen(),
      ),
      GoRoute(
        path: Routes.settings,
        builder: (context, state) => const SettingsScreen(),
      ),
    ],
  );
  ref.onDispose(router.dispose);
  return router;
}
