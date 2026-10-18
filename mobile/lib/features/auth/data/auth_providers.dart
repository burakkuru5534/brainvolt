import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/providers.dart';
import '../../../core/l10n/locale_controller.dart';
import '../../../core/network/api_client.dart';
import '../../../core/network/token_source.dart';
import '../../../core/storage/installation_id.dart';
import '../../../firebase_options.dart';
import '../domain/auth_repository.dart';
import '../domain/session_repository.dart';
import 'api_session_repository.dart';
import 'firebase_auth_repository.dart';

part 'auth_providers.g.dart';

/// Firebase Auth. Tests override it with a fake.
@Riverpod(keepAlive: true)
AuthRepository authRepository(Ref ref) => FirebaseAuthRepository(
  googleClientId: defaultTargetPlatform == TargetPlatform.iOS
      ? DefaultFirebaseOptions.ios.iosClientId
      : null,
);

/// The identity endpoints.
@Riverpod(keepAlive: true)
SessionRepository sessionRepository(Ref ref) => ApiSessionRepository(
  api: ref.watch(apiClientProvider),
  installationId: ref.watch(installationIdProvider),
  platform: ref.watch(devicePlatformProvider),
  appVersion: ref.watch(appVersionProvider),
  locale: () => effectiveLocale(ref.read(localeOverrideProvider)).languageCode,
  countryHint: () =>
      WidgetsBinding.instance.platformDispatcher.locale.countryCode,
);

/// ID tokens of the signed-in Firebase user; `main.dart` installs it as the
/// API client's [TokenSource].
class AuthTokenSource implements TokenSource {
  const AuthTokenSource(this._auth);

  final AuthRepository _auth;

  @override
  Future<String?> idToken({bool forceRefresh = false}) =>
      _auth.idToken(forceRefresh: forceRefresh);
}
