import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../chat/domain/chat_repository.dart';
import '../../chat/presentation/chat_screen.dart';
import '../../profile/domain/avatars.dart';
import '../domain/match_state.dart';
import 'match_type_name.dart';
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

  /// Quick phrases on screen: user → (phrase, shown until).
  final _quick = <String, (String, DateTime)>{};
  StreamSubscription<ServerFrame>? _quickSub;

  @override
  void initState() {
    super.initState();
    _quickSub = ref.read(realtimeClientProvider).frames.listen((f) {
      if (f.type == 'match.quick_chat' &&
          f.data['match_id'] == widget.matchId) {
        _quick[f.data['user_id'] as String] = (
          f.data['phrase_id'] as String? ?? '',
          DateTime.now().add(const Duration(seconds: 3)),
        );
      }
    });
    // Countdowns and the show_at gate redraw ten times a second.
    _tick = Timer.periodic(
      const Duration(milliseconds: 100),
      (_) => setState(() {}),
    );
  }

  @override
  void dispose() {
    _tick?.cancel();
    unawaited(_quickSub?.cancel());
    super.dispose();
  }

  Future<void> _pickQuick() async {
    final l10n = context.l10n;
    final phrase = await showModalBottomSheet<String>(
      context: context,
      builder: (context) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          child: Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final q in quickPhrases)
                ActionChip(
                  key: Key('match-quick-$q'),
                  label: Text(quickPhraseText(l10n, q)),
                  onPressed: () => Navigator.pop(context, q),
                ),
            ],
          ),
        ),
      ),
    );
    if (phrase == null || !mounted) {
      return;
    }
    await _run(
      () => ref.read(realtimeClientProvider).send('match.quick_chat', {
        'match_id': widget.matchId,
        'phrase_id': phrase,
      }),
    );
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
        body: SafeArea(
          child: Column(
            children: [
              _Header(
                title: state.round != null && state.questionCount > 0
                    ? l10n.roundOf(state.round!.roundNo, state.questionCount)
                    : matchTypeName(l10n, state.matchType),
                progress: state.round != null && state.questionCount > 0
                    ? state.round!.roundNo / state.questionCount
                    : null,
                onLeave: _leave,
              ),
              if (rt.status != RealtimeStatus.connected)
                _Notice(l10n.reconnecting, key: const Key('reconnecting')),
              if (opponent?.status == 'disconnected')
                _Notice(l10n.opponentDisconnected),
              if (state.seatMoved) _Notice(l10n.seatMoved),
              if (state.resumed && state.phase == MatchPhase.question)
                _Notice(l10n.matchResumed),
              _Scoreboard(state: state),
              _QuickLine(
                quick: {
                  for (final e in _quick.entries)
                    if (e.value.$2.isAfter(DateTime.now())) e.key: e.value.$1,
                },
                names: {for (final p in state.players) p.userId: p.displayName},
                me: state.myUserId,
                onPick: state.over ? null : _pickQuick,
              ),
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
                              onAnswer: (id) => _run(
                                () => state.sharedAnswer
                                    ? _ctrl.propose(id)
                                    : _ctrl.answer(id),
                              ),
                              onLock: (id) => _run(() => _ctrl.lock(id)),
                            ),
                    MatchPhase.completed ||
                    MatchPhase.cancelled => _End(state: state),
                  },
                ),
              ),
            ],
          ),
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
            GradientButton(
              key: const Key('ready'),
              label: l10n.readyButton,
              icon: Icons.bolt,
              onPressed: () async {
                setState(() => _readySent = true);
                await _run(_ctrl.ready);
              },
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
    margin: const EdgeInsets.fromLTRB(16, 4, 16, 0),
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
    decoration: BoxDecoration(
      color: BvColors.of(context).badge,
      borderRadius: BorderRadius.circular(14),
    ),
    child: Text(text, textAlign: TextAlign.center),
  );
}

