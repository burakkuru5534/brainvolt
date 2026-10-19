import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
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
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
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
        Text(l10n.dailyIntro, textAlign: TextAlign.center),
        const SizedBox(height: 16),
        if (!today.available)
          Text(l10n.dailyNotAvailable, key: const Key('daily-unavailable'))
        else if (official == null ||
            !(official.completed || official.inProgress))
          FilledButton.icon(
            key: const Key('daily-start'),
            icon: const Icon(Icons.play_arrow),
            onPressed: () => _start('official'),
            label: Text(l10n.dailyStart),
          )
        else if (official.inProgress)
          FilledButton.icon(
            key: const Key('daily-resume'),
            icon: const Icon(Icons.play_arrow),
            onPressed: () async {
              await context.push(Routes.dailyRun(official.id));
              ref.invalidate(dailyTodayProvider);
            },
            label: Text(l10n.dailyResume),
          )
        else ...[
          Text(
            l10n.dailyResult(
              official.correct ?? 0,
              seconds(official.totalTimeMs ?? 0),
            ),
            key: const Key('daily-result'),
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.titleLarge,
          ),
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
        const SizedBox(height: 24),
        Text(
          l10n.dailyBoardTitle,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        Wrap(
          spacing: 6,
          children: [
            for (final (scope, name) in [
              ('global', l10n.scopeGlobal),
              ('country', l10n.scopeCountry),
              ('friends', l10n.scopeFriends),
            ])
              ChoiceChip(
                key: Key('daily-scope-$scope'),
                label: Text(name),
                selected: _scope == scope,
                onSelected: (_) => setState(() => _scope = scope),
              ),
          ],
        ),
        if (board != null && board.entries.isEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(l10n.leaderboardEmpty, textAlign: TextAlign.center),
          ),
        for (final e in board?.entries ?? const <DailyEntry>[])
          ListTile(
            key: Key('daily-row-${e.userId}'),
            selected: e.userId == board?.me?.userId,
            leading: Text('#${e.rank}'),
            title: Text(
              '${avatarFor(e.user?.avatarId ?? '')} ${e.user?.displayName ?? ''}',
            ),
            trailing: Text(l10n.dailyResult(e.correct, seconds(e.totalTimeMs))),
          ),
      ],
    );
  }
}
