import '../../../core/network/api_client.dart';
import '../domain/me.dart';
import '../domain/profile_repository.dart';

/// [ProfileRepository] over the REST API.
class ApiProfileRepository implements ProfileRepository {
  const ApiProfileRepository(this.api);

  final ApiClient api;

  @override
  Future<Me> me() async => Me.fromJson(await api.getJson('/v1/me'));

  @override
  Future<OwnProfile> updateProfile({
    String? displayName,
    String? username,
    String? avatarId,
    String? locale,
  }) async => OwnProfile.fromJson(
    await api.patchJson(
      '/v1/me/profile',
      body: {
        'display_name': ?displayName,
        'username': ?username,
        'avatar_id': ?avatarId,
        'locale': ?locale,
      },
    ),
  );

  @override
  Future<UserSettings> settings() async =>
      UserSettings.fromJson(await api.getJson('/v1/me/settings'));

  @override
  Future<UserSettings> updateSettings(UserSettings settings) async =>
      UserSettings.fromJson(
        await api.patchJson('/v1/me/settings', body: settings.toJson()),
      );
}
