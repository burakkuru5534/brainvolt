// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'me.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Me _$MeFromJson(Map<String, dynamic> json) => _Me(
  user: MeUser.fromJson(json['user'] as Map<String, dynamic>),
  profile: OwnProfile.fromJson(json['profile'] as Map<String, dynamic>),
  settings: UserSettings.fromJson(json['settings'] as Map<String, dynamic>),
);

Map<String, dynamic> _$MeToJson(_Me instance) => <String, dynamic>{
  'user': instance.user.toJson(),
  'profile': instance.profile.toJson(),
  'settings': instance.settings.toJson(),
};

_MeUser _$MeUserFromJson(Map<String, dynamic> json) => _MeUser(
  id: json['id'] as String,
  tier: $enumDecode(_$TierEnumMap, json['tier']),
);

Map<String, dynamic> _$MeUserToJson(_MeUser instance) => <String, dynamic>{
  'id': instance.id,
  'tier': _$TierEnumMap[instance.tier]!,
};

const _$TierEnumMap = {
  Tier.guest: 'guest',
  Tier.unverified: 'unverified',
  Tier.full: 'full',
};

_OwnProfile _$OwnProfileFromJson(Map<String, dynamic> json) => _OwnProfile(
  username: json['username'] as String,
  displayName: json['display_name'] as String,
  avatarId: json['avatar_id'] as String,
  locale: json['locale'] as String,
  countryCode: json['country_code'] as String?,
  usernameChangeAvailableAt: json['username_change_available_at'] == null
      ? null
      : DateTime.parse(json['username_change_available_at'] as String),
  countryChangeAvailableAt: json['country_change_available_at'] == null
      ? null
      : DateTime.parse(json['country_change_available_at'] as String),
);

Map<String, dynamic> _$OwnProfileToJson(_OwnProfile instance) =>
    <String, dynamic>{
      'username': instance.username,
      'display_name': instance.displayName,
      'avatar_id': instance.avatarId,
      'locale': instance.locale,
      'country_code': instance.countryCode,
      'username_change_available_at': instance.usernameChangeAvailableAt
          ?.toIso8601String(),
      'country_change_available_at': instance.countryChangeAvailableAt
          ?.toIso8601String(),
    };

_UserSettings _$UserSettingsFromJson(Map<String, dynamic> json) =>
    _UserSettings(
      challengesFrom: $enumDecode(
        _$ChallengesFromEnumMap,
        json['challenges_from'],
      ),
      showOnlineStatus: json['show_online_status'] as bool,
      allowFriendRequests: json['allow_friend_requests'] as bool,
      soundEnabled: json['sound_enabled'] as bool,
      musicEnabled: json['music_enabled'] as bool,
      hapticsEnabled: json['haptics_enabled'] as bool,
      profilePrivate: json['profile_private'] as bool? ?? false,
    );

Map<String, dynamic> _$UserSettingsToJson(_UserSettings instance) =>
    <String, dynamic>{
      'challenges_from': _$ChallengesFromEnumMap[instance.challengesFrom]!,
      'show_online_status': instance.showOnlineStatus,
      'allow_friend_requests': instance.allowFriendRequests,
      'sound_enabled': instance.soundEnabled,
      'music_enabled': instance.musicEnabled,
      'haptics_enabled': instance.hapticsEnabled,
      'profile_private': instance.profilePrivate,
    };

const _$ChallengesFromEnumMap = {
  ChallengesFrom.everyone: 'everyone',
  ChallengesFrom.friends: 'friends',
  ChallengesFrom.nobody: 'nobody',
};
