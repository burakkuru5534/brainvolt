import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/progress_repository.dart';

/// [ProgressRepository] over the REST API.
class ApiProgressRepository implements ProgressRepository {
  const ApiProgressRepository(this.api);

  final ApiClient api;

  @override
  Future<Progress> progress() async =>
      Progress.fromJson(await api.getJson('/v1/me/progress'));

  @override
  Future<Standing> standing(String kind, String id) async =>
      Standing.fromJson(await api.getJson('/v1/standings/$kind/$id'));

  @override
  Future<QuestList> quests() async =>
      QuestList.fromJson(await api.getJson('/v1/me/quests'));

  @override
  Future<int> claim(String questId) async {
    final j = await api.postJson(
      '/v1/me/quests/$questId/claim',
      idempotencyKey: const Uuid().v7(),
    );
    return (j?['xp'] as num?)?.toInt() ?? 0;
  }

  @override
  Future<QuestList> reroll(String questId) async => QuestList.fromJson(
    (await api.postJson(
      '/v1/me/quests/$questId/reroll',
      idempotencyKey: const Uuid().v7(),
    ))!,
  );

  @override
  Future<List<BadgeInfo>> badges() async {
    final j = await api.getJson('/v1/badges');
    return [
      for (final b in (j['badges'] as List? ?? const []))
        BadgeInfo.fromJson((b as Map).cast<String, dynamic>()),
    ];
  }

  @override
  Future<Board> leaderboard({
    required BoardPeriod period,
    required BoardEntity entity,
    required BoardScope scope,
    String? categoryId,
    String? cursor,
  }) async {
    final q = Uri(
      queryParameters: {
        'period': period.wire,
        'entity': entity.name,
        'scope': scope.name,
        'category_id': ?categoryId,
        'cursor': ?cursor,
      },
    ).query;
    return Board.fromJson(await api.getJson('/v1/leaderboards?$q'));
  }

  @override
  Future<MatchAwards> awards(String matchId) async =>
      const MatchAwards().merge(await api.getJson('/v1/me/awards/$matchId'));
}
