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
  });

  final String id;

  /// direct / rematch / revenge.
  final String kind;

  /// pending / declined / cancelled / expired / started / failed.
  final String status;
  final UserCard challenger;
  final UserCard target;
  final String matchType;
  final DateTime expiresAt;
  final String? matchId;
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
  });

  final String id;

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

  bool get full => members.length >= capacity;
}

/// Challenge actions.
enum ChallengeAction { accept, decline, cancel }

abstract interface class LobbyRepository {
  Future<FriendCode> friendCode();

  Future<ResolvedCode> resolveCode(String code);

  Future<Challenge> challenge(String targetUserId, MatchType type);

  Future<List<Challenge>> challenges({required bool incoming});

  Future<Challenge> respondChallenge(String id, ChallengeAction action);

  /// Revenge to the winners of [matchId] with a home category (D-045).
  Future<Challenge> revenge(String matchId, String homeCategoryId);

  /// Creates a room hosted by the caller (who leaves a previous room).
  Future<Room> createRoom(
    MatchType type, {
    String? categoryMode,
    String format = '1v1',
    String? teamMode,
  });

  /// Joins the room of an invite code.
  Future<Room> joinRoom(String code);

  Future<Room> room(String id);

  Future<void> leaveRoom(String id);
}
