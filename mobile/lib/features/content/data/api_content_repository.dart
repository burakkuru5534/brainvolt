import '../../../core/network/api_client.dart';
import '../domain/content_repository.dart';
import '../domain/credit.dart';

/// [ContentRepository] over the REST API.
class ApiContentRepository implements ContentRepository {
  const ApiContentRepository(this.api);

  final ApiClient api;

  @override
  Future<List<Credit>> credits() async {
    final json = await api.getJson('/v1/credits');
    return [
      for (final s in json['sources'] as List<dynamic>)
        Credit.fromJson(s as Map<String, dynamic>),
    ];
  }
}
