import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../data/daily_providers.dart';
import '../domain/daily_repository.dart';

final dailyTodayProvider = FutureProvider.autoDispose<DailyToday>(
  (ref) => ref.watch(dailyRepositoryProvider).today(),
);

final dailyBoardProvider = FutureProvider.autoDispose
    .family<DailyBoard, (String, String)>(
      (ref, k) => ref.watch(dailyRepositoryProvider).board(k.$1, k.$2),
    );

/// Seconds with one decimal.
String seconds(int ms) => (ms / 1000).toStringAsFixed(1);

/// Today's Daily Challenge: start / resume the official run, its result
/// and place, practice, and the day's board (category-modes §5).
class DailyScreen extends ConsumerStatefulWidget {
  const DailyScreen({super.key});

  @override
  ConsumerState<DailyScreen> createState() => _DailyScreenState();
}

class _DailyScreenState extends ConsumerState<DailyScreen> {
  var _scope = 'global';

  Future<void> _start(String kind) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final run = await ref.read(dailyRepositoryProvider).start(kind);
      if (mounted) {
        await context.push(Routes.dailyRun(run.id));
        ref.invalidate(dailyTodayProvider);
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.dailyTitle)),
      body: switch (ref.watch(dailyTodayProvider)) {
        AsyncData(:final value) => _body(context, value),
        AsyncError(:final error) => ErrorView(
          message: playErrorText(l10n, error),
        ),
        _ => const LoadingView(),
      },
    );
  }

  Widget _body(BuildContext context, DailyToday today) {
    final l10n = context.l10n;
    final official = today.official;
    final board = ref.watch(dailyBoardProvider((today.day, _scope))).value;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        BvCard(
          gradient: BvColors.of(context).heroGradient,
          padding: const EdgeInsets.all(18),
          child: Row(
            children: [
              const IconBadge(Icons.today_rounded, size: 56, onGradient: true),
              const SizedBox(width: 14),
              Expanded(
                child: Text(
                  l10n.dailyIntro,
                  style: Theme.of(context).textTheme.bodyLarge
                      ?.copyWith(color: Colors.white),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        if (!today.available)
          Text(l10n.dailyNotAvailable, key: const Key('daily-unavailable'))
        else if (official == null ||
            !(official.completed || official.inProgress))
          GradientButton(
            key: const Key('daily-start'),
            icon: Icons.play_arrow_rounded,
            onPressed: () => _start('official'),
            label: l10n.dailyStart,
          )
        else if (official.inProgress)
          GradientButton(
            key: const Key('daily-resume'),
            icon: Icons.play_arrow_rounded,
            onPressed: () async {
              await context.push(Routes.dailyRun(official.id));
              ref.invalidate(dailyTodayProvider);
            },
            label: l10n.dailyResume,
          )
        else ...[
          BvCard(
            child: Column(
              children: [
                Icon(
                  Icons.emoji_events_rounded,
                  color: BvColors.of(context).gold,
                  size: 36,
                ),
                const SizedBox(height: 6),
                Text(
                  l10n.dailyResult(
                    official.correct ?? 0,
                    seconds(official.totalTimeMs ?? 0),
                  ),
                  key: const Key('daily-result'),
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
              ],
            ),
          ),
          const SizedBox(height: 6),
          if (board?.me case final me?)
            Text(
              l10n.dailyRank(me.rank),
              key: const Key('daily-rank'),
              textAlign: TextAlign.center,
            ),
          const SizedBox(height: 8),
          if (today.practiceAllowed)
            OutlinedButton.icon(
              key: const Key('daily-practice'),
              icon: const Icon(Icons.school),
              onPressed: () => _start('practice'),
              label: Text(l10n.dailyPractice),
            ),
        ],
        const SizedBox(height: 8),
        SectionTitle(l10n.dailyBoardTitle),
        WcSegmented<String>(
          keyPrefix: 'daily-scope',
          selected: _scope,
          onChanged: (s) => setState(() => _scope = s),
          segments: [
            ('global', l10n.scopeGlobal),
            ('country', l10n.scopeCountry),
            ('friends', l10n.scopeFriends),
          ],
        ),
        const SizedBox(height: 12),
        if (board != null && board.entries.isEmpty)
          EmptyState(icon: Icons.leaderboard, text: l10n.leaderboardEmpty),
        for (final e in board?.entries ?? const <DailyEntry>[])
          TileCard(
            highlight: e.userId == board?.me?.userId
                ? BvColors.of(context).gradientStart
                : null,
            child: ListTile(
              key: Key('daily-row-${e.userId}'),
              leading: SizedBox(
                width: 84,
                child: Row(
                  children: [
                    SizedBox(
                      width: 36,
                      child: Text(
                        '#${e.rank}',
                        style: TextStyle(
                          color: e.rank <= 3
                              ? BvColors.of(context).gold
                              : BvColors.of(context).muted,
                        ),
                      ),
                    ),
                    AvatarBubble(avatarFor(e.user?.avatarId ?? ''), size: 40),
                  ],
                ),
              ),
              title: Text(e.user?.displayName ?? ''),
              trailing: Text(
                l10n.dailyResult(e.correct, seconds(e.totalTimeMs)),
                style: TextStyle(color: BvColors.of(context).accent),
              ),
            ),
          ),
      ],
    );
  }
}
