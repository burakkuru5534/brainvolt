import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/dialogs.dart';
import '../../profile/domain/avatars.dart';
import '../domain/match_state.dart';
import 'play_controllers.dart';
import 'play_error_text.dart';

/// The live match: ready, countdown, questions shown at `show_at`, the
/// answer, the reveal and the end. Every number comes from the server; the
/// app only counts down to server deadlines (timing.md §6).
class MatchScreen extends ConsumerStatefulWidget {
  const MatchScreen({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<MatchScreen> createState() => _MatchScreenState();
}

class _MatchScreenState extends ConsumerState<MatchScreen> {
  Timer? _tick;
  bool _readySent = false;

  @override
  void initState() {
    super.initState();
    // Countdowns and the show_at gate redraw ten times a second.
    _tick = Timer.periodic(
      const Duration(milliseconds: 100),
      (_) => setState(() {}),
    );
  }

  @override
  void dispose() {
    _tick?.cancel();
    super.dispose();
  }

  MatchController get _ctrl =>
      ref.read(matchControllerProvider(widget.matchId).notifier);

  Future<void> _run(Future<void> Function() action) async {
    try {
      await action();
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
          ..hideCurrentSnackBar()
          ..showSnackBar(
            SnackBar(content: Text(playErrorText(context.l10n, e))),
          );
      }
    }
  }

  Future<void> _leave() async {
    final state = ref.read(matchControllerProvider(widget.matchId));
    if (state.over) {
      context.go(Routes.home);
      return;
    }
    final l10n = context.l10n;
    final ok = await confirmDialog(
      context,
      title: l10n.leaveMatchTitle,
      body: l10n.leaveMatchBody,
      confirmLabel: l10n.leaveButton,
      destructive: true,
    );
    if (!ok || !mounted) {
      return;
    }
    await _run(_ctrl.leave);
    if (mounted) {
      context.go(Routes.home);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final state = ref.watch(matchControllerProvider(widget.matchId));
    final rt = ref.watch(realtimeClientProvider);
    final now = rt.serverNowMs();
    final opponent = state.opponent;
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          unawaited(_leave());
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: Text(
            state.round != null && state.questionCount > 0
                ? l10n.roundOf(state.round!.roundNo, state.questionCount)
                : l10n.playQuickDuel,
          ),
          leading: IconButton(
            key: const Key('match-leave'),
            icon: const Icon(Icons.close),
            onPressed: _leave,
          ),
        ),
        body: Column(
          children: [
            if (rt.status != RealtimeStatus.connected)
              _Notice(l10n.reconnecting, key: const Key('reconnecting')),
            if (opponent?.status == 'disconnected')
              _Notice(l10n.opponentDisconnected),
            if (state.seatMoved) _Notice(l10n.seatMoved),
            if (state.resumed && state.phase == MatchPhase.question)
              _Notice(l10n.matchResumed),
            _Scoreboard(state: state),
            const Divider(height: 1),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: switch (state.phase) {
                  MatchPhase.waiting => _waiting(context, state, now),
                  MatchPhase.countdown => Center(
                    child: Text(
                      l10n.matchStartsIn(
                        ((state.startsAt - now) / 1000).ceil().clamp(0, 9),
                      ),
                      key: const Key('countdown'),
                      style: Theme.of(context).textTheme.displayLarge,
                    ),
                  ),
                  MatchPhase.preRound => _PickView(
                    state: state,
                    now: now,
                    onPick: (id) => _run(() => _ctrl.pick(id)),
                  ),
                  MatchPhase.question || MatchPhase.reveal =>
                    state.round == null
                        ? const Center(child: CircularProgressIndicator())
                        : _RoundView(
                            state: state,
                            now: now,
                            onAnswer: (id) => _run(() => _ctrl.answer(id)),
                          ),
                  MatchPhase.completed ||
                  MatchPhase.cancelled => _End(state: state),
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _waiting(BuildContext context, MatchState state, int now) {
    final l10n = context.l10n;
    final ready = _readySent || (state.me?.ready ?? false);
    final left = ((state.readyDeadlineAt - now) / 1000).ceil();
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            l10n.matchFoundTitle,
            style: Theme.of(context).textTheme.titleLarge,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          if (state.readyDeadlineAt > 0 && left >= 0)
            Text(l10n.readyDeadline(left)),
          const SizedBox(height: 24),
          if (ready)
            Text(l10n.waitingForOpponent)
          else
            FilledButton(
              key: const Key('ready'),
              onPressed: () async {
                setState(() => _readySent = true);
                await _run(_ctrl.ready);
              },
              child: Text(l10n.readyButton),
            ),
        ],
      ),
    );
  }
}

class _Notice extends StatelessWidget {
  const _Notice(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    color: Theme.of(context).colorScheme.secondaryContainer,
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
    child: Text(text, textAlign: TextAlign.center),
  );
}

class _Scoreboard extends StatelessWidget {
  const _Scoreboard({required this.state});

  final MatchState state;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final me = state.me;
    final opponent = state.opponent;
    final answered = state.round?.answered ?? const {};
    Widget side(Player? p, String name, int score, {required bool mine}) =>
        Expanded(
          child: Column(
            children: [
              Text(
                avatarFor(p?.avatarId ?? ''),
                style: const TextStyle(fontSize: 32),
              ),
              Text(name, overflow: TextOverflow.ellipsis),
              Text(
                '$score',
                key: Key(mine ? 'my-score' : 'opponent-score'),
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              if (!mine &&
                  p != null &&
                  answered.contains(p.userId) &&
                  state.phase == MatchPhase.question)
                Text(
                  '✓ ${l10n.opponentAnswered}',
                  key: const Key('opponent-answered'),
                ),
            ],
          ),
        );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        children: [
          side(me, l10n.you, state.myScore, mine: true),
          const Text('—'),
          side(
            opponent,
            opponent?.displayName ?? '',
            state.opponentScore,
            mine: false,
          ),
        ],
      ),
    );
  }
}

