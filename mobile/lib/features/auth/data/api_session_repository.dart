import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/session.dart';
import '../domain/session_repository.dart';

final _country = RegExp(r'^[A-Z]{2}$');

/// [SessionRepository] over the REST API.
class ApiSessionRepository implements SessionRepository {
  ApiSessionRepository({
    required this.api,
    required this.installationId,
    required this.platform,
    required this.appVersion,
    required this.locale,
    required this.countryHint,
  });

  final ApiClient api;
  final String installationId;
  final String platform;
  final String appVersion;

  /// The app's current language (`tr` or `en`).
  final String Function() locale;

  /// The device region (ISO 3166 alpha-2), a hint for the profile country.
  final String? Function() countryHint;

  @override
  Future<Session> sync() async {
    // Regions like "419" (Latin America) are not countries; leave them out.
    final region = countryHint()?.toUpperCase();
    final country = region != null && _country.hasMatch(region) ? region : null;
    final json = await api.postJson(
      '/v1/session',
      body: {
        // The age gate (D-020) is shown before any sign-in, so every
        // session call carries the confirmation.
        'age_confirmed': true,
        'installation_id': installationId,
        'platform': platform,
        'app_version': appVersion,
        'locale': locale(),
        'country_hint': ?country,
      },
    );
    return Session.fromJson(json!);
  }

  @override
  Future<void> registerDevice() async {
    await api.postJson(
      '/v1/me/devices',
      body: {
        'installation_id': installationId,
        'platform': platform,
        'app_version': appVersion,
        'locale': locale(),
        'fcm_token': null, // push arrives in Phase 9
      },
    );
  }

  @override
  Future<void> unregisterDevice() async {
    await api.delete('/v1/me/devices/$installationId');
  }

  @override
  Future<void> abandonGuest({required String guestToken}) async {
    await api.postJson(
      '/v1/session/abandon-guest',
      idempotencyKey: const Uuid().v4(),
      idToken: guestToken,
    );
  }

  @override
  Future<DateTime> requestDeletion() async {
    final json = await api.postJson(
      '/v1/me/deletion',
      idempotencyKey: const Uuid().v4(),
    );
    return DateTime.parse(json!['purge_after'] as String);
  }

  @override
  Future<void> cancelDeletion() async {
    await api.delete('/v1/me/deletion');
  }
}
