/// Friends, friend requests and blocks (rest.md §7, schema-social §2).
library;

/// A user's public card.
class UserCard {
  const UserCard({
    required this.userId,
    required this.displayName,
    this.username = '',
    this.avatarId = '',
    this.frame = '',
    this.nameColor = '',
  });

  factory UserCard.fromJson(Map<String, dynamic> j) => UserCard(
    userId: j['user_id'] as String,
    username: j['username'] as String? ?? '',
    displayName: j['display_name'] as String? ?? '',
    avatarId: j['avatar_id'] as String? ?? '',
    frame: j['frame'] as String? ?? '',
    nameColor: j['name_color'] as String? ?? '',
  );

  final String userId;
  final String username;
  final String displayName;
  final String avatarId;

  /// Worn cosmetics (D-106); empty when none.
  final String frame;
  final String nameColor;
}

/// A friend; [online] is null when they hide their status.
class Friend {
  const Friend(this.card, {this.online});

  final UserCard card;
  final bool? online;
}

/// A pending friend request with the other user's card.
class FriendRequest {
  const FriendRequest({
    required this.id,
    required this.other,
    required this.incoming,
  });

  final String id;
  final UserCard other;
  final bool incoming;
}

/// The viewer's tie to a player (D-101).
enum Relation { self, friend, requestSent, requestReceived, blocked, none }

Relation _relation(String? v) => switch (v) {
  'self' => Relation.self,
  'friend' => Relation.friend,
  'request_sent' => Relation.requestSent,
  'request_received' => Relation.requestReceived,
  'blocked' => Relation.blocked,
  _ => Relation.none,
};

/// A player's public progress.
class PlayerStats {
  const PlayerStats({
    this.level = 1,
    this.matches = 0,
    this.wins = 0,
    this.badges = 0,
    this.streak = 0,
    this.trophies = 0,
  });

  factory PlayerStats.fromJson(Map<String, dynamic> j) => PlayerStats(
    level: (j['level'] as num?)?.toInt() ?? 1,
    matches: (j['matches'] as num?)?.toInt() ?? 0,
    wins: (j['wins'] as num?)?.toInt() ?? 0,
    badges: (j['badges'] as num?)?.toInt() ?? 0,
    streak: (j['streak'] as num?)?.toInt() ?? 0,
    trophies: (j['trophies'] as num?)?.toInt() ?? 0,
  );

  final int level;
  final int matches;
  final int wins;
  final int badges;
  final int streak;

  /// Friend tournaments won (D-109).
  final int trophies;

  /// Wins in percent of matches (0 without matches).
  int get winRate => matches == 0 ? 0 : (wins * 100 / matches).round();
}

/// GET /v1/profiles/{id}: a player as the viewer sees them (D-101).
/// [stats] is null when the profile is private and the viewer is not a
/// friend, or on a block.
class PlayerProfile {
  const PlayerProfile({
    required this.card,
    this.countryCode,
    this.private = false,
    this.relation = Relation.none,
    this.stats,
  });

  factory PlayerProfile.fromJson(Map<String, dynamic> j) => PlayerProfile(
    card: UserCard.fromJson(j),
    countryCode: j['country_code'] as String?,
    private: j['private'] == true,
    relation: _relation(j['relation'] as String?),
    stats: j['stats'] is Map
        ? PlayerStats.fromJson((j['stats'] as Map).cast<String, dynamic>())
        : null,
  );

  final UserCard card;
  final String? countryCode;
  final bool private;
  final Relation relation;
  final PlayerStats? stats;
}

/// Friend request actions.
enum RequestAction { accept, decline, cancel }

abstract interface class SocialRepository {
  Future<List<Friend>> friends();

  Future<List<FriendRequest>> requests({required bool incoming});

  /// Sends a request by user id or username; true when it made a
  /// friendship at once (the other user had asked too).
  Future<bool> sendRequest({
    String? userId,
    String? username,
    String via = 'search',
  });

  Future<void> respond(String requestId, RequestAction action);

  Future<void> unfriend(String userId);

  Future<void> block(String userId);

  /// A player's profile as the caller sees it (D-101).
  Future<PlayerProfile> profile(String userId);
}
