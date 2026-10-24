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
import '../../match/presentation/mode_picker_screen.dart';
import '../../match/presentation/topic_picker.dart';
import '../../notifications/presentation/inbox_screen.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';
import 'main_nav.dart';

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

/// Home (BrainVolt, D-099): logo bar, player card with level and XP, the
/// daily challenge, the game modes, shortcuts, stats and the bottom bar.
/// Development builds also offer a practice match against the server's bot
/// (D-066).
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
      // Loads the topics early, so the topic sheet opens with them.
      ..listen(categoriesProvider, (_, _) {})
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
    return Scaffold(
      bottomNavigationBar: const MainNavBar(current: MainTab.home),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
          children: [
            const _TopBar(),
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
            const SizedBox(height: 14),
            _PlayerCard(
              name: session?.profile.displayName,
              avatar: session?.profile.avatarId,
            ),
            const SizedBox(height: 14),
            _DailyCard(onTap: () => context.push(Routes.daily)),
            SectionTitle(
              l10n.homeModesTitle,
              action: l10n.homeSeeAll,
              onAction: () => context.push(Routes.modes),
            ),
            _ModeGrid(
              tiles: [
                if (resume != null)
                  _ModeTile(
                    key: const Key('resume-match'),
                    icon: Icons.play_arrow_rounded,
                    title: l10n.resumeMatch,
                    subtitle: l10n.homeQuickDuelSub,
                    highlight: true,
                    onTap: () => context.go(Routes.match(resume)),
                  )
                else
                  _ModeTile(
                    key: const Key('play-quick-duel'),
                    icon: Icons.bolt,
                    title: l10n.matchTypeQuickDuel,
                    subtitle: l10n.homeQuickDuelSub,
                    highlight: true,
                    onTap: () => _play(context, 'quick_duel'),
                  ),
                _ModeTile(
                  key: const Key('play-ranked'),
                  icon: Icons.emoji_events_rounded,
                  title: l10n.matchTypeRanked,
                  subtitle: l10n.homeRankedSub,
                  color: BvColors.of(context).gold,
                  onTap: () => _play(context, 'ranked'),
                ),
                _ModeTile(
                  key: const Key('play-sudden-death'),
                  icon: Icons.whatshot_rounded,
                  title: l10n.matchTypeSuddenDeath,
                  subtitle: l10n.roomSuddenDeathSub,
                  color: BvColors.of(context).danger,
                  onTap: () => _play(context, 'sudden_death'),
                ),
                _ModeTile(
                  key: const Key('play-marathon'),
                  icon: Icons.directions_run_rounded,
                  title: l10n.matchTypeMarathon,
                  subtitle: l10n.roomMarathonSub,
                  color: const Color(0xFF22D3EE),
                  onTap: () => _play(context, 'marathon'),
                ),
              ],
            ),
            SectionTitle(l10n.homeMoreTitle),
            _Shortcuts(
              items: [
                _Shortcut(
                  const Key('play-modes'),
                  Icons.grid_view_rounded,
                  l10n.homeAllModes,
                  () => context.push(Routes.modes),
                ),
                _Shortcut(
                  const Key('private-room'),
                  Icons.lock_outline_rounded,
                  l10n.privateRoom,
                  () => showRoomSheet(context, ref),
                ),
                _Shortcut(
                  const Key('open-teams'),
                  Icons.groups_rounded,
                  l10n.homeTeams,
                  () => context.push(Routes.teams),
                ),
                _Shortcut(
                  const Key('open-friends'),
                  Icons.people_alt_rounded,
                  l10n.homeFriends,
                  () => context.push(Routes.friends),
                ),
                _Shortcut(
                  const Key('open-challenges'),
                  Icons.sports_kabaddi_rounded,
                  l10n.homeChallenges,
                  () => context.push(Routes.challenges),
                ),
                _Shortcut(
                  const Key('open-quizzes'),
                  Icons.edit_note_rounded,
                  l10n.homeQuizzes,
                  () => context.push(Routes.quizzes),
                ),
                _Shortcut(
                  const Key('open-quests'),
                  Icons.flag_rounded,
                  l10n.questsTitle,
                  () => context.push(Routes.quests),
                ),
                _Shortcut(
                  const Key('open-badges'),
                  Icons.workspace_premium_rounded,
                  l10n.badgesTitle,
                  () => context.push(Routes.badges),
                ),
              ],
            ),
            if (dev) ...[
              const SizedBox(height: 14),
              ActionCard(
                key: const Key('play-practice'),
                icon: Icons.smart_toy_outlined,
                title: l10n.playPractice,
                subtitle: l10n.homePracticeSub,
                onTap: () => context.push(Routes.practice),
              ),
            ],
            if (session != null) ...[
              const SizedBox(height: 14),
              const StatsRow(),
            ],
          ],
        ),
      ),
    );
  }
}

