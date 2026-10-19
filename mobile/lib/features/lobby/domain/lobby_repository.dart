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
  });

  final String id;

  /// pending / declined / cancelled / expired / started / failed.
  final String status;
  final UserCard challenger;
  final UserCard target;
  final String matchType;
  final DateTime expiresAt;
  final String? matchId;
}

/// Challenge actions.
enum ChallengeAction { accept, decline, cancel }

abstract interface class LobbyRepository {
  Future<FriendCode> friendCode();

  Future<ResolvedCode> resolveCode(String code);

  Future<Challenge> challenge(String targetUserId, MatchType type);

  Future<List<Challenge>> challenges({required bool incoming});

  Future<Challenge> respondChallenge(String id, ChallengeAction action);
}
