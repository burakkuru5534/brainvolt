/// Ranking and progression reads and actions (rest.md §9–10).
library;

import '../../social/domain/social_repository.dart';

int _int(Object? v) => (v as num?)?.toInt() ?? 0;

DateTime? _time(Object? v) => v is String ? DateTime.parse(v) : null;

List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

/// XP, level and play streak (GET /v1/me/progress).
class Progress {
  const Progress({
    required this.xpTotal,
    required this.level,
    required this.levelXp,
    required this.levelSize,
    required this.streak,
    required this.longestStreak,
    required this.playedToday,
    required this.badges,
    this.matches = 0,
    this.wins = 0,
    this.shields = 0,
    this.maxShields = 2,
  });

  factory Progress.fromJson(Map<String, dynamic> j) {
    final s = (j['streak'] as Map?)?.cast<String, dynamic>() ?? const {};
    return Progress(
      xpTotal: _int(j['xp_total']),
      level: _int(j['level']),
      levelXp: _int(j['level_xp']),
      levelSize: _int(j['level_size']),
      streak: _int(s['current']),
      longestStreak: _int(s['longest']),
      playedToday: s['played_today'] == true,
      badges: _int(j['badges']),
      matches: _int(j['matches']),
      wins: _int(j['wins']),
      shields: _int(s['shields']),
      maxShields: (s['max_shields'] as num?)?.toInt() ?? 2,
    );
  }

  final int xpTotal;
  final int level;
  final int levelXp;
  final int levelSize;
  final int streak;
  final int longestStreak;
  final bool playedToday;
  final int badges;

  /// Every completed match and win.
  final int matches;
  final int wins;

  /// Streak shields held (D-105); each covers a missed day.
  final int shields;
  final int maxShields;
}

/// A season standing (GET /v1/standings/{kind}/{id}).
class Standing {
  const Standing({
    required this.seasonPoints,
    required this.league,
    required this.division,
    required this.matches,
    required this.wins,
    this.nextAt,
    this.rank,
  });

  factory Standing.fromJson(Map<String, dynamic> j) => Standing(
    seasonPoints: _int(j['season_points']),
    league: j['league'] as String? ?? 'bronze',
    division: _int(j['division']),
    matches: _int(j['matches']),
    wins: _int(j['wins']),
    nextAt: (j['next_at'] as num?)?.toInt(),
    rank: (j['rank'] as num?)?.toInt(),
  );

  final int seasonPoints;
  final String league;
  final int division;
  final int matches;
  final int wins;
  final int? nextAt;
  final int? rank;
}

/// An assigned quest.
class Quest {
  const Quest({
    required this.id,
    required this.cadence,
    required this.title,
    required this.progress,
    required this.target,
    required this.rewardXp,
    required this.endsAt,
    this.rewardVolt = 0,
    this.completedAt,
    this.claimedAt,
  });

  factory Quest.fromJson(Map<String, dynamic> j) => Quest(
    id: j['id'] as String,
    cadence: j['cadence'] as String? ?? 'daily',
    title: j['title'] as String? ?? '',
    progress: _int(j['progress']),
    target: _int(j['target']),
    rewardXp: _int(j['reward_xp']),
    rewardVolt: _int(j['reward_volt']),
    endsAt: _time(j['ends_at']) ?? DateTime.now(),
    completedAt: _time(j['completed_at']),
    claimedAt: _time(j['claimed_at']),
  );

  final String id;
  final String cadence;
  final String title;
  final int progress;
  final int target;
  final int rewardXp;
  final int rewardVolt;
  final DateTime endsAt;
  final DateTime? completedAt;
  final DateTime? claimedAt;

  bool get completed => completedAt != null;
  bool get claimed => claimedAt != null;
}

class QuestList {
  const QuestList({
    required this.daily,
    required this.weekly,
    required this.rerollsLeft,
  });

  factory QuestList.fromJson(Map<String, dynamic> j) => QuestList(
    daily: [for (final q in _maps(j['daily'])) Quest.fromJson(q)],
    weekly: [for (final q in _maps(j['weekly'])) Quest.fromJson(q)],
    rerollsLeft: _int(j['rerolls_left']),
  );

  final List<Quest> daily;
  final List<Quest> weekly;
  final int rerollsLeft;
}

/// A badge of the catalogue; [earnedAt] is null when not earned.
class BadgeInfo {
  const BadgeInfo({
    required this.code,
    required this.family,
    required this.tier,
    required this.name,
    required this.description,
    this.earnedAt,
  });

  factory BadgeInfo.fromJson(Map<String, dynamic> j) => BadgeInfo(
    code: j['code'] as String,
    family: j['family'] as String? ?? '',
    tier: _int(j['tier']),
    name: j['name'] as String? ?? '',
    description: j['description'] as String? ?? '',
    earnedAt: _time(j['earned_at']),
  );

