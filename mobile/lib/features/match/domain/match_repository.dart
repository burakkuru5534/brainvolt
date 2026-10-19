/// REST side of playing (rest.md §4–§5).
library;

/// What to queue for (rest.md §5): a match type and an optional category
/// mode (category_master with a top-level [categoryId]).
class QueueMode {
  const QueueMode({
    this.matchType = 'quick_duel',
    this.categoryMode,
    this.categoryId,
    this.format = '1v1',
    this.partyUserIds = const [],
    this.teamId,
  });

  final String matchType;
  final String? categoryMode;
  final String? categoryId;

  /// 1v1 / 2v2 / 3v3 / 4v4 / couple (the player's couple).
  final String format;

  /// Friends (or, with [teamId], teammates) on the player's side.
  final List<String> partyUserIds;
  final String? teamId;

  bool get isDefault =>
      matchType == 'quick_duel' && categoryMode == null && format == '1v1';
}

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
    this.questionId = '',
  });

  final int roundNo;

  /// The question (reports, schema-moderation).
  final String questionId;
  final String prompt;
  final String correctText;
  final Map<String, String> options;
  final List<ResultAnswer> answers;
}

/// GET /v1/matches/{id}.
/// The caller's revenge state on a finished match (D-045).
class RevengeInfo {
  const RevengeInfo({
    required this.available,
    this.wins = 0,
    this.losses = 0,
    this.chain = 0,
  });

  final bool available;
  final int wins;
  final int losses;

  /// Revenge matches so far in this series (0 for the first match).
  final int chain;
}

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
    this.origin = '',
    this.endedAt,
    this.revenge,
  });

  final String id;
  final String status;

  /// matchmaking / challenge / private_room / rematch / …
  final String origin;
  final DateTime? endedAt;
  final List<int> scores;
  final List<ResultPlayer> players;
  final List<ResultRound> rounds;
  final int? winnerSide;
  final bool isDraw;
  final String? endReason;
  final String? cancelReason;
  final RevengeInfo? revenge;
}

abstract interface class MatchRepository {
  /// Joins a queue (Quick Duel by default), or starts a practice match
  /// (dev).
  Future<QueueTicket> enqueue({
    bool practice = false,
    QueueMode mode = const QueueMode(),
  });

  /// Leaves the queue (harmless when not queued).
  Future<void> leaveQueue();

  /// The caller's live match id, if any.
  Future<String?> currentMatch();

  Future<MatchResult> result(String matchId);
}