/// "Question n/N", the leave button and the match progress.
class _Header extends StatelessWidget {
  const _Header({required this.title, required this.onLeave, this.progress});

  final String title;
  final double? progress;
  final VoidCallback onLeave;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(20, 8, 16, 4),
    child: Column(
      children: [
        Row(
          children: [
            Expanded(
              child: Text(title, style: Theme.of(context).textTheme.titleLarge),
            ),
            BvCard(
              key: const Key('match-leave'),
              radius: 24,
              padding: const EdgeInsets.all(8),
              onTap: onLeave,
              child: Icon(Icons.close, color: BvColors.of(context).ink),
            ),
          ],
        ),
        if (progress != null) ...[
          const SizedBox(height: 10),
          LinearProgressIndicator(value: progress!.clamp(0, 1)),
        ],
      ],
    ),
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
    final text = Theme.of(context).textTheme;
    Widget side(Player? p, String name, int score, {required bool mine}) {
      final avatar = AvatarBubble(avatarFor(p?.avatarId ?? ''), size: 64);
      final info = Expanded(
        child: Column(
          crossAxisAlignment: mine
              ? CrossAxisAlignment.start
              : CrossAxisAlignment.end,
          children: [
            Text(name, overflow: TextOverflow.ellipsis, style: text.labelLarge),
            Text(
              '$score',
              key: Key(mine ? 'my-score' : 'opponent-score'),
              style: text.headlineMedium,
            ),
            if (!mine &&
                p != null &&
                answered.contains(p.userId) &&
                state.phase == MatchPhase.question)
              Text(
                '✓ ${l10n.opponentAnswered}',
                key: const Key('opponent-answered'),
                style: text.labelSmall,
              ),
          ],
        ),
      );
      return Expanded(
        child: Row(
          children: mine
              ? [avatar, const SizedBox(width: 8), info]
              : [info, const SizedBox(width: 8), avatar],
        ),
      );
    }

    if (state.isTeam) {
      Widget team(int sideNo, String name, int score) => Expanded(
        child: Column(
          children: [
            Wrap(
              alignment: WrapAlignment.center,
              children: [
                for (final p in state.sidePlayers(sideNo))
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 2),
                    child: Badge(
                      isLabelVisible:
                          answered.contains(p.userId) &&
                          state.phase == MatchPhase.question,
                      label: const Text('✓'),
                      child: Text(
                        avatarFor(p.avatarId),
                        style: const TextStyle(fontSize: 26),
                      ),
                    ),
                  ),
              ],
            ),
            Text(name, overflow: TextOverflow.ellipsis),
            Text(
              '$score',
              key: Key(sideNo == state.mySide ? 'my-score' : 'opponent-score'),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
          ],
        ),
      );
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 12),
        child: Row(
          children: [
            team(state.mySide, l10n.yourTeam, state.myScore),
            const _Versus(),
            team(3 - state.mySide, l10n.opponentsTeam, state.opponentScore),
          ],
        ),
      );
    }
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          side(me, l10n.you, state.myScore, mine: true),
          const _Versus(),
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

/// Quick phrases just sent in the match, and the button to send one.
class _QuickLine extends StatelessWidget {
  const _QuickLine({
    required this.quick,
    required this.names,
    required this.me,
    required this.onPick,
  });

  final Map<String, String> quick;
  final Map<String, String> names;
  final String me;
  final VoidCallback? onPick;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Expanded(
            child: Wrap(
              spacing: 6,
              runSpacing: 4,
              children: [
                for (final e in quick.entries)
                  Chip(
                    key: Key('quick-shown-${e.key}'),
                    backgroundColor: e.key == me ? bv.badge : bv.card,
                    label: Text(
                      '${e.key == me ? l10n.you : names[e.key] ?? ''}: '
                      '${quickPhraseText(l10n, e.value)}',
                    ),
                  ),
              ],
            ),
          ),
          if (onPick != null)
            IconButton(
              key: const Key('match-quick'),
              tooltip: l10n.chatTitle,
              icon: Icon(Icons.chat_bubble_outline, color: bv.badgeInk),
              onPressed: onPick,
            ),
        ],
      ),
    );
  }
}

