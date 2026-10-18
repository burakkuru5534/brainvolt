import 'me.dart';

/// The profile endpoints of the backend (rest.md, `profile` module).
abstract interface class ProfileRepository {
  /// `GET /v1/me`.
  Future<Me> me();

  /// `PATCH /v1/me/profile` with the non-null fields.
  Future<OwnProfile> updateProfile({
    String? displayName,
    String? username,
    String? avatarId,
    String? locale,
  });

  /// `GET /v1/me/settings`.
  Future<UserSettings> settings();

  /// `PATCH /v1/me/settings` with every field of [settings].
  Future<UserSettings> updateSettings(UserSettings settings);
}
