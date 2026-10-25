import 'package:freezed_annotation/freezed_annotation.dart';

import '../../auth/domain/session.dart';

part 'me.freezed.dart';
part 'me.g.dart';

/// `GET /v1/me`.
@freezed
abstract class Me with _$Me {
  const factory Me({
    required MeUser user,
    required OwnProfile profile,
    required UserSettings settings,
  }) = _Me;

  factory Me.fromJson(Map<String, dynamic> json) => _$MeFromJson(json);
}

/// The user part of [Me].
@freezed
abstract class MeUser with _$MeUser {
  const factory MeUser({required String id, required Tier tier}) = _MeUser;

  factory MeUser.fromJson(Map<String, dynamic> json) => _$MeUserFromJson(json);
}

/// The caller's own profile.
@freezed
abstract class OwnProfile with _$OwnProfile {
  const factory OwnProfile({
    required String username,
    required String displayName,
    required String avatarId,
    required String locale,
    String? countryCode,

    /// null: the username may be changed now (full accounts only).
    DateTime? usernameChangeAvailableAt,
    DateTime? countryChangeAvailableAt,
  }) = _OwnProfile;

  factory OwnProfile.fromJson(Map<String, dynamic> json) =>
      _$OwnProfileFromJson(json);
}

/// Who may send challenges (`settings.challenges_from`).
enum ChallengesFrom { everyone, friends, nobody }

/// Privacy and app settings.
@freezed
abstract class UserSettings with _$UserSettings {
  const factory UserSettings({
    required ChallengesFrom challengesFrom,
    required bool showOnlineStatus,
    required bool allowFriendRequests,
    required bool soundEnabled,
    required bool musicEnabled,
    required bool hapticsEnabled,
    // D-101: only friends see the stats.
    @Default(false) bool profilePrivate,
  }) = _UserSettings;

  factory UserSettings.fromJson(Map<String, dynamic> json) =>
      _$UserSettingsFromJson(json);
}