/// The VS shield between the two sides.
class _Versus extends StatelessWidget {
  const _Versus();

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 6),
      child: Stack(
        alignment: Alignment.center,
        children: [
          ShaderMask(
            shaderCallback: (r) => bv.gradient.createShader(r),
            child: const Icon(Icons.shield, size: 52, color: Colors.white),
          ),
          Text(
            'VS',
            style: weighted(
              const TextStyle(
                fontSize: 16,
                color: Colors.white,
                fontFamily: brandFont,
              ),
              FontWeight.w900,
            ),
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
    required this.onLock,
  });

  final MatchState state;
  final int now;

  /// Answer (1v1, Individual Turns, Fast Team Round) or propose (Shared
  /// Answer).
  final ValueChanged<String> onAnswer;
  final ValueChanged<String> onLock;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final r = state.round!;
    final revealed = r.correctOptionId != null;
    if (!revealed && now < r.showAt) {
      return const Center(child: CircularProgressIndicator());
    }
    final window = (r.deadlineAt - r.showAt).clamp(1, 1 << 31);
    final left = (r.deadlineAt - now).clamp(0, window);
    // The server decides lateness; the buttons stay on a little past zero.
    final myTurn = r.answering.isEmpty || r.answering.contains(state.myUserId);
    final open =
        !revealed &&
        !r.locked &&
        r.myOptionId == null &&
        myTurn &&
        now <= r.deadlineAt + 250;
    final mine = r.revealed[state.myUserId];
    final myProposal = r.proposals[state.myUserId];
    String nameOf(String id) =>
        state.players.where((p) => p.userId == id).firstOrNull?.displayName ??
        '';
    // Individual Turns: whose turn it is on my side.
    final turnHolder = state.teamMode == 'individual_turns'
        ? state
              .sidePlayers(state.mySide)
              .where((p) => r.answering.contains(p.userId))
              .firstOrNull
        : null;
    final sideResult = state.isTeam ? r.sideCorrect[state.mySide] : null;
    // Long prompts and four options scroll on small phones.
    return SingleChildScrollView(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          LinearProgressIndicator(
            key: const Key('round-timer'),
            value: revealed ? 0 : left / window,
          ),
          const SizedBox(height: 16),
          if (r.category != null)
            Align(
              child: Chip(
                avatar: Icon(
                  Icons.public,
                  size: 18,
                  color: BvColors.of(context).badgeInk,
                ),
                label: Text(r.category!),
              ),
            ),
          const SizedBox(height: 8),
          BvCard(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 22),
            child: Text(
              r.prompt,
              key: const Key('prompt'),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
          ),
          if (turnHolder != null && !revealed) ...[
            const SizedBox(height: 8),
            Text(
              turnHolder.userId == state.myUserId
                  ? l10n.yourTurn
                  : l10n.turnOf(turnHolder.displayName),
              key: const Key('turn'),
              textAlign: TextAlign.center,
            ),
          ],
          const SizedBox(height: 16),
          for (final (i, o) in r.options.indexed)
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: OptionTile(
                key: Key('option-${o.id}'),
                letter: String.fromCharCode(65 + i),
                text: o.text,
                look: switch (o.id) {
                  _ when revealed && o.id == r.correctOptionId =>
                    OptionLook.correct,
                  _ when revealed && o.id == r.myOptionId => OptionLook.wrong,
                  _ when o.id == r.myOptionId => OptionLook.chosen,
                  _ when o.id == myProposal => OptionLook.proposed,
                  _ => OptionLook.idle,
                },
                onPressed: open ? () => onAnswer(o.id) : null,
                footer: state.sharedAnswer && !revealed
                    ? Wrap(
                        spacing: 6,
                        children: [
                          for (final e in r.proposals.entries)
                            if (e.value == o.id && e.key != state.myUserId)
                              Text(
                                nameOf(e.key),
                                style: Theme.of(context).textTheme.labelSmall,
                              ),
                        ],
                      )
                    : null,
              ),
            ),
          if (state.sharedAnswer && !revealed)
            FilledButton(
              key: const Key('lock'),
              onPressed: open && myProposal != null
                  ? () => onLock(myProposal)
                  : null,
              child: Text(l10n.lockAnswer),
            ),
          if (revealed) ...[
            const SizedBox(height: 8),
            _Feedback(
              good: sideResult ?? (mine?.correct ?? false),
              text: switch (mine) {
                _ when sideResult != null =>
                  sideResult ? l10n.answerCorrect : l10n.answerWrong,
                RevealedAnswer(optionId: null) || null => l10n.answerTimeout,
                RevealedAnswer(correct: true) => l10n.answerCorrect,
                _ => l10n.answerWrong,
              },
              points: (mine?.points ?? 0) > 0
                  ? l10n.pointsGained(mine!.points)
                  : null,
            ),
            if (r.explanation != null) ...[
              const SizedBox(height: 8),
              Text(r.explanation!, textAlign: TextAlign.center),
            ],
          ],
        ],
      ),
    );
  }
}

