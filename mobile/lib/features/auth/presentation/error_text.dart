import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../domain/auth_repository.dart';

/// Text for a failed sign-in or API call; null when there is nothing to say
/// (the user closed the sign-in UI).
String? errorText(AppLocalizations l, Object error) => switch (error) {
  AuthFailure(kind: AuthFailureKind.cancelled) => null,
  AuthFailure(:final kind) => switch (kind) {
    AuthFailureKind.invalidCredential => l.authErrorInvalidCredential,
    AuthFailureKind.weakPassword => l.authErrorWeakPassword,
    AuthFailureKind.invalidEmail => l.authErrorInvalidEmail,
    AuthFailureKind.userNotFound => l.authErrorUserNotFound,
    AuthFailureKind.tooManyRequests => l.authErrorTooManyRequests,
    AuthFailureKind.unsupported => l.authErrorUnsupported,
    AuthFailureKind.network => l.errorNetwork,
    AuthFailureKind.credentialAlreadyInUse ||
    AuthFailureKind.cancelled ||
    AuthFailureKind.unknown => l.errorUnknown,
  },
  ApiException() => error.localizedMessage(l),
  _ => l.errorUnknown,
};
