import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_providers.dart';
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

/// Home's progress card: level and XP bar, league and SP, play streak,
/// with shortcuts to leaderboards, quests and badges.
class ProgressCard extends ConsumerWidget {
  const ProgressCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final progress = ref.watch(myProgressProvider).value;
    final standing = ref.watch(myStandingProvider).value;
    final theme = Theme.of(context);
    return Card(
      key: const Key('progress-card'),
      margin: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (progress != null) ...[
              Row(
                children: [
                  Text(
                    l10n.levelLabel(progress.level),
                    style: theme.textTheme.titleMedium,
                  ),
                  const Spacer(),
                  Text(
                    '🔥 ${l10n.streakDays(progress.streak)}',
                    key: const Key('progress-streak'),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              LinearProgressIndicator(
                value: progress.levelSize == 0
                    ? 1
                    : progress.levelXp / progress.levelSize,
              ),
              const SizedBox(height: 4),
              Align(
                alignment: AlignmentDirectional.centerEnd,
                child: Text(
                  l10n.xpProgress(progress.levelXp, progress.levelSize),
                  style: theme.textTheme.bodySmall,
                ),
              ),
            ],
            if (standing != null)
              Row(
                children: [
                  const Icon(Icons.military_tech, size: 20),
                  const SizedBox(width: 4),
                  Text(
                    placementName(l10n, standing.league, standing.division),
                    key: const Key('progress-league'),
                  ),
                  const Spacer(),
                  Text(l10n.seasonPoints(standing.seasonPoints)),
                ],
              ),
            const SizedBox(height: 4),
            Wrap(
              spacing: 8,
              alignment: WrapAlignment.center,
              children: [
                TextButton.icon(
                  key: const Key('open-leaderboards'),
                  icon: const Icon(Icons.leaderboard),
                  onPressed: () => context.push(Routes.leaderboards),
                  label: Text(l10n.leaderboardsTitle),
                ),
                TextButton.icon(
                  key: const Key('open-quests'),
                  icon: const Icon(Icons.flag),
                  onPressed: () => context.push(Routes.quests),
                  label: Text(l10n.questsTitle),
                ),
                TextButton.icon(
                  key: const Key('open-badges'),
                  icon: const Icon(Icons.workspace_premium),
                  onPressed: () => context.push(Routes.badges),
                  label: Text(l10n.badgesTitle),
                ),
              ],
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
    return Card(
      key: const Key('match-awards'),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.awardsTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            if (a.isEmpty)
              Text(l10n.awardsPending)
            else
              for (final line in lines) Text(line),
          ],
        ),
      ),
    );
  }
}
