import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../../social/domain/social_repository.dart';
import '../domain/lobby_repository.dart';

List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

Map<String, dynamic> _map(Object? v) =>
    (v as Map?)?.cast<String, dynamic>() ?? const {};

/// Parses a challenge of the API.
Challenge challengeFromJson(Map<String, dynamic> j) => Challenge(
  id: j['id'] as String,
  status: j['status'] as String,
  challenger: UserCard.fromJson(_map(j['challenger'])),
  target: UserCard.fromJson(_map(j['target'])),
  matchType: _map(j['match_config'])['match_type'] as String? ?? '',
  expiresAt: DateTime.parse(j['expires_at'] as String),
  matchId: j['match_id'] as String?,
);

/// [LobbyRepository] over the REST API.
class ApiLobbyRepository implements LobbyRepository {
  const ApiLobbyRepository(this.api);

  final ApiClient api;

  @override
  Future<FriendCode> friendCode() async {
    final j = await api.postJson(
      '/v1/invite-codes',
      body: {'kind': 'friend'},
      idempotencyKey: const Uuid().v7(),
    );
    return FriendCode(
      j!['code'] as String,
      DateTime.parse(j['expires_at'] as String),
    );
  }

  @override
  Future<ResolvedCode> resolveCode(String code) async {
    final j = await api.getJson(
      '/v1/invite-codes/${Uri.encodeComponent(code.trim())}',
    );
    return ResolvedCode(
      kind: j['kind'] as String,
      user: j['user'] is Map ? UserCard.fromJson(_map(j['user'])) : null,
      roomId: _map(j['room'])['id'] as String?,
    );
  }

  @override
  Future<Challenge> challenge(String targetUserId, MatchType type) async {
    final j = await api.postJson(
      '/v1/challenges',
      body: {
        'target_user_id': targetUserId,
        'match_config': {'match_type': type.wire, 'format': '1v1'},
      },
      idempotencyKey: const Uuid().v7(),
    );
    return challengeFromJson(j!);
  }

  @override
  Future<List<Challenge>> challenges({required bool incoming}) async {
    final j = await api.getJson(
      '/v1/challenges?box=${incoming ? 'in' : 'out'}',
    );
    return [for (final c in _maps(j['challenges'])) challengeFromJson(c)];
  }

  @override
  Future<Challenge> respondChallenge(String id, ChallengeAction action) async {
    final j = await api.postJson('/v1/challenges/$id/${action.name}');
    return challengeFromJson(j!);
  }
}
