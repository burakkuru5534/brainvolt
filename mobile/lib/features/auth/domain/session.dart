import 'package:freezed_annotation/freezed_annotation.dart';

part 'session.freezed.dart';
part 'session.g.dart';

/// Account tier (D-016, D-017): guests and unverified email accounts share
/// the restricted limits.
enum Tier { guest, unverified, full }

/// Cooling-off between a deletion request and the purge (D-022).
const deletionGracePeriod = Duration(days: 14);

/// Account status.
enum AccountStatus { active, suspended, deleted }

/// The answer of `POST /v1/session` (rest.md).
@freezed
abstract class Session with _$Session {
  const factory Session({
    required SessionUser user,
    required ProfileSummary profile,
    required Tier tier,
    required AccountStatus status,
    required SessionFlags flags,
  }) = _Session;

  factory Session.fromJson(Map<String, dynamic> json) =>
      _$SessionFromJson(json);
}

/// The user part of the session.
@freezed
abstract class SessionUser with _$SessionUser {
  const factory SessionUser({
    required String id,
    @Default(<String>[]) List<String> roles,
    DateTime? deletionRequestedAt,
  }) = _SessionUser;

  factory SessionUser.fromJson(Map<String, dynamic> json) =>
      _$SessionUserFromJson(json);
}

/// The profile part of the session.
@freezed
abstract class ProfileSummary with _$ProfileSummary {
  const factory ProfileSummary({
    required String username,
    required String displayName,
    required String avatarId,
    required String locale,
    String? countryCode,
  }) = _ProfileSummary;

  factory ProfileSummary.fromJson(Map<String, dynamic> json) =>
      _$ProfileSummaryFromJson(json);
}

/// What the app shows next (D-059).
@freezed
abstract class SessionFlags with _$SessionFlags {
  const factory SessionFlags({
    required bool created,
    required bool restricted,
    required bool emailVerificationRequired,
    required bool deletionPending,
    required bool suspended,
  }) = _SessionFlags;

  factory SessionFlags.fromJson(Map<String, dynamic> json) =>
      _$SessionFlagsFromJson(json);
}
