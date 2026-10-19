/// The Daily Challenge (rest.md §4, category-modes §5).
library;

import '../../social/domain/social_repository.dart';

int _int(Object? v) => (v as num?)?.toInt() ?? 0;

/// A daily run: official (once a day) or practice.
class DailyRun {
  const DailyRun({
    required this.id,
    required this.kind,
    required this.status,
    this.correct,
    this.totalTimeMs,
    this.score,
  });

  factory DailyRun.fromJson(Map<String, dynamic> j) => DailyRun(
    id: j['id'] as String,
    kind: j['kind'] as String? ?? 'official',
    status: j['status'] as String? ?? 'in_progress',
    correct: (j['correct_count'] as num?)?.toInt(),
    totalTimeMs: (j['total_time_ms'] as num?)?.toInt(),
    score: (j['score'] as num?)?.toInt(),
  );

  final String id;
  final String kind;

  /// in_progress / completed / void_not_started / void_server_error.
  final String status;
  final int? correct;
  final int? totalTimeMs;
  final int? score;

  bool get inProgress => status == 'in_progress';
  bool get completed => status == 'completed';
}

/// Today's Daily Challenge for the caller.
class DailyToday {
  const DailyToday({
    required this.day,
    required this.available,
    required this.practiceAllowed,
    this.official,
  });

  factory DailyToday.fromJson(Map<String, dynamic> j) => DailyToday(
    day: j['day'] as String,
    available: j['available'] == true,
    practiceAllowed: j['practice_allowed'] == true,
    official: j['official'] is Map
        ? DailyRun.fromJson((j['official'] as Map).cast<String, dynamic>())
        : null,
  );

  /// yyyy-mm-dd, Europe/Istanbul.
  final String day;
  final bool available;
  final bool practiceAllowed;
  final DailyRun? official;
}

/// One place on a day's board.
class DailyEntry {
  const DailyEntry({
    required this.rank,
    required this.userId,
    required this.correct,
    required this.totalTimeMs,
    this.user,
  });

  factory DailyEntry.fromJson(Map<String, dynamic> j) => DailyEntry(
    rank: _int(j['rank']),
    userId: j['user_id'] as String,
    correct: _int(j['correct_count']),
    totalTimeMs: _int(j['total_time_ms']),
    user: j['user'] is Map
        ? UserCard.fromJson((j['user'] as Map).cast<String, dynamic>())
        : null,
  );

  final int rank;
  final String userId;
  final int correct;
  final int totalTimeMs;
  final UserCard? user;
}

class DailyBoard {
  const DailyBoard({required this.entries, this.me});

  factory DailyBoard.fromJson(Map<String, dynamic> j) => DailyBoard(
    entries: [
      for (final e in (j['entries'] as List? ?? const []))
        DailyEntry.fromJson((e as Map).cast<String, dynamic>()),
    ],
    me: j['me'] is Map
        ? DailyEntry.fromJson((j['me'] as Map).cast<String, dynamic>())
        : null,
  );

  final List<DailyEntry> entries;
  final DailyEntry? me;
}

abstract interface class DailyRepository {
  Future<DailyToday> today();

  /// official / practice.
  Future<DailyRun> start(String kind);

  /// global / country / friends.
  Future<DailyBoard> board(String day, String scope);
}
