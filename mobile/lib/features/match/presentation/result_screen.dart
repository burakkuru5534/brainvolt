import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../data/match_providers.dart';
import '../domain/match_repository.dart';
import 'play_error_text.dart';

part 'result_screen.g.dart';

/// GET /v1/matches/{id} for the result screen.
@riverpod
Future<MatchResult> matchResult(Ref ref, String matchId) =>
    ref.watch(matchRepositoryProvider).result(matchId);

/// The finished match round by round: the question, the correct answer,
/// and whether this player got it right.
class ResultScreen extends ConsumerWidget {
  const ResultScreen({required this.matchId, super.key});

  final String matchId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final result = ref.watch(matchResultProvider(matchId));
    return Scaffold(
      appBar: AppBar(title: Text(l10n.resultTitle)),
      body: switch (result) {
        AsyncData(:final value) => _body(context, value, me),
        AsyncError(:final error) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(playErrorText(l10n, error)),
              TextButton(
                onPressed: () => ref.invalidate(matchResultProvider(matchId)),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  Widget _body(BuildContext context, MatchResult r, String me) {
    final l10n = context.l10n;
    final mySide =
        r.players.where((p) => p.userId == me).firstOrNull?.side ?? 1;
    final headline = switch (r) {
      MatchResult(status: 'cancelled') => l10n.matchCancelled,
      MatchResult(isDraw: true) => l10n.matchDraw,
      _ when r.winnerSide == mySide => l10n.matchWon,
      _ => l10n.matchLost,
    };
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(
          headline,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.headlineMedium,
        ),
        Text(
          '${r.scores[mySide - 1]} — ${r.scores[2 - mySide]}',
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.displaySmall,
        ),
        const SizedBox(height: 16),
        for (final round in r.rounds)
          Card(
            child: ListTile(
              leading: Text(switch (round.answers
                  .where((a) => a.userId == me)
                  .firstOrNull) {
                ResultAnswer(correct: true) => '✅',
                _ => '❌',
              }, style: const TextStyle(fontSize: 24)),
              title: Text('${round.roundNo}. ${round.prompt}'),
              subtitle: Text(l10n.resultCorrectAnswer(round.correctText)),
              trailing: Text(
                '+${round.answers.where((a) => a.userId == me).firstOrNull?.points ?? 0}',
              ),
            ),
          ),
      ],
    );
  }
}
