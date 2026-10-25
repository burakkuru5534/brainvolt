import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/bv.dart';
import '../../ads/domain/ads_repository.dart';
import '../../ads/presentation/ads.dart';
import '../../match/presentation/play_error_text.dart';
import '../../store/domain/store_repository.dart';
import '../../store/presentation/volt_screen.dart';
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
      ..invalidate(myProgressProvider)
      ..invalidate(walletProvider);
  }

  /// A rewarded ad buys one more reroll (D-103), then rerolls [id].
  Future<String?> _adReroll(
    WidgetRef ref,
    AppLocalizations l10n,
    String id,
  ) async {
    final (outcome, _) = await ref
        .read(rewardedAdRunnerProvider)
        .watch(AdPlacement.questReroll);
    if (adOutcomeText(l10n, outcome) case final text?) {
      return text;
    }
    final repo = ref.read(progressRepositoryProvider);
    // The extra reroll is applied a moment after the ad is verified.
    for (var i = 0; ; i++) {
      try {
        await repo.reroll(id);
        return null;
      } on ApiException catch (e) {
        if (e.code != 'QUEST_REROLL_LIMIT' || i >= 4) {
          rethrow;
        }
        await Future<void>.delayed(const Duration(seconds: 1));
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final repo = ref.watch(progressRepositoryProvider);
    final adRerolls = ref.watch(adsConfigProvider).value?.questRerollsLeft ?? 0;
    final rerollPrice =
        ref.watch(walletProvider).value?.priceOf(StoreItemCode.questReroll) ??
        30;
    Widget tile(Quest q, bool canReroll, IconData icon) => Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: BvCard(
        key: Key('quest-${q.id}'),
        color: q.completed && !q.claimed ? bv.success.withAlpha(30) : null,
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                IconBadge(q.claimed ? Icons.check_rounded : icon, size: 40),
                const SizedBox(width: 12),
                Expanded(child: Text(q.title, style: text.titleSmall)),
                const SizedBox(width: 8),
                Pill(
                  l10n.questReward(q.rewardXp),
                  icon: Icons.bolt,
                  color: bv.gold,
                ),
                if (q.rewardVolt > 0) ...[
                  const SizedBox(width: 6),
                  Pill(
                    l10n.questVoltReward(q.rewardVolt),
                    key: Key('quest-volt-${q.id}'),
                    icon: voltIcon,
                    color: bv.accent,
                  ),
                ],
              ],
            ),
            const SizedBox(height: 12),
            GradientProgressBar(
              value: q.target == 0 ? 0 : q.progress / q.target,
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  '${q.progress} / ${q.target}',
                  style: text.labelMedium?.copyWith(color: bv.muted),
                ),
                const Spacer(),
                if (q.claimed)
                  Text(
                    l10n.questClaimed,
                    style: text.labelLarge?.copyWith(color: bv.success),
                  )
                else if (q.completed)
                  SizedBox(
                    width: 140,
                    child: GradientButton(
                      key: Key('quest-claim-${q.id}'),
                      height: 40,
                      label: l10n.questClaim,
                      onPressed: () => _run(
                        context,
                        ref,
                        () async => l10n.questXpGained(await repo.claim(q.id)),
                      ),
                    ),
                  )
                else if (canReroll)
                  TextButton.icon(
                    key: Key('quest-reroll-${q.id}'),
                    icon: const Icon(Icons.autorenew_rounded, size: 18),
                    onPressed: () => _run(context, ref, () async {
                      await repo.reroll(q.id);
                      return null;
                    }),
                    label: Text(l10n.questReroll),
                  )
                else ...[
                  if (adRerolls > 0)
                    TextButton.icon(
                      key: Key('quest-ad-reroll-${q.id}'),
                      icon: const Icon(Icons.ondemand_video_rounded, size: 18),
                      onPressed: () =>
                          _run(context, ref, () => _adReroll(ref, l10n, q.id)),
                      label: Text(l10n.adQuestReroll),
                    ),
                  // An extra reroll for Volt (D-105).
                  IconButton(
                    key: Key('quest-volt-reroll-${q.id}'),
                    tooltip: l10n.voltRerollWith(rerollPrice),
                    icon: const Icon(voltIcon),
                    color: bv.accent,
                    onPressed: () async {
                      if (await buyVoltItem(
                            context,
                            ref,
                            StoreItemCode.questReroll,
                            rerollPrice,
                          ) &&
                          context.mounted) {
                        await _run(context, ref, () async {
                          await repo.reroll(q.id);
                          return null;
                        });
                      }
                    },
                  ),
                ],
              ],
            ),
          ],
        ),
      ),
    );
    Widget empty() => BvCard(
      child: Row(
        children: [
          const IconBadge(Icons.flag_rounded, size: 40),
          const SizedBox(width: 12),
          Expanded(child: Text(l10n.questsEmpty, style: text.bodyMedium)),
        ],
      ),
    );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.questsTitle)),
      body: switch (ref.watch(questsProvider)) {
        AsyncData(:final value) => ListView(
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 24),
          children: [
            Row(
              children: [
                Expanded(child: SectionTitle(l10n.questsDaily)),
                Pill(
                  l10n.questRerollsLeft(value.rerollsLeft),
                  key: const Key('rerolls-left'),
                  icon: Icons.autorenew_rounded,
                ),
              ],
            ),
            if (value.daily.isEmpty) empty(),
            for (final q in value.daily)
              tile(q, value.rerollsLeft > 0, Icons.today_rounded),
            SectionTitle(l10n.questsWeekly),
            if (value.weekly.isEmpty) empty(),
            for (final q in value.weekly)
              tile(q, value.rerollsLeft > 0, Icons.date_range_rounded),
          ],
        ),
        AsyncError(:final error) => ErrorView(
          message: playErrorText(l10n, error),
          retryLabel: l10n.retry,
          onRetry: () => ref.invalidate(questsProvider),
        ),
        _ => const LoadingView(),
      },
    );
  }
}