class _RoundView extends StatelessWidget {
  const _RoundView({
    required this.state,
    required this.now,
    required this.onAnswer,
  });

  final MatchState state;
  final int now;
  final ValueChanged<String> onAnswer;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final r = state.round!;
    final scheme = Theme.of(context).colorScheme;
    final revealed = r.correctOptionId != null;
    if (!revealed && now < r.showAt) {
      return const Center(child: CircularProgressIndicator());
    }
    final window = (r.deadlineAt - r.showAt).clamp(1, 1 << 31);
    final left = (r.deadlineAt - now).clamp(0, window);
    // The server decides lateness; the buttons stay on a little past zero.
    final open =
        !revealed &&
        !r.locked &&
        r.myOptionId == null &&
        now <= r.deadlineAt + 250;
    final mine = r.revealed[state.myUserId];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        LinearProgressIndicator(
          key: const Key('round-timer'),
          value: revealed ? 0 : left / window,
        ),
        const SizedBox(height: 16),
        if (r.category != null) Align(child: Chip(label: Text(r.category!))),
        const SizedBox(height: 8),
        Text(
          r.prompt,
          key: const Key('prompt'),
          style: Theme.of(context).textTheme.titleLarge,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),
        for (final o in r.options)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: FilledButton.tonal(
              key: Key('option-${o.id}'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.all(16),
                backgroundColor: switch (o.id) {
                  _ when revealed && o.id == r.correctOptionId =>
                    Colors.green.shade400,
                  _ when revealed && o.id == r.myOptionId => scheme.error,
                  _ when o.id == r.myOptionId => scheme.primaryContainer,
                  _ => null,
                },
              ),
              onPressed: open ? () => onAnswer(o.id) : null,
              child: Text(o.text, textAlign: TextAlign.center),
            ),
          ),
        if (revealed) ...[
          const SizedBox(height: 8),
          Text(
            switch (mine) {
              RevealedAnswer(optionId: null) || null => l10n.answerTimeout,
              RevealedAnswer(correct: true) => l10n.answerCorrect,
              _ => l10n.answerWrong,
            },
            key: const Key('reveal-result'),
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          if ((mine?.points ?? 0) > 0)
            Text(l10n.pointsGained(mine!.points), textAlign: TextAlign.center),
          if (r.explanation != null) ...[
            const SizedBox(height: 8),
            Text(r.explanation!, textAlign: TextAlign.center),
          ],
        ],
      ],
    );
  }
}

class _End extends StatelessWidget {
  const _End({required this.state});

  final MatchState state;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final opponentGone = {'left', 'forfeited'}.contains(state.opponent?.status);
    final headline = switch (state) {
      MatchState(phase: MatchPhase.cancelled) => l10n.matchCancelled,
      MatchState(isDraw: true) => l10n.matchDraw,
      _ when state.winnerSide == state.mySide => l10n.matchWon,
      _ => l10n.matchLost,
    };
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            headline,
            key: const Key('match-result'),
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          if (opponentGone && state.phase == MatchPhase.completed) ...[
            const SizedBox(height: 8),
            Text(l10n.opponentLeft),
          ],
          const SizedBox(height: 8),
          Text(
            '${state.myScore} — ${state.opponentScore}',
            style: Theme.of(context).textTheme.displaySmall,
          ),
          const SizedBox(height: 24),
          if (state.phase == MatchPhase.completed)
            OutlinedButton(
              key: const Key('show-details'),
              onPressed: () => context.push(Routes.matchResult(state.matchId)),
              child: Text(l10n.showDetails),
            ),
          FilledButton(
            key: const Key('back-home'),
            onPressed: () => context.go(Routes.home),
            child: Text(l10n.backHome),
          ),
        ],
      ),
    );
  }
}

/// Mixed Battle: the picker chooses the next category, the others wait.
class _PickView extends StatelessWidget {
  const _PickView({
    required this.state,
    required this.now,
    required this.onPick,
  });

  final MatchState state;
  final int now;
  final void Function(String id) onPick;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final pre = state.preRound;
    if (pre == null) {
      return const Center(child: CircularProgressIndicator());
    }
    final mine = pre.pickerUserId == state.myUserId;
    final picker = state.players
        .where((p) => p.userId == pre.pickerUserId)
        .firstOrNull;
    final left = ((pre.deadlineAt - now) / 1000).ceil().clamp(0, 99);
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          mine
              ? l10n.pickCategory
              : l10n.opponentPicking(picker?.displayName ?? ''),
          key: const Key('pre-round'),
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 8),
        Text('$left', textAlign: TextAlign.center),
        const SizedBox(height: 16),
        for (final (id, name) in pre.categories)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: FilledButton.tonal(
              key: Key('pick-$id'),
              onPressed: mine && pre.allowed.contains(id)
                  ? () => onPick(id)
                  : null,
              child: Text(name),
            ),
          ),
      ],
    );
  }
}
