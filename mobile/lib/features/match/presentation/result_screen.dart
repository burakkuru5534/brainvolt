import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/router.dart';
import '../../../app/session_controller.dart';
import '../../../core/l10n/l10n.dart';
import '../../lobby/data/lobby_providers.dart';
import '../../lobby/presentation/room_controller.dart';
import '../../profile/domain/avatars.dart';
import '../../progress/presentation/progress_widgets.dart';
import '../data/match_providers.dart';
import '../domain/match_repository.dart';
import 'mode_picker_screen.dart';
import 'play_controllers.dart';
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
        if (r.players.length > 2) _TeamLines(result: r, mySide: mySide),
        if (r.revenge case final rv? when rv.chain > 0)
          Text(
            l10n.revengeSeries(rv.wins, rv.losses),
            key: const Key('revenge-series'),
            textAlign: TextAlign.center,
          ),
        if (r.revenge case RevengeInfo(available: true))
          _Revenge(matchId: r.id),
        if (r.status == 'completed')
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: MatchAwardsCard(matchId: r.id),
          ),
        if (r.status == 'completed' &&
            r.origin != 'private_room' &&
            (r.endedAt
                    ?.add(const Duration(seconds: 30))
                    .isAfter(DateTime.now()) ??
                false))
          _Rematch(matchId: r.id),
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

/// Revenge (D-045): the losing side picks a home category and challenges
/// the winners; they answer from their challenges.
class _Revenge extends ConsumerStatefulWidget {
  const _Revenge({required this.matchId});

  final String matchId;

  @override
  ConsumerState<_Revenge> createState() => _RevengeState();
}

class _RevengeState extends ConsumerState<_Revenge> {
  var _sent = false;

  Future<void> _ask() async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final cats = await ref.read(categoriesProvider.future);
      if (!mounted) {
        return;
      }
      final cat = await showDialog<String>(
        context: context,
        builder: (context) => SimpleDialog(
          title: Text(l10n.revengePickCategory),
          children: [
            for (final c in cats)
              SimpleDialogOption(
                key: Key('revenge-cat-${c.id}'),
                onPressed: () => Navigator.pop(context, c.id),
                child: Text(c.name),
              ),
          ],
        ),
      );
      if (cat == null) {
        return;
      }
      await ref.read(lobbyRepositoryProvider).revenge(widget.matchId, cat);
      if (mounted) {
        setState(() => _sent = true);
      }
      messenger.showSnackBar(SnackBar(content: Text(l10n.revengeSent)));
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: OutlinedButton.icon(
        key: const Key('revenge'),
        icon: const Icon(Icons.sports_mma),
        onPressed: _sent ? null : _ask,
        label: Text(_sent ? l10n.revengeSent : l10n.revengeButton),
      ),
    );
  }
}

/// Rematch offer: ask, see the opponent's answer, go to the new match.
class _Rematch extends ConsumerWidget {
  const _Rematch({required this.matchId});

  final String matchId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    ref.listen(rematchControllerProvider(matchId), (_, next) {
      if (next.newMatchId case final id?) {
        context.go(Routes.match(id));
      }
    });
    final s = ref.watch(rematchControllerProvider(matchId));
    final ctrl = ref.read(rematchControllerProvider(matchId).notifier);
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: Column(
        children: [
          if (s.declined)
            Text(l10n.rematchDeclined)
          else if (s.asked)
            Text(l10n.rematchWaiting, key: const Key('rematch-waiting'))
          else ...[
            if (s.othersAccepted) Text(l10n.rematchOffered),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                FilledButton.icon(
                  key: const Key('rematch'),
                  icon: const Icon(Icons.replay),
                  onPressed: () =>
                      ctrl.respond(accept: true).catchError((Object _) {}),
                  label: Text(l10n.rematchButton),
                ),
                if (s.othersAccepted) ...[
                  const SizedBox(width: 8),
                  TextButton(
                    onPressed: () =>
                        ctrl.respond(accept: false).catchError((Object _) {}),
                    child: Text(l10n.decline),
                  ),
                ],
              ],
            ),
          ],
        ],
      ),
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
