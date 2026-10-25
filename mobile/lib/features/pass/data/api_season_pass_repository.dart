import '../../../core/network/api_client.dart';
import '../domain/season_pass.dart';

/// [SeasonPassRepository] over the REST API.
class ApiSeasonPassRepository implements SeasonPassRepository {
  const ApiSeasonPassRepository(this.api);

  final ApiClient api;

  @override
  Future<SeasonPass> pass() async =>
      SeasonPass.fromJson(await api.getJson('/v1/me/season-pass'));

  @override
  Future<SeasonPass> buyPremium() async => SeasonPass.fromJson(
    (await api.postJson('/v1/me/season-pass/premium')) ?? const {},
  );

  @override
  Future<PassClaim> claim() async => PassClaim.fromJson(
    (await api.postJson('/v1/me/season-pass/claim')) ?? const {},
  );
}
