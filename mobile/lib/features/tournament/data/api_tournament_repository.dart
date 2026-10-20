import '../../../core/network/api_client.dart';
import '../domain/tournament_repository.dart';

class ApiTournamentRepository implements TournamentRepository {
  const ApiTournamentRepository(this.api);

  final ApiClient api;

  @override
  Future<List<Tournament>> list() async {
    final j = await api.getJson('/v1/tournaments');
    return [
      for (final t in (j['tournaments'] as List? ?? const []))
        Tournament.fromJson((t as Map).cast<String, dynamic>()),
    ];
  }

  @override
  Future<TournamentDetail> get(String id) async =>
      TournamentDetail.fromJson(await api.getJson('/v1/tournaments/$id'));

  @override
  Future<void> register(String id, {String? teamId, List<String>? lineup}) =>
      api.postJson(
        '/v1/tournaments/$id/entries',
        body: {'team_id': ?teamId, 'lineup': ?lineup},
      );

  @override
  Future<void> withdraw(String id) =>
      api.delete('/v1/tournaments/$id/entries/mine');

  @override
  Future<void> checkIn(String id) =>
      api.postJson('/v1/tournaments/$id/check-in');

  @override
  Future<String?> play(String id) async {
    final j = await api.postJson('/v1/tournaments/$id/play');
    return j?['match_id'] as String?;
  }

  @override
  Future<String> create(TournamentDraft d) async {
    final now = DateTime.now().toUtc();
    final j = await api.postJson(
      '/v1/admin/tournaments',
      body: {
        'texts': {
          'tr': {'name': d.nameTr, 'description': ?d.description},
          if (d.nameEn case final en? when en.isNotEmpty)
            'en': {'name': en, 'description': ?d.description},
        },
        'entity_kind': d.entityKind,
        'match_config': {'match_type': d.matchType, 'format': d.format},
        'min_entrants': 2,
        'max_entrants': d.maxEntrants,
        'registration_opens_at': now
            .subtract(const Duration(minutes: 1))
            .toIso8601String(),
        'registration_closes_at': d.startsAt.toUtc().toIso8601String(),
        'starts_at': d.startsAt.toUtc().toIso8601String(),
        'round_ready_window_s': d.readyWindowS,
        'rewards': {
          'places': [
            for (final (i, xp) in d.rewards.indexed) {'place': i + 1, 'xp': xp},
          ],
          'participation_xp': d.participationXp,
        },
        'publish': d.publish,
      },
    );
    return j!['id'] as String;
  }
}
