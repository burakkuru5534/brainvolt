// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'session.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Session _$SessionFromJson(Map<String, dynamic> json) => _Session(
  user: SessionUser.fromJson(json['user'] as Map<String, dynamic>),
  profile: ProfileSummary.fromJson(json['profile'] as Map<String, dynamic>),
  tier: $enumDecode(_$TierEnumMap, json['tier']),
  status: $enumDecode(_$AccountStatusEnumMap, json['status']),
  flags: SessionFlags.fromJson(json['flags'] as Map<String, dynamic>),
);

Map<String, dynamic> _$SessionToJson(_Session instance) => <String, dynamic>{
  'user': instance.user.toJson(),
  'profile': instance.profile.toJson(),
  'tier': _$TierEnumMap[instance.tier]!,
  'status': _$AccountStatusEnumMap[instance.status]!,
  'flags': instance.flags.toJson(),
};

const _$TierEnumMap = {
  Tier.guest: 'guest',
  Tier.unverified: 'unverified',
  Tier.full: 'full',
};

const _$AccountStatusEnumMap = {
  AccountStatus.active: 'active',
  AccountStatus.suspended: 'suspended',
  AccountStatus.deleted: 'deleted',
};

_SessionUser _$SessionUserFromJson(Map<String, dynamic> json) => _SessionUser(
  id: json['id'] as String,
  roles:
      (json['roles'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const <String>[],
  deletionRequestedAt: json['deletion_requested_at'] == null
      ? null
      : DateTime.parse(json['deletion_requested_at'] as String),
);

Map<String, dynamic> _$SessionUserToJson(_SessionUser instance) =>
    <String, dynamic>{
      'id': instance.id,
      'roles': instance.roles,
      'deletion_requested_at': instance.deletionRequestedAt?.toIso8601String(),
    };

_ProfileSummary _$ProfileSummaryFromJson(Map<String, dynamic> json) =>
    _ProfileSummary(
      username: json['username'] as String,
      displayName: json['display_name'] as String,
      avatarId: json['avatar_id'] as String,
      locale: json['locale'] as String,
      countryCode: json['country_code'] as String?,
    );

Map<String, dynamic> _$ProfileSummaryToJson(_ProfileSummary instance) =>
    <String, dynamic>{
      'username': instance.username,
      'display_name': instance.displayName,
      'avatar_id': instance.avatarId,
      'locale': instance.locale,
      'country_code': instance.countryCode,
    };

_SessionFlags _$SessionFlagsFromJson(Map<String, dynamic> json) =>
    _SessionFlags(
      created: json['created'] as bool,
      restricted: json['restricted'] as bool,
      emailVerificationRequired: json['email_verification_required'] as bool,
      deletionPending: json['deletion_pending'] as bool,
      suspended: json['suspended'] as bool,
    );

Map<String, dynamic> _$SessionFlagsToJson(_SessionFlags instance) =>
    <String, dynamic>{
      'created': instance.created,
      'restricted': instance.restricted,
      'email_verification_required': instance.emailVerificationRequired,
      'deletion_pending': instance.deletionPending,
      'suspended': instance.suspended,
    };