/// Correct / wrong after the reveal, with the points and, when right, a
/// burst of sparkles.
class _Feedback extends StatelessWidget {
  const _Feedback({required this.good, required this.text, this.points});

  final bool good;
  final String text;
  final String? points;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final color = good ? bv.success : bv.danger;
    final body = Column(
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              good ? Icons.check_circle : Icons.cancel,
              color: color,
              size: 36,
            ),
            const SizedBox(width: 8),
            Flexible(
              child: Text(
                text,
                key: const Key('reveal-result'),
                style: Theme.of(context).textTheme.headlineSmall
                    ?.copyWith(color: color),
              ),
            ),
          ],
        ),
        if (points != null)
          Text(points!, style: Theme.of(context).textTheme.titleLarge),
      ],
    );
    return good
        ? CustomPaint(
            painter: _Sparkles(bv),
            child: Padding(padding: const EdgeInsets.all(12), child: body),
          )
        : body;
  }
}

class _Sparkles extends CustomPainter {
  _Sparkles(this.bv);

  final BvColors bv;

  @override
  void paint(Canvas canvas, Size size) {
    final colors = [
      bv.gradientStart,
      bv.success,
      const Color(0xFFF5B301),
      bv.badgeInk,
    ];
    // Fixed positions: a calm, deterministic burst around the text.
    const spots = [
      (0.05, 0.2),
      (0.12, 0.85),
      (0.2, 0.45),
      (0.3, 0.05),
      (0.7, 0.1),
      (0.8, 0.9),
      (0.88, 0.4),
      (0.95, 0.75),
      (0.4, 0.95),
      (0.62, 0.92),
    ];
    for (final (i, (x, y)) in spots.indexed) {
      final paint = Paint()..color = colors[i % colors.length];
      final c = Offset(x * size.width, y * size.height);
      if (i.isEven) {
        canvas.drawCircle(c, 3, paint);
      } else {
        final path = Path()
          ..moveTo(c.dx, c.dy - 6)
          ..lineTo(c.dx + 2, c.dy - 2)
          ..lineTo(c.dx + 6, c.dy)
          ..lineTo(c.dx + 2, c.dy + 2)
          ..lineTo(c.dx, c.dy + 6)
          ..lineTo(c.dx - 2, c.dy + 2)
          ..lineTo(c.dx - 6, c.dy)
          ..lineTo(c.dx - 2, c.dy - 2)
          ..close();
        canvas.drawPath(path, paint);
      }
    }
  }

  @override
  bool shouldRepaint(_Sparkles old) => false;
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
          const SizedBox(height: 12),
          GradientButton(
            key: const Key('back-home'),
            label: l10n.backHome,
            icon: Icons.home_rounded,
            onPressed: () => context.go(Routes.home),
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