/// Starts a mode from Home: Ranked searches at once (no topic choice);
/// the others ask for the topic first (D-095), then search.
Future<void> _play(BuildContext context, String matchType) async {
  if (matchType == 'ranked') {
    unawaited(
      context.push(Routes.playMode(const QueueMode(matchType: 'ranked'))),
    );
    return;
  }
  final topic = await pickTopic(context);
  if (topic == null || !context.mounted) {
    return;
  }
  final mode = QueueMode(
    matchType: matchType,
    categoryMode: topic.all ? null : 'category_master',
    categoryId: topic.categoryId,
  );
  unawaited(context.push(mode.isDefault ? Routes.play : Routes.playMode(mode)));
}

class _TopBar extends ConsumerWidget {
  const _TopBar();

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
          child: Icon(icon, color: BvColors.of(context).ink),
        ),
      ),
    );
    return Row(
      children: [
        const Expanded(
          child: FittedBox(
            fit: BoxFit.scaleDown,
            alignment: Alignment.centerLeft,
            child: BrandLogo(size: 28),
          ),
        ),
        Badge(
          key: const Key('inbox-badge'),
          isLabelVisible: unread > 0,
          label: Text('$unread'),
          offset: const Offset(-2, 2),
          child: round(
            const Key('open-inbox'),
            l10n.inboxTitle,
            Icons.notifications_none_rounded,
            Routes.inbox,
          ),
        ),
        round(
          const Key('open-settings'),
          l10n.settingsTitle,
          Icons.settings_rounded,
          Routes.settings,
        ),
      ],
    );
  }
}

/// The player card: avatar with the brand ring, greeting, level and XP.
class _PlayerCard extends StatelessWidget {
  const _PlayerCard({this.name, this.avatar});

