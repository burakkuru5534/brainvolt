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
  kind: j['kind'] as String? ?? 'direct',
  parentMatchId: j['parent_match_id'] as String?,
);

/// Parses a room of the API (REST and room.state).
Room roomFromJson(Map<String, dynamic> j) {
  final cfg = _map(j['match_config']);
  return Room(
    id: j['id'] as String,
    status: j['status'] as String,
    version: (j['version'] as num?)?.toInt() ?? 0,
    hostUserId: j['host_user_id'] as String,
    code: j['code'] as String? ?? '',
    matchType: cfg['match_type'] as String? ?? '',
    format: cfg['format'] as String? ?? '1v1',
    teamMode: cfg['team_answer_mode'] as String?,
    categoryMode: cfg['category_mode'] as String?,
    categoryIds: [
      for (final id in (cfg['category_ids'] as List?) ?? const []) '$id',
    ],
    capacity: (j['capacity'] as num?)?.toInt() ?? 2,
    matchId: j['match_id'] as String?,
    creatorQuizId: cfg['creator_quiz_id'] as String?,
    quizLength: (cfg['quiz_length'] as num?)?.toInt(),
    friendsOnly: j['friends_of_host_only'] == true,
    kind: j['kind'] as String? ?? '',
    leaderUserId: j['leader_user_id'] as String?,
    members: [
      for (final m in _maps(j['members']))
        RoomMember(
          card: UserCard.fromJson(m),
          side: (m['side_no'] as num).toInt(),
          seat: (m['seat_no'] as num).toInt(),
          ready: m['ready'] == true,
          isHost: m['is_host'] == true,
        ),
    ],
  );
}

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

  @override
  Future<Room> createRoom(
    MatchType type, {
    String? categoryMode,
    String format = '1v1',
    String? teamMode,
    String? creatorQuizId,
    List<String> categoryIds = const [],
  }) async {
    final j = await api.postJson(
      '/v1/rooms',
      body: {
        'match_config': {
          'match_type': type.wire,
          'format': format,
          'team_answer_mode': ?teamMode,
          'category_mode': ?(creatorQuizId != null
              ? 'creator_quiz'
              : categoryMode),
          'creator_quiz_id': ?creatorQuizId,
          if (creatorQuizId == null && categoryIds.isNotEmpty)
            'category_ids': categoryIds,
        },
      },
      idempotencyKey: const Uuid().v7(),
    );
    return roomFromJson(j!);
  }

  @override
  Future<Room> joinRoom(String code) async {
    final j = await api.postJson('/v1/rooms/join', body: {'code': code.trim()});
    return roomFromJson(j!);
  }

  @override
  Future<Room> updateRoom(
    String id,
    RoomSettings settings, {
    String? creatorQuizId,
    int? version,
  }) async => roomFromJson(
    await api.patchJson(
      '/v1/rooms/$id',
      body: {
        'match_config': {
          'match_type': settings.matchType.wire,
          'format': settings.format,
          'team_answer_mode': ?(settings.format == '1v1'
              ? null
              : settings.teamMode),
          'category_mode': ?(creatorQuizId != null
              ? 'creator_quiz'
              : settings.categoryMode),
          'creator_quiz_id': ?creatorQuizId,
          if (creatorQuizId == null && settings.categoryIds.isNotEmpty)
            'category_ids': settings.categoryIds,
        },
      },
      ifMatch: version?.toString(),
    ),
  );

  @override
  Future<Room> room(String id) async =>
      roomFromJson(await api.getJson('/v1/rooms/$id'));

  @override
  Future<void> leaveRoom(String id) => api.postJson('/v1/rooms/$id/leave');
}
