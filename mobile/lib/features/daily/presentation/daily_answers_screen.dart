import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/daily_providers.dart';
import '../domain/daily_repository.dart';

final dailyAnswersProvider = FutureProvider.autoDispose<DailyAnswers>(
  (ref) => ref.watch(dailyRepositoryProvider).answers(),
);

/// "Yesterday's answers" (D-100): the finished day's questions with the
/// right option, the player's own pick and the explanation. A day's
/// answers open only once the Istanbul day is over, so nobody can learn
/// them and replay the challenge from another account.
class DailyAnswersScreen extends ConsumerWidget {
  const DailyAnswersScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.dailyAnswersTitle)),
      body: switch (ref.watch(dailyAnswersProvider)) {
        AsyncData(:final value) when value.questions.isEmpty => EmptyState(
          icon: Icons.fact_check_rounded,
          text: l10n.dailyAnswersEmpty,
          textKey: const Key('daily-answers-empty'),
        ),
        AsyncData(:final value) => ListView(
          padding: const EdgeInsets.all(16),
          children: [
            for (final q in value.questions) ...[
              _QuestionCard(q),
              const SizedBox(height: 12),
            ],
          ],
        ),
        AsyncError(:final error) => ErrorView(
          message: playErrorText(l10n, error),
          onRetry: () => ref.invalidate(dailyAnswersProvider),
        ),
        _ => const LoadingView(),
      },
    );
  }
}

class _QuestionCard extends StatelessWidget {
  const _QuestionCard(this.q);

  final DailyAnswerReview q;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      key: Key('daily-answer-${q.position}'),
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Pill('${q.position}', icon: Icons.tag_rounded),
              if (q.category.isNotEmpty) ...[
                const SizedBox(width: 8),
                Flexible(child: Pill(q.category, icon: Icons.public)),
              ],
              const Spacer(),
              Icon(
                q.yourOptionId == null
                    ? Icons.remove_circle_outline_rounded
                    : q.yourCorrect
                    ? Icons.check_circle_rounded
                    : Icons.cancel_rounded,
                color: q.yourOptionId == null
                    ? bv.muted
                    : q.yourCorrect
                    ? bv.success
                    : bv.danger,
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(q.prompt, style: text.titleMedium),
          const SizedBox(height: 10),
          for (final (i, (id, label)) in q.options.indexed)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: OptionTile(
                key: Key('daily-answer-${q.position}-$id'),
                letter: String.fromCharCode(65 + i),
                text: label,
                look: id == q.correctOptionId
                    ? OptionLook.correct
                    : id == q.yourOptionId
                    ? OptionLook.wrong
                    : OptionLook.idle,
                onPressed: null,
                footer: id == q.yourOptionId
                    ? Text(
                        l10n.dailyAnswersYours,
                        style: text.labelMedium?.copyWith(color: Colors.white),
                      )
                    : null,
              ),
            ),
          if (q.yourOptionId == null)
            Text(
              l10n.dailyAnswersNotPlayed,
              style: text.bodySmall?.copyWith(color: bv.muted),
            ),
          if (q.explanation case final e? when e.isNotEmpty) ...[
            const SizedBox(height: 4),
            NoticeBar(e, icon: Icons.lightbulb_outline_rounded),
          ],
        ],
      ),
    );
  }
}
