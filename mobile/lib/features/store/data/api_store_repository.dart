import '../../../core/network/api_client.dart';
import '../domain/store_repository.dart';

/// [StoreRepository] over the REST API.
class ApiStoreRepository implements StoreRepository {
  const ApiStoreRepository(this.api);

  final ApiClient api;

  @override
  Future<Entitlements> entitlements() async =>
      Entitlements.fromJson(await api.getJson('/v1/me/entitlements'));

  @override
  Future<Wallet> wallet() async =>
      Wallet.fromJson(await api.getJson('/v1/me/wallet'));

  @override
  Future<int> buy(String item) async {
    final j = await api.postJson('/v1/store/items/$item/buy');
    return (j?['balance'] as num?)?.toInt() ?? 0;
  }

  @override
  Future<MyCosmetics> cosmetics() async =>
      MyCosmetics.fromJson(await api.getJson('/v1/store/cosmetics'));

  @override
  Future<int> buyCosmetic(String code) async {
    final j = await api.postJson('/v1/store/cosmetics/$code/buy');
    return (j?['balance'] as num?)?.toInt() ?? 0;
  }

  @override
  Future<MyCosmetics> equip(String kind, String? code) async =>
      MyCosmetics.fromJson(
        await api.putJson('/v1/me/cosmetics/$kind', body: {'code': code}),
      );
}