  final String code;
  final String family;
  final int tier;
  final String name;
  final String description;
  final DateTime? earnedAt;
}

/// Leaderboard filters (schema-ranking §4.1).
enum BoardPeriod { daily, weekly, monthly, allTime, season }

enum BoardEntity { user, team, couple }

enum BoardScope { global, country, friends }

extension BoardPeriodWire on BoardPeriod {
  String get wire => this == BoardPeriod.allTime ? 'all_time' : name;
}

/// One leaderboard row: a user, a team or a couple.
class BoardEntry {
  const BoardEntry({
    required this.rank,
    required this.entityId,
    required this.score,
    required this.name,
    this.avatarId = '',
    this.emblemId = '',
    this.user,
  });

  factory BoardEntry.fromJson(Map<String, dynamic> j) {
    final user = (j['user'] as Map?)?.cast<String, dynamic>();
    final team = (j['team'] as Map?)?.cast<String, dynamic>();
    final couple = (j['couple'] as Map?)?.cast<String, dynamic>();
    final partners = [
      for (final p in _maps(couple?['partners'])) UserCard.fromJson(p),
    ];
    final coupleName = couple?['name'] as String? ?? '';
    return BoardEntry(
      rank: _int(j['rank']),
      entityId: j['entity_id'] as String,
      score: _int(j['score']),
      user: user == null ? null : UserCard.fromJson(user),
      name: switch ((user, team, couple)) {
        (final u?, _, _) => u['display_name'] as String? ?? '',
        (_, final t?, _) => '${t['name']} [${t['tag']}]',
        (_, _, _?) when coupleName.isNotEmpty => coupleName,
        _ => partners.map((p) => p.displayName).join(' & '),
      },
      avatarId: user?['avatar_id'] as String? ?? '',
      emblemId: team?['emblem_id'] as String? ?? '',
    );
  }

  final int rank;
  final String entityId;
  final int score;
  final String name;
  final String avatarId;
  final String emblemId;
  final UserCard? user;
}

class Board {
  const Board({required this.entries, this.me, this.nextCursor});

  factory Board.fromJson(Map<String, dynamic> j) => Board(
    entries: [for (final e in _maps(j['entries'])) BoardEntry.fromJson(e)],
    me: j['me'] is Map
        ? BoardEntry.fromJson((j['me'] as Map).cast<String, dynamic>())
        : null,
    nextCursor: j['next_cursor'] as String?,
  );

  final List<BoardEntry> entries;
  final BoardEntry? me;
  final String? nextCursor;
}

/// The awards of a finished match, merged from the two match.awards
/// messages (D-077) or read back from GET /v1/me/awards/{id}.
class MatchAwards {
  const MatchAwards({
    this.xp,
    this.level,
    this.levelUp = false,
    this.streak,
    this.badges = const [],
    this.questsCompleted = 0,
    this.spDelta,
    this.league,
    this.division,
    this.leagueUp = false,
  });

  final int? xp;
  final int? level;
  final bool levelUp;
  final int? streak;
  final List<String> badges;
  final int questsCompleted;
  final int? spDelta;
  final String? league;
  final int? division;
  final bool leagueUp;

  bool get isEmpty => xp == null && spDelta == null;

  /// Adds one match.awards message (or the REST answer).
  MatchAwards merge(Map<String, dynamic> d) {
    final progression = d.containsKey('xp');
    final ranking = d.containsKey('sp_delta');
    return MatchAwards(
      xp: progression ? _int(d['xp']) : xp,
      level: progression ? _int(d['level']) : level,
      levelUp: progression ? d['level_up'] == true : levelUp,
      streak: progression ? _int(d['streak']) : streak,
      badges: progression
          ? [
              for (final b in _maps(d['badges']))
                (b['name'] as String?)?.isNotEmpty ?? false
                    ? b['name'] as String
                    : b['code'] as String,
            ]
          : badges,
      questsCompleted: progression
          ? _maps(d['quests']).where((q) => q['completed'] == true).length
          : questsCompleted,
      spDelta: ranking ? _int(d['sp_delta']) : spDelta,
      league: ranking ? d['league'] as String? : league,
      division: ranking ? _int(d['division']) : division,
      leagueUp: ranking ? d['league_up'] == true : leagueUp,
    );
  }
}

abstract interface class ProgressRepository {
  Future<Progress> progress();

  Future<Standing> standing(String kind, String id);

  Future<QuestList> quests();

  /// Returns the XP granted.
  Future<int> claim(String questId);

  Future<QuestList> reroll(String questId);

  Future<List<BadgeInfo>> badges();

  Future<Board> leaderboard({
    required BoardPeriod period,
    required BoardEntity entity,
    required BoardScope scope,
    String? categoryId,
    String? cursor,
  });

  Future<MatchAwards> awards(String matchId);
}
