import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import 'daily_screen.dart';

/// A question of a run as daily.question sent it.
class _Question {
  _Question(Map<String, dynamic> d)
    : position = (d['position'] as num).toInt(),
      total = (d['total'] as num?)?.toInt() ?? 10,
      showAt = (d['show_at'] as num).toInt(),
      deadline = (d['deadline'] as num).toInt(),
      prompt = d['prompt'] as String? ?? '',
      category = d['category'] as String? ?? '',
      practice = d['kind'] == 'practice',
      options = [
        for (final o in (d['options'] as List? ?? const []))
          ((o as Map)['id'] as String, o['text'] as String? ?? ''),
      ];

  final int position;
  final int total;
  final int showAt;
  final int deadline;
  final String prompt;
  final String category;
  final bool practice;
  final List<(String, String)> options;
}

/// A solo Daily Challenge run over the WebSocket (D-040): daily.join
/// starts or resumes it; the clock never stops, so after a deadline the
/// screen syncs again to get the reveal and the next question.
class DailyRunScreen extends ConsumerStatefulWidget {
  const DailyRunScreen({required this.runId, super.key});

  final String runId;

  @override
  ConsumerState<DailyRunScreen> createState() => _DailyRunScreenState();
}

class _DailyRunScreenState extends ConsumerState<DailyRunScreen> {
  final _questions = <int, _Question>{};
  final _revealed = <int, Map<String, dynamic>>{};
  final _chosen = <int, String>{};
  final _synced = <int>{};
  Map<String, dynamic>? _completed;
  var _voided = false;
  StreamSubscription<Object>? _frames;
  StreamSubscription<Object>? _status;
  Timer? _tick;

  RealtimeClient get _rt => ref.read(realtimeClientProvider);

  @override
  void initState() {
    super.initState();
    final rt = _rt..start();
    _frames = rt.frames.listen(_onFrame);
    _status = rt.statusChanges.listen((s) {
      if (s == RealtimeStatus.connected) {
        unawaited(_join());
      }
    });
    if (rt.status == RealtimeStatus.connected) {
      unawaited(Future.microtask(_join));
    }
    _tick = Timer.periodic(const Duration(milliseconds: 200), (_) => _onTick());
  }

  @override
  void dispose() {
    _tick?.cancel();
    unawaited(_frames?.cancel());
    unawaited(_status?.cancel());
    super.dispose();
  }

  Future<void> _join() async {
    try {
      await _rt.send('daily.join', {'run_id': widget.runId});
    } on RealtimeError {
      // The connection banner shows it; the next connect joins again.
    }
  }

  void _onFrame(ServerFrame f) {
    if (!f.type.startsWith('daily.') || f.data['run_id'] != widget.runId) {
      return;
    }
    setState(() {
      switch (f.type) {
        case 'daily.question':
          final q = _Question(f.data);
          _questions[q.position] = q;
        case 'daily.revealed':
          _revealed[(f.data['position'] as num).toInt()] = f.data;
        case 'daily.completed':
          _completed = f.data;
        case 'daily.voided':
          _voided = true;
      }
    });
  }

  /// The question on screen: the newest one already shown, or the last
  /// revealed one while the next waits for its show time.
  _Question? _current(int now) {
    final shown = _questions.values.where((q) => q.showAt <= now).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
    if (shown.isNotEmpty) {
      return shown.last;
    }
    return null;
  }

  void _onTick() {
    if (!mounted) {
      return;
    }
    final now = _rt.serverNowMs();
    final q = _current(now);
    // Past the deadline without a reveal: ask the server to settle.
    if (q != null &&
        !_revealed.containsKey(q.position) &&
        now > q.deadline + 400 &&
        _synced.add(q.position)) {
      unawaited(_join());
    }
    setState(() {});
  }

  Future<void> _answer(_Question q, String option) async {
    if (_chosen.containsKey(q.position)) {
      return;
    }
    setState(() => _chosen[q.position] = option);
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await _rt.send('daily.answer', {
        'run_id': widget.runId,
        'position': q.position,
        'option_id': option,
      });
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final now = _rt.serverNowMs();
    final q = _current(now);
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.dailyTitle),
        actions: [
          if (q?.practice ?? false)
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(l10n.dailyPracticeLabel),
            ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: _completed != null
            ? _result(context, _completed!)
            : _voided
            ? Center(
                child: Text(l10n.dailyVoided, key: const Key('daily-voided')),
              )
            : q == null
            ? const Center(
                child: CircularProgressIndicator(key: Key('daily-waiting')),
              )
            : SingleChildScrollView(child: _question(context, q, now)),
      ),
    );
  }

  Widget _question(BuildContext context, _Question q, int now) {
    final l10n = context.l10n;
    final reveal = _revealed[q.position];
    final left = ((q.deadline - now) / 1000).clamp(0, 15);
    final correct = reveal?['correct_option_id'] as String?;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.dailyQuestionOf(q.position, q.total),
          key: const Key('daily-position'),
        ),
        const SizedBox(height: 8),
        LinearProgressIndicator(value: reveal != null ? 0 : left / 15),
        const SizedBox(height: 16),
        if (q.category.isNotEmpty)
          Text(q.category, style: Theme.of(context).textTheme.labelLarge),
        BvCard(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 22),
          child: Text(
            q.prompt,
            key: const Key('daily-prompt'),
            style: Theme.of(context).textTheme.headlineSmall,
          ),
        ),
        const SizedBox(height: 16),
        for (final (i, (id, text)) in q.options.indexed)
          Padding(
            padding: const EdgeInsets.only(bottom: 10),
            child: OptionTile(
              key: Key('daily-option-$id'),
              letter: String.fromCharCode(65 + i),
              text: text,
              look: correct == null
                  ? (_chosen[q.position] == id
                        ? OptionLook.chosen
                        : OptionLook.idle)
                  : id == correct
                  ? OptionLook.correct
                  : (_chosen[q.position] == id
                        ? OptionLook.wrong
                        : OptionLook.idle),
              onPressed: reveal == null && !_chosen.containsKey(q.position)
                  ? () => _answer(q, id)
                  : null,
            ),
          ),
        if (reveal != null)
          Text(
            reveal['option_id'] == null
                ? l10n.dailyTimeout
                : reveal['correct'] == true
                ? l10n.dailyCorrect
                : l10n.dailyWrong,
            key: const Key('daily-reveal'),
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.titleMedium,
          ),
      ],
    );
  }

  Widget _result(BuildContext context, Map<String, dynamic> d) {
    final l10n = context.l10n;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            l10n.dailyDone,
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          const SizedBox(height: 8),
          Text(
            l10n.dailyResult(
              (d['correct_count'] as num?)?.toInt() ?? 0,
              seconds((d['total_time_ms'] as num?)?.toInt() ?? 0),
            ),
            key: const Key('daily-run-result'),
          ),
          const SizedBox(height: 16),
          FilledButton(
            key: const Key('daily-back'),
            onPressed: () => Navigator.of(context).maybePop(),
            child: Text(l10n.dailyBack),
          ),
        ],
      ),
    );
  }
}