  final String? name;
  final String? avatar;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      padding: const EdgeInsets.all(14),
      child: Row(
        children: [
          InkWell(
            key: const Key('open-profile'),
            customBorder: const CircleBorder(),
            onTap: () => context.go(Routes.profile),
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                AvatarBubble(
                  avatarFor(avatar ?? 'penguin'),
                  size: 66,
                  online: true,
                ),
                const Positioned(
                  top: -14,
                  left: 0,
                  right: 0,
                  child: Text(
                    '👑',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 20),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 14),
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
                Text(
                  l10n.homeWelcome,
                  style: text.bodySmall?.copyWith(color: bv.muted),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (name != null) ...[
                  const SizedBox(height: 10),
                  const LevelSummary(),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// The featured card: today's challenge (the daily run).
class _DailyCard extends StatelessWidget {
  const _DailyCard({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      key: const Key('open-daily'),
      gradient: bv.heroGradient,
      onTap: onTap,
      padding: EdgeInsets.zero,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(20),
        child: Stack(
          children: [
            Positioned(
              right: -18,
              bottom: -26,
              child: Icon(
                Icons.emoji_events_rounded,
                size: 140,
                color: Colors.white.withAlpha(40),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Pill(
                          l10n.homeDailyTag,
                          icon: Icons.local_fire_department_rounded,
                          color: Colors.white,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          l10n.dailyTitle,
                          style: text.titleLarge?.copyWith(color: Colors.white),
                        ),
                        Text(
                          l10n.homeDailySub,
                          style: text.bodyMedium?.copyWith(
                            color: Colors.white.withAlpha(215),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 12),
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 18,
                      vertical: 10,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(30),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.play_arrow_rounded, color: bv.gradientStart),
                        Text(
                          l10n.homeDailyPlay,
                          style: text.labelLarge?.copyWith(
                            color: bv.gradientStart,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A game mode tile: a tinted icon badge, the name and a short line;
/// [highlight] fills it with the brand gradient (the main call to action).
class _ModeTile extends StatelessWidget {
  const _ModeTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.color,
    this.highlight = false,
    super.key,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final Color? color;
  final bool highlight;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final tint = color ?? bv.accent;
    return BvCard(
      gradient: highlight ? bv.gradient : null,
      onTap: onTap,
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(14),
              color: highlight ? Colors.white.withAlpha(50) : null,
              gradient: highlight
                  ? null
                  : LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [tint, tint.withAlpha(140)],
                    ),
              boxShadow: [
                if (!highlight)
                  BoxShadow(color: tint.withAlpha(90), blurRadius: 12),
              ],
            ),
            child: Icon(icon, color: Colors.white, size: 26),
          ),
          const Spacer(),
          Text(
            title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: text.titleMedium?.copyWith(color: Colors.white),
          ),
          Text(
            subtitle,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: text.bodySmall?.copyWith(
              color: highlight ? Colors.white.withAlpha(220) : bv.muted,
            ),
          ),
        ],
      ),
    );
  }
}

/// Mode tiles, two per row.
class _ModeGrid extends StatelessWidget {
  const _ModeGrid({required this.tiles});

  final List<Widget> tiles;

  @override
  Widget build(BuildContext context) => Column(
    children: [
      for (var i = 0; i < tiles.length; i += 2) ...[
        if (i > 0) const SizedBox(height: 12),
        SizedBox(
          height: 98 + MediaQuery.textScalerOf(context).scale(34),
          child: Row(
            children: [
              Expanded(child: tiles[i]),
              const SizedBox(width: 12),
              Expanded(
                child: i + 1 < tiles.length ? tiles[i + 1] : const SizedBox(),
              ),
            ],
          ),
        ),
      ],
    ],
  );
}

class _Shortcut {
  const _Shortcut(this.key, this.icon, this.label, this.onTap);

  final Key key;
  final IconData icon;
  final String label;
  final VoidCallback onTap;
}

/// Small shortcuts, four per row: an icon badge over a label.
class _Shortcuts extends StatelessWidget {
  const _Shortcuts({required this.items});

  final List<_Shortcut> items;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    Widget cell(_Shortcut s) => BvCard(
      key: s.key,
      onTap: s.onTap,
      radius: 18,
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 10),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          IconBadge(s.icon, size: 40),
          const SizedBox(height: 6),
          Text(
            s.label,
            maxLines: 2,
            textAlign: TextAlign.center,
            overflow: TextOverflow.ellipsis,
            style: text.labelMedium?.copyWith(height: 1.1),
          ),
        ],
      ),
    );
    return Column(
      children: [
        for (var i = 0; i < items.length; i += 4) ...[
          if (i > 0) const SizedBox(height: 10),
          SizedBox(
            // Two label lines, growing with the text size.
            height: 70 + MediaQuery.textScalerOf(context).scale(26),
            child: Row(
              children: [
                for (var j = i; j < i + 4; j++) ...[
                  if (j > i) const SizedBox(width: 10),
                  Expanded(
                    child: j < items.length ? cell(items[j]) : const SizedBox(),
                  ),
                ],
              ],
            ),
          ),
        ],
      ],
    );
  }
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
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: BvCard(
        color: bv.accent.withAlpha(30),
        padding: const EdgeInsets.fromLTRB(14, 8, 8, 8),
        child: Row(
          children: [
            Icon(Icons.info_outline_rounded, color: bv.accent),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                text,
                style: Theme.of(context).textTheme.bodySmall
                    ?.copyWith(color: bv.ink),
              ),
            ),
            TextButton(
              onPressed: onPressed,
              style: TextButton.styleFrom(foregroundColor: bv.accent),
              child: Text(action),
            ),
          ],
        ),
      ),
    );
  }
}
