import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/progress_providers.dart';
import '../domain/progress_repository.dart';
import 'progress_widgets.dart';

final questsProvider = FutureProvider.autoDispose<QuestList>(
  (ref) => ref.watch(progressRepositoryProvider).quests(),
);

/// Today's and this week's quests with claim and the daily reroll.
class QuestsScreen extends ConsumerWidget {
  const QuestsScreen({super.key});

  Future<void> _run(
    BuildContext context,
    WidgetRef ref,
    Future<String?> Function() action,
  ) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final msg = await action();
      if (msg != null) {
        messenger.showSnackBar(SnackBar(content: Text(msg)));
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref
      ..invalidate(questsProvider)
      ..invalidate(myProgressProvider);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final repo = ref.watch(progressRepositoryProvider);
    Widget tile(Quest q, bool canReroll) => Card(
      key: Key('quest-${q.id}'),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    q.title,
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ),
                Text(l10n.questReward(q.rewardXp)),
              ],
            ),
            const SizedBox(height: 6),
            LinearProgressIndicator(
              value: q.target == 0 ? 0 : q.progress / q.target,
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Text('${q.progress} / ${q.target}'),
                const Spacer(),
                if (q.claimed)
                  Text(l10n.questClaimed)
                else if (q.completed)
                  FilledButton(
                    key: Key('quest-claim-${q.id}'),
                    onPressed: () => _run(
                      context,
                      ref,
                      () async => l10n.questXpGained(await repo.claim(q.id)),
                    ),
                    child: Text(l10n.questClaim),
                  )
                else if (canReroll)
                  TextButton(
                    key: Key('quest-reroll-${q.id}'),
                    onPressed: () => _run(context, ref, () async {
                      await repo.reroll(q.id);
                      return null;
                    }),
                    child: Text(l10n.questReroll),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.questsTitle)),
      body: switch (ref.watch(questsProvider)) {
        AsyncData(:final value) => ListView(
          padding: const EdgeInsets.all(12),
          children: [
            Text(
              l10n.questsDaily,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Text(
              l10n.questRerollsLeft(value.rerollsLeft),
              key: const Key('rerolls-left'),
            ),
            for (final q in value.daily) tile(q, value.rerollsLeft > 0),
            const SizedBox(height: 16),
            Text(
              l10n.questsWeekly,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            for (final q in value.weekly) tile(q, value.rerollsLeft > 0),
          ],
        ),
        AsyncError(:final error) => Center(
          child: Text(playErrorText(l10n, error)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}
