import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/social_repository.dart';

List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

/// [SocialRepository] over the REST API.
class ApiSocialRepository implements SocialRepository {
  const ApiSocialRepository(this.api);

  final ApiClient api;

  @override
  Future<List<Friend>> friends() async {
    final j = await api.getJson('/v1/friends');
    return [
      for (final f in _maps(j['friends']))
        Friend(UserCard.fromJson(f), online: f['online'] as bool?),
    ];
  }

  @override
  Future<List<FriendRequest>> requests({required bool incoming}) async {
    final j = await api.getJson(
      '/v1/friend-requests?box=${incoming ? 'in' : 'out'}',
    );
    return [
      for (final r in _maps(j['requests']))
        FriendRequest(
          id: r['id'] as String,
          other: UserCard.fromJson((r['user'] as Map).cast<String, dynamic>()),
          incoming: incoming,
        ),
    ];
  }

  @override
  Future<bool> sendRequest({
    String? userId,
    String? username,
    String via = 'search',
  }) async {
    final j = await api.postJson(
      '/v1/friend-requests',
      body: {'to_user_id': ?userId, 'username': ?username, 'via': via},
      idempotencyKey: const Uuid().v7(),
    );
    return j?['accepted'] == true;
  }

  @override
  Future<void> respond(String requestId, RequestAction action) =>
      api.postJson('/v1/friend-requests/$requestId/${action.name}');

  @override
  Future<void> unfriend(String userId) => api.delete('/v1/friends/$userId');

  @override
  Future<PlayerProfile> profile(String userId) async =>
      PlayerProfile.fromJson(await api.getJson('/v1/profiles/$userId'));

  @override
  Future<void> block(String userId) =>
      api.postJson('/v1/blocks', body: {'user_id': userId});
}
