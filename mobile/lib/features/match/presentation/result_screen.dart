import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/share/result_card.dart';
import '../../../core/widgets/bv.dart';
import '../../lobby/presentation/room_controller.dart';
import '../../moderation/presentation/report_dialog.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../data/match_providers.dart';
import '../domain/match_repository.dart';
import 'play_again.dart';
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
      appBar: AppBar(
        title: Text(l10n.resultTitle),
        actions: [
          if (result.value case final r? when r.status == 'completed')
            IconButton(
              key: const Key('share-result'),
              tooltip: l10n.shareResult,
              icon: const Icon(Icons.ios_share),
              onPressed: () => _share(context, ref, r, me),
            ),
        ],
      ),
      body: switch (result) {
        AsyncData(:final value) => _body(context, ref, value, me),
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

  Future<void> _share(
    BuildContext context,
    WidgetRef ref,
    MatchResult r,
    String me,
  ) {
    final l10n = context.l10n;
    final mine = r.players.where((p) => p.userId == me).firstOrNull;
    final side = mine?.side ?? 1;
    final headline = switch (r) {
      MatchResult(isDraw: true) => l10n.matchDraw,
      _ when r.winnerSide == side => l10n.matchWon,
      _ => l10n.matchLost,
    };
    final score = '${r.scores[side - 1]} — ${r.scores[2 - side]}';
    return showShareCard(
      context,
      ref,
      card: ResultCard(
        key: const Key('share-card'),
        headline: headline,
        big: score,
        detail: l10n.shareCorrectOf(mine?.correct ?? 0, r.rounds.length),
        icon: r.winnerSide == side && !r.isDraw
            ? Icons.emoji_events
            : Icons.bolt,
      ),
      text: l10n.shareMatchText(score.replaceAll(' ', ''), headline),
    );
  }

  Widget _body(BuildContext context, WidgetRef ref, MatchResult r, String me) {
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
        BvCard(
          gradient: BvColors.of(context).gradient,
          padding: const EdgeInsets.symmetric(vertical: 22, horizontal: 16),
          child: Column(
            children: [
              Icon(
                r.winnerSide == mySide && !r.isDraw
                    ? Icons.emoji_events
                    : Icons.bolt,
                size: 48,
                color: const Color(0xFFFFD45C),
              ),
              Text(
                headline,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.headlineMedium
                    ?.copyWith(color: Colors.white),
              ),
              Text(
                '${r.scores[mySide - 1]} — ${r.scores[2 - mySide]}',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.displaySmall
                    ?.copyWith(color: Colors.white),
              ),
            ],
          ),
        ),
        if (r.players.length > 2) _TeamLines(result: r, mySide: mySide),
        if (r.status == 'completed')
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: MatchAwardsCard(matchId: r.id),
          ),
        if (r.status == 'completed') PlayAgainPanel(matchId: r.id),
        if (r.origin == 'private_room')
          if (ref.watch(currentRoomProvider) case final room?)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: FilledButton.icon(
                key: const Key('back-to-room'),
                icon: const Icon(Icons.meeting_room),
                onPressed: () => context.go(Routes.room(room)),
                label: Text(l10n.backToRoom),
              ),
            ),
        const SizedBox(height: 16),
        for (final round in r.rounds)
          Card(
            child: ListTile(
              leading: switch (round.answers
                  .where((a) => a.userId == me)
                  .firstOrNull) {
                ResultAnswer(correct: true) => Icon(
                  Icons.check_circle,
                  color: BvColors.of(context).success,
                  size: 30,
                ),
                _ => Icon(
                  Icons.cancel,
                  color: BvColors.of(context).danger,
                  size: 30,
                ),
              },
              title: Text('${round.roundNo}. ${round.prompt}'),
              subtitle: Text(l10n.resultCorrectAnswer(round.correctText)),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    '+${round.answers.where((a) => a.userId == me).firstOrNull?.points ?? 0}',
                  ),
                  if (round.questionId.isNotEmpty)
                    IconButton(
                      key: Key('report-q-${round.roundNo}'),
                      tooltip: l10n.reportQuestion,
                      icon: const Icon(Icons.flag_outlined),
                      onPressed: () => reportQuestion(
                        context,
                        ref,
                        questionId: round.questionId,
                        matchId: r.id,
                      ),
                    ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}

/// Team matches: each side's players with their correct answers.
class _TeamLines extends StatelessWidget {
  const _TeamLines({required this.result, required this.mySide});

  final MatchResult result;
  final int mySide;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    Widget column(int side, String title) => Expanded(
      child: Column(
        children: [
          Text(title, style: Theme.of(context).textTheme.titleSmall),
          for (final p in result.players.where((p) => p.side == side))
            Text(
              '${avatarFor(p.avatarId)} ${p.displayName} · ${p.correct}',
              key: Key('result-player-${p.userId}'),
            ),
        ],
      ),
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          column(mySide, l10n.yourTeam),
          column(3 - mySide, l10n.opponentsTeam),
        ],
      ),
    );
  }
}
