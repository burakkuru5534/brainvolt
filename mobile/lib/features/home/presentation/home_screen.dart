import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/config.dart';
import '../../../app/providers.dart';
import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/push/push_service.dart';
import '../../../core/realtime/realtime_lifecycle.dart';
import '../../../core/widgets/bv.dart';
import '../../auth/data/auth_providers.dart';
import '../../auth/domain/session.dart';
import '../../lobby/presentation/room_screen.dart';
import '../../match/data/match_providers.dart';
import '../../match/domain/match_repository.dart';
import '../../match/presentation/topic_picker.dart';
import '../../notifications/presentation/inbox_screen.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';

/// The caller's live match, if any (back from a crash or another screen).
final currentMatchProvider = FutureProvider.autoDispose<String?>(
  (ref) => ref.watch(matchRepositoryProvider).currentMatch(),
);

/// Routes of notifications the user tapped, starting with the one that
/// launched the app.
final pushOpenedProvider = StreamProvider.autoDispose<String>((ref) async* {
  final push = ref.watch(pushServiceProvider);
  if (await push.initialRoute() case final route?) {
    yield route;
  }
  yield* push.opened;
});

/// Rotated FCM tokens, sent to the server.
final pushTokenRefreshProvider = StreamProvider.autoDispose<String>(
  (ref) => ref.watch(pushServiceProvider).tokenRefresh,
);

/// Home (D-085): logo bar, greeting hero with level and mascot, the Quick
/// Duel call to action, mode tiles, shortcuts and stats. Development
/// builds also offer a practice match against the server's bot (D-066).
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final session = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session,
      _ => null,
    };
    ref
      ..listen(inboxEventsProvider, (_, _) => ref.invalidate(inboxProvider))
      // No socket on Home (D-097): the bell catches up on return.
      ..listen(appForegroundProvider, (was, now) {
        if (now && was == false) {
          ref.invalidate(inboxProvider);
        }
      })
      ..listen(pushOpenedProvider, (_, next) {
        if (next.value case final route?) {
          openRoute(context, route);
        }
      })
      ..listen(pushTokenRefreshProvider, (_, next) {
        if (next.value case final token?) {
          ref
              .read(sessionRepositoryProvider)
              .registerDevice(fcmToken: token)
              .catchError((Object _) {})
              .ignore();
        }
      });
    final resume = ref.watch(currentMatchProvider).value;
    final dev = ref.watch(appConfigProvider).flavor == Flavor.dev;
    const gap = SizedBox(height: 14);
    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          children: [
            _TopBar(avatar: session?.profile.avatarId),
            if (session?.tier == Tier.guest)
              _Banner(
                key: const Key('guest-banner'),
                text: l10n.homeGuestBanner,
                action: l10n.linkAccountAction,
                onPressed: () => context.push(Routes.account),
              )
            else if (session?.flags.emailVerificationRequired ?? false)
              _Banner(
                key: const Key('verify-banner'),
                text: l10n.homeVerifyBanner,
                action: l10n.verifyAction,
                onPressed: () => context.push(Routes.verifyEmail),
              ),
            const SizedBox(height: 12),
            _Hero(
              name: session?.profile.displayName,
              avatar: session?.profile.avatarId,
            ),
            const SizedBox(height: 10),
            _Tiles(
              height: 96,
              tiles: [
                _Tile(
                  const Key('open-leaderboards'),
                  Icons.leaderboard,
                  l10n.leaderboardsTitle,
                  null,
                  () => context.push(Routes.leaderboards),
                ),
                _Tile(
                  const Key('open-quests'),
                  Icons.flag,
                  l10n.questsTitle,
                  null,
                  () => context.push(Routes.quests),
                ),
                _Tile(
                  const Key('open-badges'),
                  Icons.workspace_premium,
                  l10n.badgesTitle,
                  null,
                  () => context.push(Routes.badges),
                ),
                _Tile(
                  const Key('open-friends'),
                  Icons.people,
                  l10n.homeFriends,
                  null,
                  () => context.push(Routes.friends),
                ),
              ],
            ),
            gap,
            if (resume != null)
              _QuickDuel(
                key: const Key('resume-match'),
                title: l10n.resumeMatch,
                subtitle: l10n.homeQuickDuelSub,
                icon: Icons.play_arrow,
                onTap: () => context.go(Routes.match(resume)),
              )
            else
              _QuickDuel(
                key: const Key('play-quick-duel'),
                title: l10n.playQuickDuel,
                subtitle: l10n.homeQuickDuelSub,
                icon: Icons.bolt,
                onTap: () => _quickDuel(context),
              ),
            gap,
            _Tiles(
              tiles: [
                _Tile(
                  const Key('play-modes'),
                  Icons.grid_view_rounded,
                  l10n.playModes,
                  l10n.homeModesSub,
                  () => context.push(Routes.modes),
                ),
                _Tile(
                  const Key('private-room'),
                  Icons.lock_outline,
                  l10n.privateRoom,
                  l10n.homeRoomSub,
                  () => showRoomSheet(context, ref),
                ),
                _Tile(
                  const Key('open-teams'),
                  Icons.groups_outlined,
                  l10n.homeTeams,
                  l10n.homeTeamsSub,
                  () => context.push(Routes.teams),
                ),
              ],
            ),
            gap,
            _Tiles(
              tiles: [
                _Tile(
                  const Key('open-daily'),
                  Icons.today,
                  l10n.dailyTitle,
                  l10n.homeDailySub,
                  () => context.push(Routes.daily),
                ),
                _Tile(
                  const Key('open-quizzes'),
                  Icons.edit_note,
                  l10n.homeQuizzes,
                  l10n.homeQuizzesSub,
                  () => context.push(Routes.quizzes),
                ),
                _Tile(
                  const Key('open-tournaments'),
                  Icons.emoji_events_outlined,
                  l10n.tournamentsTitle,
                  l10n.homeTournamentsSub,
                  () => context.push(Routes.tournaments),
                ),
              ],
            ),
            if (dev) ...[
              gap,
              ActionCard(
                key: const Key('play-practice'),
                icon: Icons.smart_toy_outlined,
                title: l10n.playPractice,
                subtitle: l10n.homePracticeSub,
                onTap: () => context.push(Routes.practice),
              ),
            ],
            if (session != null) ...[gap, const StatsRow()],
          ],
        ),
      ),
    );
  }
}

