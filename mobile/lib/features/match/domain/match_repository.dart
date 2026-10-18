/// REST side of playing (rest.md §4–§5).
library;

/// A queue ticket; [matchId] is set when a practice match started at once.
class QueueTicket {
  const QueueTicket({required this.id, required this.state, this.matchId});

  final String id;

  /// searching / found.
  final String state;
  final String? matchId;
}

/// One player's line on the result screen.
class ResultPlayer {
  const ResultPlayer({
    required this.userId,
    required this.side,
    required this.displayName,
    required this.avatarId,
    required this.correct,
    required this.status,
  });

  final String userId;
  final int side;
  final String displayName;
  final String avatarId;
  final int correct;
  final String status;
}

/// One answer on the result screen.
class ResultAnswer {
  const ResultAnswer({
    required this.userId,
    required this.optionId,
    required this.correct,
    required this.points,
  });

  final String userId;
  final String? optionId;
  final bool correct;
  final int points;
}

/// One played round on the result screen.
class ResultRound {
  const ResultRound({
    required this.roundNo,
    required this.prompt,
    required this.correctText,
    required this.answers,
    required this.options,
  });

  final int roundNo;
  final String prompt;
  final String correctText;
  final Map<String, String> options;
  final List<ResultAnswer> answers;
}

/// GET /v1/matches/{id}.
class MatchResult {
  const MatchResult({
    required this.id,
    required this.status,
    required this.scores,
    required this.players,
    required this.rounds,
    this.winnerSide,
    this.isDraw = false,
    this.endReason,
    this.cancelReason,
  });

  final String id;
  final String status;
  final List<int> scores;
  final List<ResultPlayer> players;
  final List<ResultRound> rounds;
  final int? winnerSide;
  final bool isDraw;
  final String? endReason;
  final String? cancelReason;
}

abstract interface class MatchRepository {
  /// Joins the Quick Duel queue, or starts a practice match (dev).
  Future<QueueTicket> enqueue({bool practice = false});

  /// Leaves the queue (harmless when not queued).
  Future<void> leaveQueue();

  /// The caller's live match id, if any.
  Future<String?> currentMatch();

  Future<MatchResult> result(String matchId);
}
