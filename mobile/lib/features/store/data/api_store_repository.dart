import '../../../core/network/api_client.dart';
import '../domain/store_repository.dart';

/// [StoreRepository] over the REST API.
class ApiStoreRepository implements StoreRepository {
  const ApiStoreRepository(this.api);

  final ApiClient api;

  @override
  Future<Entitlements> entitlements() async =>
      Entitlements.fromJson(await api.getJson('/v1/me/entitlements'));
}
