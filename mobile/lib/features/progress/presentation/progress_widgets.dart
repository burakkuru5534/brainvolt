import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../data/progress_providers.dart';
import '../domain/progress_repository.dart';

/// The caller's XP, level and streak.
final myProgressProvider = FutureProvider.autoDispose<Progress>(
  (ref) => ref.watch(progressRepositoryProvider).progress(),
);

/// The caller's standing in the current season.
final myStandingProvider = FutureProvider.autoDispose<Standing?>((ref) async {
  final id = switch (ref.watch(sessionControllerProvider).value) {
    SignedIn(:final session) => session.user.id,
    _ => null,
  };
  if (id == null) {
    return null;
  }
  return ref.watch(progressRepositoryProvider).standing('user', id);
});

/// A league's display name.
String leagueName(AppLocalizations l10n, String code) => switch (code) {
  'silver' => l10n.leagueSilver,
  'gold' => l10n.leagueGold,
  'platinum' => l10n.leaguePlatinum,
  'diamond' => l10n.leagueDiamond,
  _ => l10n.leagueBronze,
};

/// League with its division (III … I); Diamond has none.
String placementName(AppLocalizations l10n, String league, int division) {
  final name = leagueName(l10n, league);
  if (league == 'diamond') {
    return name;
  }
  return '$name ${const {1: 'I', 2: 'II', 3: 'III'}[division] ?? ''}'.trim();
}

/// Home's level line: level chip, XP bar and play streak.
class LevelSummary extends ConsumerWidget {
  const LevelSummary({this.onGradient = false, super.key});

  /// White text for the dark hero card.
  final bool onGradient;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final progress = ref.watch(myProgressProvider).value;
    final bv = BvColors.of(context);
    if (progress == null) {
      return const SizedBox(height: 32);
    }
    final small = Theme.of(context).textTheme.labelMedium;
    return Column(
      key: const Key('progress-card'),
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Pill(
              l10n.levelLabel(progress.level),
              icon: Icons.bolt,
              gradient: true,
            ),
            const SizedBox(width: 10),
            Expanded(
              child: GradientProgressBar(
                value: progress.levelSize == 0
                    ? 1
                    : progress.levelXp / progress.levelSize,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        Row(
          children: [
            Text(
              '🔥 ${l10n.streakDays(progress.streak)}',
              key: const Key('progress-streak'),
              style: small?.copyWith(color: bv.muted),
            ),
            const Spacer(),
            Text(
              l10n.xpProgress(progress.levelXp, progress.levelSize),
              style: small?.copyWith(color: bv.muted),
            ),
          ],
        ),
      ],
    );
  }
}

/// Home's stats: total wins and the current league.
class StatsRow extends ConsumerWidget {
  const StatsRow({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final progress = ref.watch(myProgressProvider).value;
    final standing = ref.watch(myStandingProvider).value;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    Widget stat(
      IconData icon,
      Color color,
      String label,
      String value,
      Key key, [
      String? extra,
    ]) => Expanded(
      child: Row(
        children: [
          Icon(icon, size: 40, color: color),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label.toUpperCase(), style: text.labelSmall),
                Text(
                  value,
                  key: key,
                  style: text.headlineSmall,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (extra != null) Text(extra, style: text.labelSmall),
              ],
            ),
          ),
        ],
      ),
    );
    return BvCard(
      onTap: () => context.push(Routes.leaderboards),
      child: IntrinsicHeight(
        child: Row(
          children: [
            stat(
              Icons.emoji_events,
              bv.gold,
              l10n.homeTotalWins,
              '${progress?.wins ?? 0}',
              const Key('progress-wins'),
            ),
            VerticalDivider(color: bv.cardBorder, width: 24),
            stat(
              Icons.shield,
              bv.badgeInk,
              l10n.homeCurrentRank,
              standing == null
                  ? l10n.homeNoRank
                  : placementName(l10n, standing.league, standing.division),
              const Key('progress-league'),
              standing == null
                  ? null
                  : l10n.seasonPoints(standing.seasonPoints),
            ),
          ],
        ),
      ),
    );
  }
}

/// The result screen's rewards: the two match.awards messages of the
/// match (D-077), read back over REST if they came before the screen.
class MatchAwardsCard extends ConsumerStatefulWidget {
  const MatchAwardsCard({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<MatchAwardsCard> createState() => _MatchAwardsCardState();
}

class _MatchAwardsCardState extends ConsumerState<MatchAwardsCard> {
  StreamSubscription<Object>? _sub;
  Timer? _fallback;
  var _awards = const MatchAwards();

  @override
  void initState() {
    super.initState();
    _sub = ref.read(realtimeClientProvider).frames.listen((f) {
      if (f.type != 'match.awards') {
        return;
      }
      final d = f.data;
      if (d['match_id'] == widget.matchId && mounted) {
        setState(() => _awards = _awards.merge(d));
      }
    });
    _fallback = Timer(const Duration(seconds: 3), _fetch);
  }

  Future<void> _fetch() async {
    if (_awards.xp != null) {
      return;
    }
    try {
      final read = await ref
          .read(progressRepositoryProvider)
          .awards(widget.matchId);
      if (mounted && _awards.xp == null && (read.xp ?? 0) > 0) {
        setState(
          () => _awards = MatchAwards(
            xp: read.xp,
            level: read.level,
            streak: read.streak,
            badges: read.badges,
            spDelta: _awards.spDelta,
            league: _awards.league,
            division: _awards.division,
            leagueUp: _awards.leagueUp,
          ),
        );
      }
    } on Object {
      // The awards stay pending; the profile shows them later.
    }
  }

  @override
  void dispose() {
    _fallback?.cancel();
    unawaited(_sub?.cancel());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final a = _awards;
    final lines = <String>[
      if (a.xp case final xp?) l10n.awardsXp(xp),
      if (a.levelUp && a.level != null) l10n.awardsLevelUp(a.level!),
      if (a.spDelta case final sp?) l10n.awardsSp(sp >= 0 ? '+$sp' : '$sp'),
      if (a.leagueUp && a.league != null)
        l10n.awardsLeagueUp(placementName(l10n, a.league!, a.division ?? 3)),
      for (final b in a.badges) l10n.awardsBadge(b),
      if (a.questsCompleted > 0) l10n.awardsQuests(a.questsCompleted),
    ];
    final bv = BvColors.of(context);
    return BvCard(
      key: const Key('match-awards'),
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.card_giftcard_rounded, color: bv.gold),
              const SizedBox(width: 8),
              Text(
                l10n.awardsTitle,
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 8),
          if (a.isEmpty)
            Text(l10n.awardsPending, style: TextStyle(color: bv.muted))
          else
            for (final line in lines)
              Padding(
                padding: const EdgeInsets.only(bottom: 2),
                child: Row(
                  children: [
                    Icon(Icons.auto_awesome, size: 16, color: bv.accent),
                    const SizedBox(width: 6),
                    Expanded(child: Text(line)),
                  ],
                ),
              ),
        ],
      ),
    );
  }
}
