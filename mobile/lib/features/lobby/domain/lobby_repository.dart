/// Invite codes and direct challenges (rest.md §6, schema-social §3).
library;

import '../../social/domain/social_repository.dart';

/// Matchmade and challenge match types the app offers.
enum MatchType {
  quickDuel('quick_duel'),
  suddenDeath('sudden_death'),
  marathon('marathon');

  const MatchType(this.wire);
  final String wire;

  static MatchType? parse(String? v) =>
      values.where((t) => t.wire == v).firstOrNull;
}

/// The caller's friend invite code.
class FriendCode {
  const FriendCode(this.code, this.expiresAt);
  final String code;
  final DateTime expiresAt;
}

/// What an invite code opens: a friend's card or a room.
class ResolvedCode {
  const ResolvedCode({required this.kind, this.user, this.roomId});
  final String kind;
  final UserCard? user;
  final String? roomId;
}

/// A challenge with both players' cards.
class Challenge {
  const Challenge({
    required this.id,
    required this.status,
    required this.challenger,
    required this.target,
    required this.matchType,
    required this.expiresAt,
    this.matchId,
    this.kind = 'direct',
    this.parentMatchId,
  });

  final String id;

  /// direct / rematch (revenge only on old rows, D-091).
  final String kind;

  /// pending / declined / cancelled / expired / started / failed.
  final String status;
  final UserCard challenger;
  final UserCard target;
  final String matchType;
  final DateTime expiresAt;
  final String? matchId;

  /// The match an old revenge answered (D-091 removed revenge).
  final String? parentMatchId;
}

/// A member of a private room.
class RoomMember {
  const RoomMember({
    required this.card,
    required this.side,
    required this.seat,
    required this.ready,
    required this.isHost,
  });

  final UserCard card;
  final int side;
  final int seat;
  final bool ready;
  final bool isHost;
}

/// A private room (rest.md §6, websocket.md room.state).
class Room {
  const Room({
    required this.id,
    required this.status,
    required this.version,
    required this.hostUserId,
    required this.code,
    required this.matchType,
    required this.capacity,
    required this.members,
    this.format = '1v1',
    this.teamMode,
    this.categoryMode,
    this.matchId,
    this.creatorQuizId,
    this.quizLength,
    this.friendsOnly = false,
    this.kind = '',
    this.leaderUserId,
  });

  final String id;

  /// '' for a private room, 'matchmaking' for the waiting lobby of a
  /// matchmade team match (D-094).
  final String kind;

  /// The matchmaking lobby's leader, drawn once every seat is taken.
  final String? leaderUserId;

  bool get isMatchmaking => kind == 'matchmaking';

  /// open / starting / in_match.
  final String status;
  final int version;
  final String hostUserId;
  final String code;
  final String matchType;
  final String format;
  final String? teamMode;
  final String? categoryMode;
  final int capacity;
  final List<RoomMember> members;
  final String? matchId;

  /// The Creator Quiz the room plays, with its length.
  final String? creatorQuizId;
  final int? quizLength;

  /// Only the host's friends may join (an unpublished quiz, D-023).
  final bool friendsOnly;

  bool get full => members.length >= capacity;
}

/// What the host picks for a room (D-093): every room mode can be set at
/// creation and changed between matches.
class RoomSettings {
  const RoomSettings({
    this.matchType = MatchType.quickDuel,
    this.format = '1v1',
    this.teamMode,
    this.categoryMode,
  });

  /// The settings a room plays now.
  factory RoomSettings.of(Room r) => RoomSettings(
    matchType: MatchType.parse(r.matchType) ?? MatchType.quickDuel,
    format: r.format,
    teamMode: r.teamMode,
    categoryMode: r.categoryMode,
  );

  final MatchType matchType;

  /// 1v1 / 2v2 / 3v3 / 4v4 / couple.
  final String format;

  /// Team formats only (fast_team_round / shared_answer /
  /// individual_turns).
  final String? teamMode;
  final String? categoryMode;
}

/// Challenge actions.
enum ChallengeAction { accept, decline, cancel }

abstract interface class LobbyRepository {
  Future<FriendCode> friendCode();

  Future<ResolvedCode> resolveCode(String code);

  Future<Challenge> challenge(String targetUserId, MatchType type);

  Future<List<Challenge>> challenges({required bool incoming});

  Future<Challenge> respondChallenge(String id, ChallengeAction action);

  /// Creates a room hosted by the caller (who leaves a previous room);
  /// [creatorQuizId] plays a Creator Quiz (Quick Duel).
  Future<Room> createRoom(
    MatchType type, {
    String? categoryMode,
    String format = '1v1',
    String? teamMode,
    String? creatorQuizId,
  });

  /// Changes a room's settings (host, room open, D-093); [creatorQuizId]
  /// keeps the room's Creator Quiz. [version] guards against a change
  /// made meanwhile (If-Match).
  Future<Room> updateRoom(
    String id,
    RoomSettings settings, {
    String? creatorQuizId,
    int? version,
  });

  /// Joins the room of an invite code.
  Future<Room> joinRoom(String code);

  Future<Room> room(String id);

  Future<void> leaveRoom(String id);
}
