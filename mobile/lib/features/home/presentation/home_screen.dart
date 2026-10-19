import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/config.dart';
import '../../../app/providers.dart';
import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../auth/domain/session.dart';
import '../../lobby/presentation/room_screen.dart';
import '../../match/data/match_providers.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';

/// The caller's live match, if any (back from a crash or another screen).
final currentMatchProvider = FutureProvider.autoDispose<String?>(
  (ref) => ref.watch(matchRepositoryProvider).currentMatch(),
);

/// Challenge events for the caller (websocket.md §5.2): home shows them in
/// a snack bar wherever the player is in the app's main screen.
final lobbyEventsProvider = StreamProvider.autoDispose<ServerFrame>((ref) {
  final rt = ref.watch(realtimeClientProvider)..start();
  return rt.frames.where(
    (f) => f.type == 'challenge.received' || f.type == 'challenge.updated',
  );
});

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
    ref.listen(lobbyEventsProvider, (_, next) {
      final f = next.value;
      if (f == null) {
        return;
      }
      final messenger = ScaffoldMessenger.of(context);
      if (f.type == 'challenge.received') {
        final c = (f.data['challenge'] as Map?)?.cast<String, dynamic>();
        final from = (c?['challenger'] as Map?)?['display_name'] as String?;
        messenger.showSnackBar(
          SnackBar(
            content: Text(l10n.challengeReceived(from ?? '')),
            action: SnackBarAction(
              label: l10n.view,
              onPressed: () => context.push(Routes.challenges),
            ),
          ),
        );
      } else if (f.data['status'] == 'started' &&
          f.data['match_id'] is String) {
        final id = f.data['match_id'] as String;
        messenger.showSnackBar(
          SnackBar(
            content: Text(l10n.challengeAccepted),
            action: SnackBarAction(
              label: l10n.playNow,
              onPressed: () => context.go(Routes.match(id)),
            ),
          ),
        );
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
                onTap: () => context.push(Routes.play),
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
            gap,
            if (session != null) ...[const StatsRow(), gap],
            _Tiles(
              height: 86,
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
          ],
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({this.avatar});

  final String? avatar;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
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
        round(
          const Key('open-challenges'),
          l10n.homeChallenges,
          Icons.sports_kabaddi,
          Routes.challenges,
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
      padding: const EdgeInsets.fromLTRB(20, 18, 12, 18),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (name != null)
                  Text(l10n.homeGreeting(name!), style: text.headlineSmall),
                const SizedBox(height: 4),
                Text(
                  l10n.homeWelcome,
                  style: text.bodyMedium?.copyWith(color: bv.muted),
                ),
                const SizedBox(height: 14),
                if (name != null) const LevelSummary(),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Stack(
            alignment: Alignment.topCenter,
            children: [
              Padding(
                padding: const EdgeInsets.only(top: 22),
                child: AvatarBubble(
                  avatarFor(avatar ?? 'penguin'),
                  size: 96,
                  ring: false,
                ),
              ),
              const Text('👑', style: TextStyle(fontSize: 34)),
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
