import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/daily_repository.dart';

/// [DailyRepository] over the REST API.
class ApiDailyRepository implements DailyRepository {
  const ApiDailyRepository(this.api);

  final ApiClient api;

  @override
  Future<DailyToday> today() async =>
      DailyToday.fromJson(await api.getJson('/v1/daily/today'));

  @override
  Future<DailyRun> start(String kind) async => DailyRun.fromJson(
    (await api.postJson(
      '/v1/daily/runs',
      body: {'kind': kind},
      idempotencyKey: const Uuid().v7(),
    ))!,
  );

  @override
  Future<DailyBoard> board(String day, String scope) async =>
      DailyBoard.fromJson(
        await api.getJson('/v1/leaderboards/daily-challenge/$day?scope=$scope'),
      );
}
