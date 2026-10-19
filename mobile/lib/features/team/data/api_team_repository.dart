import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../../social/domain/social_repository.dart';
import '../domain/team_repository.dart';

List<Map<String, dynamic>> _maps(Object? v) => [
  for (final x in (v as List?) ?? const []) (x as Map).cast<String, dynamic>(),
];

Team _team(Map<String, dynamic> j) => Team(
  id: j['id'] as String,
  name: j['name'] as String,
  tag: j['tag'] as String,
  emblemId: j['emblem_id'] as String,
  captainUserId: j['captain_user_id'] as String,
  members: [
    for (final m in _maps(j['members']))
      TeamMember(UserCard.fromJson(m), captain: m['role'] == 'captain'),
  ],
);

Couple _couple(Map<String, dynamic> j) => Couple(
  id: j['id'] as String,
  status: j['status'] as String,
  requestedBy: j['requested_by'] as String,
  name: j['name'] as String?,
  partners: [for (final p in _maps(j['partners'])) UserCard.fromJson(p)],
);

/// [TeamRepository] over the REST API.
class ApiTeamRepository implements TeamRepository {
  const ApiTeamRepository(this.api);

  final ApiClient api;

  @override
  Future<List<Team>> myTeams() async {
    final j = await api.getJson('/v1/me/teams');
    return [for (final t in _maps(j['teams'])) _team(t)];
  }

  @override
  Future<Team> team(String id) async =>
      _team(await api.getJson('/v1/teams/$id'));

  @override
  Future<Team> createTeam({
    required String name,
    required String tag,
    required String emblemId,
  }) async => _team(
    (await api.postJson(
      '/v1/teams',
      body: {'name': name, 'tag': tag, 'emblem_id': emblemId},
      idempotencyKey: const Uuid().v7(),
    ))!,
  );

  @override
  Future<void> invite(String teamId, String userId) => api.postJson(
    '/v1/teams/$teamId/invites',
    body: {'user_id': userId},
    idempotencyKey: const Uuid().v7(),
  );

  @override
  Future<List<TeamInvite>> invites() async {
    final j = await api.getJson('/v1/me/team-invites');
    return [
      for (final i in _maps(j['invites']))
        TeamInvite(
          id: i['id'] as String,
          teamName: i['team_name'] as String,
          teamTag: i['team_tag'] as String,
          emblemId: i['emblem_id'] as String,
          invitedBy: UserCard.fromJson(
            (i['invited_by'] as Map).cast<String, dynamic>(),
          ),
        ),
    ];
  }

  @override
  Future<void> respondInvite(String inviteId, {required bool accept}) => api
      .postJson('/v1/team-invites/$inviteId/${accept ? 'accept' : 'decline'}');

  @override
  Future<void> teamAction(String teamId, TeamAction action) =>
      api.postJson('/v1/teams/$teamId/${action.name}');

  @override
  Future<void> kick(String teamId, String userId) =>
      api.postJson('/v1/teams/$teamId/kick/$userId');

  @override
  Future<void> makeCaptain(String teamId, String userId) =>
      api.postJson('/v1/teams/$teamId/captain/$userId');

  @override
  Future<Couple?> myCouple() async {
    final j = await api.getJson('/v1/me/couple');
    final c = j['couple'];
    return c is Map ? _couple(c.cast<String, dynamic>()) : null;
  }

  @override
  Future<Couple> requestCouple(String partnerUserId) async => _couple(
    (await api.postJson(
      '/v1/couples',
      body: {'partner_user_id': partnerUserId},
      idempotencyKey: const Uuid().v7(),
    ))!,
  );

  @override
  Future<Couple> respondCouple(String coupleId, String action) async =>
      _couple((await api.postJson('/v1/couples/$coupleId/$action'))!);
}
