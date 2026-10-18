import 'session.dart';

/// The identity endpoints of the backend (rest.md, `identity` module).
abstract interface class SessionRepository {
  /// `POST /v1/session`: creates the backend user on the first call and
  /// syncs tier and providers after sign-in, link or email verification.
  Future<Session> sync();

  /// `POST /v1/me/devices`: registers this installation (push token from
  /// Phase 9).
  Future<void> registerDevice();

  /// `DELETE /v1/me/devices/{installation_id}`: before signing out.
  Future<void> unregisterDevice();

  /// `POST /v1/session/abandon-guest` with the guest's own [guestToken]
  /// (link conflict, D-015).
  Future<void> abandonGuest({required String guestToken});

  /// `POST /v1/me/deletion`: starts the 14-day cooling-off (D-022) and
  /// returns when the account will be purged.
  Future<DateTime> requestDeletion();

  /// `DELETE /v1/me/deletion`.
  Future<void> cancelDeletion();
}
