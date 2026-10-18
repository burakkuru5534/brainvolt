/// The live match as the app shows it, rebuilt from the server's match
/// stream (websocket.md §5.2). The app never decides anything: scores,
/// correctness and time come from the server (ADR-0003).
library;

import 'package:flutter/foundation.dart';

/// Where the match is.
enum MatchPhase {
  /// Created; players tap Ready before the deadline.
  waiting,

  /// Everyone is ready; the 3 s countdown runs.
  countdown,

  /// A question is open.
  question,

  /// The answers are revealed.
  reveal,

  completed,
  cancelled,
}

@immutable
class Player {
  const Player({
    required this.userId,
    required this.side,
    this.displayName = '',
    this.avatarId = '',
    this.ready = false,
    this.status = 'joined',
  });

  final String userId;
  final int side;
  final String displayName;
  final String avatarId;
  final bool ready;

  /// joined / ready / playing / disconnected / left / forfeited / finished.
  final String status;

  Player copyWith({
    String? displayName,
    String? avatarId,
    bool? ready,
    String? status,
  }) => Player(
    userId: userId,
    side: side,
    displayName: displayName ?? this.displayName,
    avatarId: avatarId ?? this.avatarId,
    ready: ready ?? this.ready,
    status: status ?? this.status,
  );
}

@immutable
class AnswerOption {
  const AnswerOption(this.id, this.text);
  final String id;
  final String text;
}

/// One participant's revealed answer.
@immutable
class RevealedAnswer {
  const RevealedAnswer({
    required this.optionId,
    required this.correct,
    required this.points,
    this.responseMs,
  });

  final String? optionId;
  final bool correct;
  final int points;
  final int? responseMs;
}

@immutable
class RoundState {
  const RoundState({
    required this.roundNo,
    required this.prompt,
    required this.options,
    required this.showAt,
    required this.deadlineAt,
    this.category,
    this.myOptionId,
    this.answered = const {},
    this.locked = false,
    this.correctOptionId,
    this.explanation,
    this.revealed = const {},
  });

  final int roundNo;
  final String prompt;
  final String? category;
  final List<AnswerOption> options;

  /// Server times (ms): show the question at [showAt], count down to
  /// [deadlineAt] (timing.md §4).
  final int showAt;
  final int deadlineAt;

  /// The option this player picked (confirmed by the server).
  final String? myOptionId;

  /// Players who answered (✓ only for opponents).
  final Set<String> answered;
  final bool locked;
  final String? correctOptionId;
  final String? explanation;
  final Map<String, RevealedAnswer> revealed;

  RoundState copyWith({
    String? myOptionId,
    Set<String>? answered,
    bool? locked,
    String? correctOptionId,
    String? explanation,
    Map<String, RevealedAnswer>? revealed,
    String? category,
  }) => RoundState(
    roundNo: roundNo,
    prompt: prompt,
    category: category ?? this.category,
    options: options,
    showAt: showAt,
    deadlineAt: deadlineAt,
    myOptionId: myOptionId ?? this.myOptionId,
    answered: answered ?? this.answered,
    locked: locked ?? this.locked,
    correctOptionId: correctOptionId ?? this.correctOptionId,
    explanation: explanation ?? this.explanation,
    revealed: revealed ?? this.revealed,
  );
}

@immutable
class MatchState {
  const MatchState({
    required this.matchId,
    required this.myUserId,
    this.phase = MatchPhase.waiting,
    this.questionCount = 0,
    this.readyDeadlineAt = 0,
    this.startsAt = 0,
    this.players = const [],
    this.scores = const [0, 0],
    this.round,
    this.endReason,
    this.cancelReason,
    this.winnerSide,
    this.isDraw = false,
    this.seatMoved = false,
    this.resumed = false,
  });

  final String matchId;
  final String myUserId;
  final MatchPhase phase;
  final int questionCount;
  final int readyDeadlineAt;
  final int startsAt;
  final List<Player> players;

  /// Side 1 and side 2 totals.
  final List<int> scores;
  final RoundState? round;
  final String? endReason;
  final String? cancelReason;
  final int? winnerSide;
  final bool isDraw;

  /// Another device took this player's seat (match.seat_moved).
  final bool seatMoved;

  /// The server took the match over and replayed a round (match.resumed).
  final bool resumed;

  Player? get me => players.where((p) => p.userId == myUserId).firstOrNull;
  Player? get opponent => players.where((p) => p.side != mySide).firstOrNull;
  int get mySide => me?.side ?? 1;
  int get myScore => scores[mySide - 1];
  int get opponentScore => scores[2 - mySide];
  bool get over =>
      phase == MatchPhase.completed || phase == MatchPhase.cancelled;

  MatchState copyWith({
    MatchPhase? phase,
    int? questionCount,
    int? readyDeadlineAt,
    int? startsAt,
    List<Player>? players,
    List<int>? scores,
    RoundState? round,
    bool clearRound = false,
    String? endReason,
    String? cancelReason,
    int? winnerSide,
    bool? isDraw,
    bool? seatMoved,
    bool? resumed,
  }) => MatchState(
    matchId: matchId,
    myUserId: myUserId,
    phase: phase ?? this.phase,
    questionCount: questionCount ?? this.questionCount,
    readyDeadlineAt: readyDeadlineAt ?? this.readyDeadlineAt,
    startsAt: startsAt ?? this.startsAt,
    players: players ?? this.players,
    scores: scores ?? this.scores,
    round: clearRound ? null : (round ?? this.round),
    endReason: endReason ?? this.endReason,
    cancelReason: cancelReason ?? this.cancelReason,
    winnerSide: winnerSide ?? this.winnerSide,
    isDraw: isDraw ?? this.isDraw,
    seatMoved: seatMoved ?? this.seatMoved,
    resumed: resumed ?? this.resumed,
  );
}