/// Quick Duel asks for the topic first (D-095), then searches.
Future<void> _quickDuel(BuildContext context) async {
  final topic = await pickTopic(context);
  if (topic == null || !context.mounted) {
    return;
  }
  unawaited(
    context.push(
      topic.all
          ? Routes.play
          : Routes.playMode(
              QueueMode(
                categoryMode: 'category_master',
                categoryId: topic.categoryId,
              ),
            ),
    ),
  );
}

class _TopBar extends ConsumerWidget {
  const _TopBar({this.avatar});

  final String? avatar;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final unread = ref.watch(inboxProvider).value?.unread ?? 0;
    Widget round(Key key, String tip, IconData icon, String route) => Padding(
      padding: const EdgeInsets.only(left: 8),
      child: BvCard(
        key: key,
        radius: 28,
        padding: const EdgeInsets.all(10),
        onTap: () => context.push(route),
        child: Tooltip(
          message: tip,
          child: Icon(icon, color: BvColors.of(context).badgeInk),
        ),
      ),
    );
    return Row(
      children: [
        const Expanded(
          child: Align(alignment: Alignment.centerLeft, child: BrandLogo()),
        ),
        Badge(
          key: const Key('inbox-badge'),
          isLabelVisible: unread > 0,
          label: Text('$unread'),
          offset: const Offset(-2, 2),
          child: round(
            const Key('open-inbox'),
            l10n.inboxTitle,
            Icons.notifications_outlined,
            Routes.inbox,
          ),
        ),
        round(
          const Key('open-settings'),
          l10n.settingsTitle,
          Icons.settings,
          Routes.settings,
        ),
        const SizedBox(width: 8),
        InkWell(
          key: const Key('open-profile'),
          customBorder: const CircleBorder(),
          onTap: () => context.push(Routes.profile),
          child: AvatarBubble(avatarFor(avatar ?? ''), size: 48, online: true),
        ),
      ],
    );
  }
}

/// The greeting card with the level line and the mascot (the player's
/// avatar until the mascot artwork arrives, D-085).
class _Hero extends StatelessWidget {
  const _Hero({this.name, this.avatar});

  final String? name;
  final String? avatar;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      padding: const EdgeInsets.fromLTRB(16, 10, 10, 10),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                if (name != null)
                  Text(
                    l10n.homeGreeting(name!),
                    style: text.titleLarge,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                const SizedBox(height: 2),
                Text(
                  l10n.homeWelcome,
                  style: text.bodySmall?.copyWith(color: bv.muted),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (name != null) ...[
                  const SizedBox(height: 8),
                  const LevelSummary(),
                ],
              ],
            ),
          ),
          const SizedBox(width: 8),
          Stack(
            alignment: Alignment.topCenter,
            children: [
              Padding(
                padding: const EdgeInsets.only(top: 14),
                child: AvatarBubble(
                  avatarFor(avatar ?? 'penguin'),
                  size: 60,
                  ring: false,
                ),
              ),
              const Text('👑', style: TextStyle(fontSize: 22)),
            ],
          ),
        ],
      ),
    );
  }
}

class _QuickDuel extends StatelessWidget {
  const _QuickDuel({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.onTap,
    super.key,
  });

  final String title;
  final String subtitle;
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return BvCard(
      gradient: bv.gradient,
      onTap: onTap,
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 20),
      child: Row(
        children: [
          IconBadge(icon, size: 64, onGradient: true),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title.toUpperCase(),
                  style: weighted(
                    const TextStyle(
                      fontSize: 24,
                      color: Colors.white,
                      fontFamily: brandFont,
                    ),
                    FontWeight.w900,
                  ),
                ),
                Text(
                  subtitle,
                  style: const TextStyle(color: Colors.white70, fontSize: 15),
                ),
              ],
            ),
          ),
          const Icon(Icons.chevron_right, color: Colors.white, size: 30),
        ],
      ),
    );
  }
}

class _Tile {
  const _Tile(this.key, this.icon, this.title, this.subtitle, this.onTap);

  final Key key;
  final IconData icon;
  final String title;
  final String? subtitle;
  final VoidCallback onTap;
}

class _Tiles extends StatelessWidget {
  const _Tiles({required this.tiles, this.height = 132});

  final List<_Tile> tiles;
  final double height;

  @override
  Widget build(BuildContext context) => SizedBox(
    height: height,
    child: Row(
      children: [
        for (final (i, t) in tiles.indexed) ...[
          if (i > 0) const SizedBox(width: 10),
          Expanded(
            child: ChoiceTile(
              key: t.key,
              icon: t.icon,
              label: t.title,
              subtitle: t.subtitle,
              selected: false,
              vertical: true,
              onTap: t.onTap,
            ),
          ),
        ],
      ],
    ),
  );
}

class _Banner extends StatelessWidget {
  const _Banner({
    required this.text,
    required this.action,
    required this.onPressed,
    super.key,
  });

  final String text;
  final String action;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 12),
    child: BvCard(
      padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
      child: Row(
        children: [
          Expanded(child: Text(text)),
          TextButton(onPressed: onPressed, child: Text(action)),
        ],
      ),
    ),
  );
}
