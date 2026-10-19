import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/session_controller.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../data/match_providers.dart';
import '../domain/match_reducer.dart';
import '../domain/match_state.dart';

part 'play_controllers.g.dart';

/// The queue screen's state.
sealed class QueueState {
  const QueueState();
}

final class QueueIdle extends QueueState {
  const QueueIdle();
}

final class QueueSearching extends QueueState {
  const QueueSearching(this.since);
  final DateTime since;
}

final class QueueFound extends QueueState {
  const QueueFound(this.matchId);
  final String matchId;
}

final class QueueFailed extends QueueState {
  const QueueFailed(this.error);
  final Object error;
}

/// Looking for an opponent (rest.md §5, websocket.md mm.*). The match is
/// announced by mm.match_found; /v1/matches/current is polled as well in
/// case the push was missed while the socket reconnected.
@riverpod
class QueueController extends _$QueueController {
  static const pollEvery = Duration(seconds: 3);

  StreamSubscription<ServerFrame>? _sub;
  Timer? _poll;

  @override
  QueueState build() {
    ref.onDispose(_stopWatching);
    return const QueueIdle();
  }

  Future<void> search({bool practice = false}) async {
    final rt = ref.read(realtimeClientProvider)..start();
    final repo = ref.read(matchRepositoryProvider);
    _stopWatching();
    _sub = rt.frames.listen((f) {
      if (state is! QueueSearching) {
        return;
      }
      if (f.type == 'mm.match_found') {
        _found(f.data['match_id'] as String);
      } else if (f.type == 'mm.status' && f.data['state'] == 'cancelled') {
        _stopWatching();
        state = QueueFailed(
          RealtimeError(f.data['reason'] as String? ?? 'INTERNAL'),
        );
      }
    });
    state = QueueSearching(DateTime.now());
    try {
      final t = await repo.enqueue(practice: practice);
      if (t.matchId != null) {
        _found(t.matchId!);
        return;
      }
      _poll = Timer.periodic(pollEvery, (_) async {
        final id = await repo.currentMatch().catchError((_) => null);
        if (id != null && state is QueueSearching) {
          _found(id);
        }
      });
    } on Object catch (e) {
      _stopWatching();
      state = QueueFailed(e);
    }
  }

  Future<void> cancel() async {
    _stopWatching();
    state = const QueueIdle();
    await ref.read(matchRepositoryProvider).leaveQueue();
  }

  void _found(String id) {
    _stopWatching();
    state = QueueFound(id);
  }

  void _stopWatching() {
    unawaited(_sub?.cancel());
    _sub = null;
    _poll?.cancel();
    _poll = null;
  }
}

/// One live match: joins it on every (re)connect, folds the match stream
/// into [MatchState] and sends the player's intents.
@riverpod
class MatchController extends _$MatchController {
  @override
  MatchState build(String matchId) {
    final rt = ref.watch(realtimeClientProvider)..start();
    final me = switch (ref.read(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final stream = 'match:$matchId';
    final frames = rt.frames.listen((f) {
      if (f.stream == stream) {
        state = reduce(state, f);
        if (state.over) {
          rt.forget(stream);
        }
      }
    });
    final status = rt.statusChanges.listen((s) {
      if (s == RealtimeStatus.connected) {
        unawaited(_join());
      }
    });
    ref.onDispose(() {
      unawaited(frames.cancel());
      unawaited(status.cancel());
    });
    if (rt.status == RealtimeStatus.connected) {
      unawaited(Future.microtask(_join));
    }
    return MatchState(matchId: matchId, myUserId: me);
  }

  RealtimeClient get _rt => ref.read(realtimeClientProvider);

  Future<void> _join() async {
    try {
      await _rt.send('match.join', {'match_id': matchId});
    } on RealtimeError {
      // Shown through the connection banner; the next connect retries.
    }
  }

  Future<void> ready() => _rt.send('match.ready', {'match_id': matchId});

  /// Sends the answer; the server's round.answer_received confirms it. The
  /// choice is shown at once and taken back if the server refuses it.
  Future<void> answer(String optionId) async {
    final round = state.round;
    if (round == null || round.myOptionId != null || round.locked) {
      return;
    }
    state = state.copyWith(round: round.copyWith(myOptionId: optionId));
    try {
      await _rt.send('match.answer', {
        'match_id': matchId,
        'round_no': round.roundNo,
        'option_id': optionId,
      });
    } on RealtimeError {
      final now = state.round;
      if (now != null &&
          now.roundNo == round.roundNo &&
          now.correctOptionId == null) {
        state = state.copyWith(
          round: RoundState(
            roundNo: now.roundNo,
            prompt: now.prompt,
            category: now.category,
            options: now.options,
            showAt: now.showAt,
            deadlineAt: now.deadlineAt,
            answered: now.answered,
            locked: now.locked,
          ),
        );
      }
      rethrow;
    }
  }

  Future<void> leave() => _rt.send('match.leave', {'match_id': matchId});
}

/// The rematch offer of a finished match (websocket.md rematch.*).
class RematchState {
  const RematchState({
    this.asked = false,
    this.othersAccepted = false,
    this.newMatchId,
    this.declined = false,
  });

  /// This player asked for the rematch.
  final bool asked;

  /// Another participant asked for it.
  final bool othersAccepted;
  final String? newMatchId;
  final bool declined;
}

/// "Play again" on the result screen: everyone accepts within 30 s of the
/// end → game creates the rematch (rematch.result carries its id).
@riverpod
class RematchController extends _$RematchController {
  @override
  RematchState build(String matchId) {
    final rt = ref.watch(realtimeClientProvider)..start();
    final me = switch (ref.read(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    final sub = rt.frames.listen((f) {
      if (f.data['match_id'] != matchId) {
        return;
      }
      if (f.type == 'rematch.offer') {
        final ids = (f.data['accepted_user_ids'] as List?) ?? const [];
        state = RematchState(
          asked: state.asked || ids.contains(me),
          othersAccepted: ids.any((id) => id != me),
        );
      } else if (f.type == 'rematch.result') {
        state = f.data['accepted'] == true
            ? RematchState(
                asked: true,
                newMatchId: f.data['new_match_id'] as String?,
              )
            : const RematchState(declined: true);
      }
    });
    ref.onDispose(() => unawaited(sub.cancel()));
    return const RematchState();
  }

  Future<void> respond({required bool accept}) async {
    if (accept) {
      state = RematchState(asked: true, othersAccepted: state.othersAccepted);
    }
    try {
      await ref.read(realtimeClientProvider).send('rematch.respond', {
        'match_id': matchId,
        'accept': accept,
      });
    } on RealtimeError {
      state = const RematchState(declined: true);
      rethrow;
    }
  }
}
