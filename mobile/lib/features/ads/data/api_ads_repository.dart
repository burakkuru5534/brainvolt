import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/ads_repository.dart';

/// [AdsRepository] over the REST API.
class ApiAdsRepository implements AdsRepository {
  const ApiAdsRepository(this.api);

  final ApiClient api;

  @override
  Future<AdsConfig> config() async =>
      AdsConfig.fromJson(await api.getJson('/v1/ads/config'));

  @override
  Future<AdReward> reserve(String placement, {String? ref}) async =>
      AdReward.fromJson(
        (await api.postJson(
          '/v1/ads/rewards',
          body: {'placement': placement, 'ref': ?ref},
          idempotencyKey: const Uuid().v7(),
        ))!,
      );

  @override
  Future<AdReward> get(String id) async =>
      AdReward.fromJson(await api.getJson('/v1/ads/rewards/$id'));

  @override
  Future<AdReward> devGrant(String id) async =>
      AdReward.fromJson((await api.postJson('/v1/ads/rewards/$id/dev-grant'))!);
}
